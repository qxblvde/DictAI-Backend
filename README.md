# DictAI Backend

Бэкенд для автоматической расшифровки и анализа встреч.

## Что делает

Загружаешь аудио — на выходе получаешь транскрипт с разбивкой по спикерам и краткое резюме встречи.

## Сервисы

- **gateway** — точка входа, маршрутизация запросов
- **auth-service** — аутентификация
- **audio-ingest-service** — приём и хранение аудио
- **transcription-service** — транскрипция через WhisperX
- **speaker-recognition-service** — определение спикеров
- **transcript-builder-service** — сборка финального транскрипта
- **summarization-service** — суммаризация через LLM
- **results-service** — отдача результатов
- **workspace-service** — управление рабочими пространствами
- **voice-profile-service** — профили голосов
- **notification-service** — уведомления

## Стек

Go, Python (WhisperX), NATS, PostgreSQL, Docker

## Запуск

```bash
cd infra
cp .env.example .env
docker compose up
```
