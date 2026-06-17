import os
import tempfile
import uuid
import logging
import gc
import warnings
import shutil
from typing import Optional

warnings.filterwarnings("ignore", module="matplotlib")
warnings.filterwarnings("ignore", module="pyannote")

os.environ['MPLBACKEND'] = 'Agg'

import whisperx
from fastapi import FastAPI, UploadFile, File, HTTPException, Query, Request
from fastapi.responses import JSONResponse
from starlette.requests import ClientDisconnect
import uvicorn
import aiofiles
import asyncio

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class _HealthCheckFilter(logging.Filter):
    def filter(self, record: logging.LogRecord) -> bool:
        return "GET /health" not in record.getMessage()

logging.getLogger("uvicorn.access").addFilter(_HealthCheckFilter())

MODEL_NAME = os.environ.get("WHISPERX_MODEL", "base")
DEVICE = os.environ.get("WHISPERX_DEVICE", "cpu")
COMPUTE_TYPE = os.environ.get("WHISPERX_COMPUTE_TYPE", "float32")
BATCH_SIZE = int(os.environ.get("WHISPERX_BATCH_SIZE", "8"))
HOST = os.environ.get("WHISPERX_HOST", "0.0.0.0")
PORT = int(os.environ.get("WHISPERX_PORT", "8000"))
MAX_FILE_SIZE = int(os.environ.get("WHISPERX_MAX_FILE_SIZE", 1024 * 1024 * 100))
CHUNK_SIZE = int(os.environ.get("WHISPERX_CHUNK_SIZE", 1024 * 1024))
DEFAULT_LANGUAGE = os.environ.get("WHISPERX_LANGUAGE", None) or None  # e.g. "ru", "en"

app = FastAPI()

whisper_model = None
align_models_cache = {}
model_loading = False
model_load_error = None

async def _load_model_background():
    global whisper_model, model_loading, model_load_error
    if whisper_model is not None or model_loading:
        return

    model_loading = True
    model_load_error = None
    logger.info(f"Загрузка модели WhisperX '{MODEL_NAME}' на устройстве {DEVICE}...")
    try:
        loop = asyncio.get_running_loop()
        # asr_options tuned for quality on CPU:
        # - beam_size=5 (default) gives best accuracy
        # - condition_on_previous_text=True helps maintain context between segments
        # - no_speech_threshold=0.6 reduces hallucinations on silence
        # - compression_ratio_threshold=2.4 filters out repetitive garbage outputs
        asr_options = {
            "beam_size": 5,
            "best_of": 5,
            "patience": 1.0,
            "length_penalty": 1.0,
            "repetition_penalty": 1.0,
            "no_repeat_ngram_size": 0,
            "temperatures": [0.0, 0.2, 0.4, 0.6, 0.8, 1.0],
            "compression_ratio_threshold": 2.4,
            "log_prob_threshold": -1.0,
            "no_speech_threshold": 0.6,
            "condition_on_previous_text": True,
            "prompt_reset_on_temperature": 0.5,
            "suppress_tokens": [-1],
            "suppress_blank": True,
            "without_timestamps": False,
            "max_initial_timestamp": 1.0,
            "word_timestamps": False,
            "prepend_punctuations": "\"'",
            "append_punctuations": "\"'.,:!?",
            "multilingual": False,
            "hallucination_silence_threshold": 0.2,
        }
        whisper_model = await loop.run_in_executor(
            None,
            lambda: whisperx.load_model(
                MODEL_NAME,
                DEVICE,
                compute_type=COMPUTE_TYPE,
                language=DEFAULT_LANGUAGE,
                asr_options=asr_options,
            ),
        )
        logger.info("Модель успешно загружена")
    except Exception as e:
        model_load_error = str(e)
        logger.error(f"Ошибка загрузки модели: {e}")
        whisper_model = None
    finally:
        model_loading = False


@app.on_event("startup")
async def startup_event():
    asyncio.create_task(_load_model_background())

@app.on_event("shutdown")
def cleanup():
    global whisper_model, align_models_cache
    logger.info("Очистка ресурсов...")
    whisper_model = None
    align_models_cache.clear()
    if DEVICE == "cuda":
        import torch
        torch.cuda.empty_cache()
    gc.collect()

def get_align_model(language_code):
    global align_models_cache

    if language_code in align_models_cache:
        return align_models_cache[language_code]

    logger.info(f"Загрузка модели выравнивания для языка {language_code}...")
    try:
        model_a, metadata = whisperx.load_align_model(
            language_code=language_code,
            device=DEVICE
        )
        align_models_cache[language_code] = (model_a, metadata)
        return model_a, metadata
    except Exception as e:
        logger.error(f"Ошибка загрузки модели выравнивания: {e}")
        return None, None

async def stream_to_file(file: UploadFile, temp_path: str) -> int:
    total_size = 0
    async with aiofiles.open(temp_path, 'wb') as f:
        while True:
            chunk = await file.read(CHUNK_SIZE)
            if not chunk:
                break
            total_size += len(chunk)
            if total_size > MAX_FILE_SIZE:
                raise HTTPException(status_code=413, detail=f"File too large. Max size: {MAX_FILE_SIZE} bytes")
            await f.write(chunk)
    return total_size


