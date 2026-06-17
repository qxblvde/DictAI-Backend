import asyncio
import io
import logging
import os
import subprocess
import tempfile
from typing import List, Optional

import numpy as np
import soundfile as sf
import torch
import requests
from fastapi import FastAPI, File, Form, HTTPException, UploadFile, Request
from fastapi.responses import JSONResponse

try:
    import grpc
except Exception:
    grpc = None

try:
    from proto import voice_profile_pb2, voice_profile_pb2_grpc
except Exception:
    try:
        import voice_profile_pb2
        import voice_profile_pb2_grpc
    except Exception:
        voice_profile_pb2 = None
        voice_profile_pb2_grpc = None

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class _HealthCheckFilter(logging.Filter):
    def filter(self, record: logging.LogRecord) -> bool:
        return "GET /docs" not in record.getMessage()


logging.getLogger("uvicorn.access").addFilter(_HealthCheckFilter())

HF_TOKEN = os.getenv("HUGGINGFACE_TOKEN", "")
WORKSPACE_SERVICE_URL = os.getenv("WORKSPACE_SERVICE_URL", "http://workspace-service:8080")
WORKSPACE_PARTICIPANTS_PATH = os.getenv("WORKSPACE_PARTICIPANTS_PATH", "/workspaces/{workspace_id}/participants")
VOICE_PROFILE_SERVICE_URL = os.getenv("VOICE_PROFILE_SERVICE_URL", "http://voice-profile-service:8080")
VOICE_PROFILE_GRPC_ADDR = os.getenv("VOICE_PROFILE_GRPC_ADDR", "")
VOICE_PROFILE_PATH = os.getenv("VOICE_PROFILE_PATH", "/voice-profiles/profiles/{participant_id}")
EMBED_THRESHOLD = float(os.getenv("EMBED_THRESHOLD", "0.35"))
SINGLE_PROFILE_EMBED_THRESHOLD = float(os.getenv("SINGLE_PROFILE_EMBED_THRESHOLD", "0.35"))
SHORT_SEGMENT_SECONDS = float(os.getenv("SHORT_SEGMENT_SECONDS", "5.0"))
SHORT_SEGMENT_EMBED_THRESHOLD = float(os.getenv("SHORT_SEGMENT_EMBED_THRESHOLD", "0.30"))
EMBED_MARGIN = float(os.getenv("EMBED_MARGIN", "0.08"))

app = FastAPI(title="Speaker Recognition Service")

embedding_inference: Optional[object] = None
model_loading: bool = False
model_load_error: Optional[str] = None


async def _load_model_background():
    global embedding_inference, model_loading, model_load_error
    model_loading = True
    model_load_error = None
    logger.info("Loading speechbrain ECAPA-TDNN model...")
    try:
        from speechbrain.pretrained import EncoderClassifier

        loop = asyncio.get_running_loop()

        def _load():
            return EncoderClassifier.from_hparams(
                source="speechbrain/spkrec-ecapa-voxceleb",
                savedir="/root/.cache/speechbrain/spkrec-ecapa-voxceleb",
                run_opts={"device": "cpu"},
            )

        embedding_inference = await loop.run_in_executor(None, _load)
        logger.info("speechbrain ECAPA-TDNN model loaded successfully")
    except Exception as e:
        model_load_error = str(e)
        logger.error(f"Failed to load speechbrain model: {e}")
    finally:
        model_loading = False


@app.on_event("startup")
async def startup_event():
    asyncio.create_task(_load_model_background())


@app.on_event("shutdown")
async def shutdown_event():
    pass


