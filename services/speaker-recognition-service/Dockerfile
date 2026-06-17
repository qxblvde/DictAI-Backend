FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    ffmpeg \
    libsndfile1 \
    git \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt /app/requirements.txt
RUN pip install --upgrade pip && \
    pip install --no-cache-dir -r /app/requirements.txt

COPY proto ./proto
COPY app.py /app/app.py
COPY README.md /app/README.md

RUN python -m grpc_tools.protoc --proto_path=./proto --python_out=./proto --grpc_python_out=./proto ./proto/audio_ingest.proto || true
RUN python -m grpc_tools.protoc --proto_path=./proto --python_out=./proto --grpc_python_out=./proto ./proto/voice_profile.proto || true
RUN cp -f ./proto/audio_ingest_pb2.py ./audio_ingest_pb2.py && \
    cp -f ./proto/audio_ingest_pb2_grpc.py ./audio_ingest_pb2_grpc.py && \
    cp -f ./proto/voice_profile_pb2.py ./voice_profile_pb2.py && \
    cp -f ./proto/voice_profile_pb2_grpc.py ./voice_profile_pb2_grpc.py

EXPOSE 8000

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8000"]
