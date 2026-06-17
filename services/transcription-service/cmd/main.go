package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/Microservices/services/transcription-service/internal/application/service"
	audio_service "github.com/Microservices/services/transcription-service/internal/infrastructure/audio-service"
	"github.com/Microservices/services/transcription-service/internal/infrastructure/publisher"
	"github.com/Microservices/services/transcription-service/internal/infrastructure/transcriber"
	"github.com/Microservices/services/transcription-service/internal/logger"
	"github.com/Microservices/services/transcription-service/internal/presentation"
	"github.com/Microservices/services/transcription-service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log := logger.New("transcription-service")
	slog.SetDefault(log)

	log.Info("starting transcription-service")

	endpoint := os.Getenv("WHISPERX_ENDPOINT")
	if endpoint == "" {
		log.Error("WHISPERX_ENDPOINT not set")
		os.Exit(1)
	}
	t := transcriber.NewWhisper(endpoint)

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		log.Error("NATS_URL not set")
		os.Exit(1)
	}

	natsSubject := os.Getenv("NATS_SUBJECT")
	if natsSubject == "" {
		log.Error("NATS_SUBJECT not set")
		os.Exit(1)
	}

	p, err := publisher.NewNatsPublisher(natsURL, "audio.transcribed")
	if err != nil {
		log.Error("failed to create nats publisher", "error", err)
		os.Exit(1)
	}
	log.Info("connected to nats", "url", natsURL)

	ingestServiceGrpcUrl := os.Getenv("INGEST_SERVICE_GRPC_URL")
	if ingestServiceGrpcUrl == "" {
		log.Error("INGEST_SERVICE_GRPC_URL not set")
		os.Exit(1)
	}
	conn, err := grpc.NewClient(ingestServiceGrpcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("failed to create grpc connection", "addr", ingestServiceGrpcUrl, "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Warn("failed to close grpc connection", "error", err)
		}
	}()
	log.Info("connected to audio-ingest-service gRPC", "addr", ingestServiceGrpcUrl)

	client := proto.NewAudioIngestServiceClient(conn)
	audioService := audio_service.NewGrpcAudioService(client)
	transcriptionService := service.NewTranscriptionService(t, p, audioService)

	listener, err := presentation.NewListener(natsURL, natsSubject, transcriptionService)
	if err != nil {
		log.Error("failed to create nats listener", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	log.Info("listening for events", "subject", natsSubject)
	go listener.Listen(ctx)

	<-ctx.Done()
	log.Info("shutting down")
}