def _to_wav_bytes(audio_bytes: bytes) -> bytes:
    """Convert any audio format to WAV 16kHz mono using ffmpeg."""
    with tempfile.NamedTemporaryFile(suffix=".input", delete=False) as fin:
        fin.write(audio_bytes)
        fin_path = fin.name
    fout_path = fin_path + ".wav"
    try:
        subprocess.run(
            [
                "ffmpeg", "-y", "-i", fin_path,
                "-ar", "16000", "-ac", "1", "-f", "wav", fout_path,
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=True,
        )
        with open(fout_path, "rb") as f:
            return f.read()
    finally:
        try:
            os.unlink(fin_path)
        except Exception:
            pass
        try:
            os.unlink(fout_path)
        except Exception:
            pass


def _load_audio_signal(audio_bytes: bytes):
    # Convert to WAV first (handles m4a, mp3, ogg, etc.)
    try:
        wav_bytes = _to_wav_bytes(audio_bytes)
    except Exception:
        wav_bytes = audio_bytes  # fallback: try as-is

    bio = io.BytesIO(wav_bytes)
    arr, sr = sf.read(bio, dtype="float32")
    if arr.ndim > 1:
        arr = arr.mean(axis=1)
    duration = float(len(arr)) / float(sr) if sr else 0.0
    return torch.tensor(arr[None, :]), sr, duration


def _embedding_from_signal(signal, sr: int) -> np.ndarray:
    if embedding_inference is None:
        raise RuntimeError("embedding model not ready")

    import torchaudio

    if sr != 16000:
        signal = torchaudio.functional.resample(signal, sr, 16000)

    emb = embedding_inference.encode_batch(signal)  # (1, 1, 192)
    emb = emb.squeeze().numpy()

    norm = np.linalg.norm(emb)
    if norm > 0:
        emb = emb / norm

    return emb


def _compute_embedding(audio_bytes: bytes) -> np.ndarray:
    """192-d L2-normalized ECAPA-TDNN speaker embedding via speechbrain."""
    signal, sr, _ = _load_audio_signal(audio_bytes)
    return _embedding_from_signal(signal, sr)


def _compute_embedding_with_duration(audio_bytes: bytes):
    """Return speaker embedding and decoded audio duration in seconds."""
    signal, sr, duration = _load_audio_signal(audio_bytes)
    return _embedding_from_signal(signal, sr), duration


def cosine_similarity(a: np.ndarray, b: np.ndarray) -> float:
    na, nb = np.linalg.norm(a), np.linalg.norm(b)
    if na == 0 or nb == 0:
        return -1.0
    return float(np.dot(a, b) / (na * nb))


async def fetch_profile_embedding(participant_id: str) -> Optional[np.ndarray]:
    if not participant_id:
        return None

    if VOICE_PROFILE_GRPC_ADDR and voice_profile_pb2 and voice_profile_pb2_grpc and grpc is not None:
        try:
            async with grpc.aio.insecure_channel(VOICE_PROFILE_GRPC_ADDR) as ch:
                stub = voice_profile_pb2_grpc.VoiceProfileServiceStub(ch)
                req = voice_profile_pb2.GetProfileRequest(voice_profile_id=participant_id)
                resp = await stub.GetProfile(req, timeout=10)
                if resp and getattr(resp, "embeddings", None) and len(resp.embeddings) > 0:
                    return np.array(list(resp.embeddings), dtype=np.float64)
        except Exception:
            pass

    base = VOICE_PROFILE_SERVICE_URL.rstrip("/")
    candidate_paths = []
    try:
        candidate_paths.append(VOICE_PROFILE_PATH.format(participant_id=participant_id))
    except Exception:
        pass
    candidate_paths.extend([
        f"/voice-profiles/profiles/{participant_id}",
        f"/profiles/{participant_id}",
    ])

    for path in candidate_paths:
        url = f"{base}{path if path.startswith('/') else '/' + path}"
        try:
            resp = requests.get(url, timeout=10)
            if resp.status_code != 200:
                continue
            payload = resp.json()
            emb = payload.get("embedding") if isinstance(payload, dict) else None
            if isinstance(emb, list) and len(emb) > 0:
                return np.array(emb, dtype=np.float64)
        except Exception:
            continue

    return None


def fetch_workspace_participant_ids(workspace_id: str, user_id: Optional[str] = None) -> List[str]:
    if not workspace_id:
        return []

    base = WORKSPACE_SERVICE_URL.rstrip("/")
    path = WORKSPACE_PARTICIPANTS_PATH.format(workspace_id=workspace_id)
    url = f"{base}{path if path.startswith('/') else '/' + path}"

    headers = {}
    if user_id:
        headers["X-User-Id"] = user_id

    try:
        resp = requests.get(url, headers=headers, timeout=10)
        if resp.status_code != 200:
            return []
        payload = resp.json()
    except Exception:
        return []

    items = (
        payload if isinstance(payload, list)
        else payload.get("participants", []) if isinstance(payload, dict)
        else []
    )

    out: List[str] = []
    for item in items:
        if isinstance(item, str) and item:
            out.append(item)
        elif isinstance(item, dict):
            pid = item.get("participant_id") or item.get("id")
            if isinstance(pid, str) and pid:
                out.append(pid)
    return out


@app.post("/embedding")
async def embedding_endpoint(file: UploadFile = File(...)):
    if embedding_inference is None:
        detail = "model is loading, try again shortly" if model_loading else f"model unavailable: {model_load_error}"
        raise HTTPException(status_code=503, detail=detail)

    data = await file.read()
    if not data:
        raise HTTPException(status_code=400, detail="empty file")

    try:
        loop = asyncio.get_running_loop()
        emb = await loop.run_in_executor(None, _compute_embedding, data)
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"embedding failed: {exc}")

    return JSONResponse({"embedding": emb.tolist()})


