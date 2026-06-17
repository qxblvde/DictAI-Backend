FROM python:3.10-slim

RUN apt-get update && apt-get install -y \
    ffmpeg \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --upgrade pip && \
    pip install -r requirements.txt

COPY app.py .

EXPOSE 8000

ENV PYTHONWARNINGS="ignore"

CMD ["python", "app.py"]