async def stream_request_body_to_file(request: Request, temp_path: str) -> int:
    total_size = 0
    try:
        async with aiofiles.open(temp_path, 'wb') as f:
            async for chunk in request.stream():
                if not chunk:
                    break
                total_size += len(chunk)
                if total_size > MAX_FILE_SIZE:
                    raise HTTPException(status_code=413, detail=f"File too large. Max size: {MAX_FILE_SIZE} bytes")
                await f.write(chunk)
    except ClientDisconnect:
        logger.warning("Клиент отключился во время загрузки файла")
        raise HTTPException(status_code=499, detail="Client disconnected while uploading")
    return total_size

def clean_words(segments):
    for segment in segments:
        if "words" in segment:
            valid_words = []
            for word in segment["words"]:
                word_text = word.get("text", "").strip()
                if word_text and word_text not in ["", " ", ".", ",", "!", "?"]:
                    valid_words.append({
                        "text": word_text,
                        "start": round(word.get("start", 0), 2),
                        "end": round(word.get("end", 0), 2)
                    })
            if valid_words:
                segment["words"] = valid_words
            else:
                del segment["words"]
    return segments

@app.get("/")
async def root():
    return {
        "service": "WhisperX Transcription",
        "status": "running",
        "model": MODEL_NAME,
        "device": DEVICE,
        "max_file_size_mb": MAX_FILE_SIZE // (1024 * 1024)
    }

@app.get("/health")
async def health():
    if model_loading:
        return JSONResponse(
            {"status": "loading", "model_loaded": False},
            status_code=202
        )
    if whisper_model is None:
        return JSONResponse(
            {"status": "unhealthy", "model_loaded": False, "error": model_load_error},
            status_code=503
        )
    return {"status": "healthy", "model_loaded": True}

@app.post("/transcribe")
async def transcribe(
        request: Request,
        file: UploadFile = File(None),
        language: Optional[str] = Query(None),
        batch_size: Optional[int] = Query(None),
        skip_align: bool = Query(False)
):
    if whisper_model is None:
        raise HTTPException(status_code=503, detail="Модель не загружена")
    content_type = request.headers.get("content-type", "").lower()
    is_octet = content_type.startswith("application/octet-stream")

    logger.info(f"Получен файл: {getattr(file, 'filename', 'raw_body')}")

    temp_dir = tempfile.mkdtemp(prefix="whisperx_")
    if is_octet:
        temp_path = os.path.join(temp_dir, f"{uuid.uuid4()}.raw")
    else:
        temp_path = os.path.join(temp_dir, f"{uuid.uuid4()}_{getattr(file, 'filename', 'upload')}")

    try:
        if is_octet:
            file_size = await stream_request_body_to_file(request, temp_path)
        else:
            if file is None:
                raise HTTPException(status_code=400, detail="No file provided")
            file_size = await stream_to_file(file, temp_path)
        logger.info(f"Файл сохранен, размер: {file_size / (1024 * 1024):.2f} MB")

        if await request.is_disconnected():
            logger.warning("Клиент отключился")
            raise HTTPException(status_code=499, detail="Client disconnected")

        audio = whisperx.load_audio(temp_path)

        bs = batch_size or BATCH_SIZE
        lang = language or DEFAULT_LANGUAGE
        logger.info(f"Транскрибация (batch_size={bs}, language={lang or 'auto'})...")

        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(
            None,
            lambda: whisper_model.transcribe(audio, batch_size=bs, language=lang)
        )

        if not skip_align and result.get("segments"):
            lang = result.get("language")
            if lang:
                model_a, metadata = get_align_model(lang)
                if model_a and metadata:
                    try:
                        logger.info("Выполнение выравнивания...")
                        result = await loop.run_in_executor(
                            None,
                            lambda: whisperx.align(
                                result["segments"],
                                model_a,
                                metadata,
                                audio,
                                DEVICE,
                                return_char_alignments=True
                            )
                        )
                    except Exception as e:
                        logger.error(f"Ошибка выравнивания: {e}")

        if DEVICE == "cuda":
            import torch
            torch.cuda.empty_cache()

        segments = []
        full_text_parts = []

        for segment in result.get("segments", []):
            text = segment.get("text", "").strip()
            if text:
                segment_data = {
                    "start": round(segment.get("start", 0), 2),
                    "end": round(segment.get("end", 0), 2),
                    "text": text
                }

                if "words" in segment:
                    words = []
                    for word in segment["words"]:
                        word_text = word.get("text", "").strip()
                        if word_text:
                            words.append({
                                "text": word_text,
                                "start": round(word.get("start", 0), 2),
                                "end": round(word.get("end", 0), 2)
                            })
                    if words:
                        segment_data["words"] = words

                segments.append(segment_data)
                full_text_parts.append(text)

        segments = clean_words(segments)

        response = {
            "success": True,
            "language": result.get("language", "unknown"),
            "segments": segments,
            "full_text": " ".join(full_text_parts),
            "segments_count": len(segments)
        }

        logger.info(f"Транскрибация завершена. Найдено {len(segments)} сегментов")
        return JSONResponse(response)

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Ошибка: {str(e)}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))

    finally:
        try:
            shutil.rmtree(temp_dir, ignore_errors=True)
        except Exception as e:
            logger.warning(f"Не удалось удалить временные файлы: {e}")

@app.get("/version")
async def version():
    return {
        "service": "whisperx-server",
        "model": MODEL_NAME,
        "device": DEVICE,
        "whisperx_version": getattr(whisperx, "__version__", "unknown"),
        "compute_type": COMPUTE_TYPE
    }

if __name__ == "__main__":
    print(f"Запуск WhisperX сервера на {HOST}:{PORT}")
    print(f"Модель: {MODEL_NAME}, устройство: {DEVICE}")
    uvicorn.run(app, host=HOST, port=PORT)