@app.post("/identify-speaker")
async def identify_speaker_endpoint(
    request: Request,
    workspace_id: str = Form(...),
    file: UploadFile = File(...),
):
    if embedding_inference is None:
        detail = "model is loading, try again shortly" if model_loading else f"model unavailable: {model_load_error}"
        raise HTTPException(status_code=503, detail=detail)

    if not workspace_id.strip():
        raise HTTPException(status_code=400, detail="workspace_id required")

    data = await file.read()
    if not data:
        raise HTTPException(status_code=400, detail="empty file")

    try:
        loop = asyncio.get_running_loop()
        audio_embedding, audio_duration = await loop.run_in_executor(None, _compute_embedding_with_duration, data)
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"embedding failed: {exc}")

    user_id = request.headers.get("X-User-Id") or request.headers.get("x-user-id")
    participant_ids = fetch_workspace_participant_ids(workspace_id, user_id)
    if not participant_ids:
        return JSONResponse({
            "workspace_id": workspace_id,
            "participant_id": None,
            "similarity": None,
            "threshold": EMBED_THRESHOLD,
            "matched": False,
            "reason": "no_participants",
        })

    best_pid = None
    best_sim = -1.0
    second_best_sim = -1.0
    scored_profiles = 0
    for pid in participant_ids:
        try:
            emb = await fetch_profile_embedding(pid)
        except Exception:
            emb = None
        if emb is None or emb.size == 0:
            continue
        sim = cosine_similarity(audio_embedding, emb)
        scored_profiles += 1
        if sim > best_sim:
            second_best_sim = best_sim
            best_sim = sim
            best_pid = pid
        elif sim > second_best_sim:
            second_best_sim = sim

    accept_threshold = EMBED_THRESHOLD
    if scored_profiles == 1:
        accept_threshold = min(EMBED_THRESHOLD, SINGLE_PROFILE_EMBED_THRESHOLD)
        if 0 < audio_duration < SHORT_SEGMENT_SECONDS:
            accept_threshold = min(accept_threshold, SHORT_SEGMENT_EMBED_THRESHOLD)

    reason = "matched"
    if best_pid is None:
        reason = "no_valid_profiles"
    elif best_sim < accept_threshold:
        reason = "below_threshold"
    elif scored_profiles > 1 and second_best_sim >= 0 and (best_sim - second_best_sim) < EMBED_MARGIN:
        reason = "ambiguous_match"

    matched = reason == "matched"
    logger.info(
        "speaker decision: workspace=%s matched=%s reason=%s best_pid=%s best_sim=%.6f second_best=%.6f threshold=%.6f duration=%.2f profiles=%d",
        workspace_id, matched, reason, best_pid, best_sim, second_best_sim, accept_threshold, audio_duration, scored_profiles,
    )

    if not matched:
        return JSONResponse({
            "workspace_id": workspace_id,
            "participant_id": None,
            "similarity": None if best_sim < 0 else round(float(best_sim), 6),
            "second_similarity": None if second_best_sim < 0 else round(float(second_best_sim), 6),
            "threshold": EMBED_THRESHOLD,
            "accept_threshold": accept_threshold,
            "duration": round(audio_duration, 3),
            "matched": False,
            "reason": reason,
        })

    return JSONResponse({
        "workspace_id": workspace_id,
        "participant_id": best_pid,
        "similarity": round(float(best_sim), 6),
        "second_similarity": None if second_best_sim < 0 else round(float(second_best_sim), 6),
        "threshold": EMBED_THRESHOLD,
        "accept_threshold": accept_threshold,
        "duration": round(audio_duration, 3),
        "matched": True,
        "reason": reason,
    })


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host=os.getenv("APP_HOST", "0.0.0.0"), port=int(os.getenv("APP_PORT", "8000")))
