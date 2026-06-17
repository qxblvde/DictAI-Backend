package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/Microservices/services/transcript-builder-service/internal/application"
	"github.com/Microservices/services/transcript-builder-service/internal/application/service"
	audio_service "github.com/Microservices/services/transcript-builder-service/internal/infrastructure/audio_service"
	"github.com/Microservices/services/transcript-builder-service/internal/infrastructure/diarization"
	"github.com/Microservices/services/transcript-builder-service/internal/infrastructure/publisher"
	infraStorage "github.com/Microservices/services/transcript-builder-service/internal/infrastructure/storage"
	"github.com/Microservices/services/transcript-builder-service/internal/logger"
	"github.com/Microservices/services/transcript-builder-service/internal/presentation"
	"github.com/Microservices/services/transcript-builder-service/proto"
	nats_go "github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log := logger.New("transcript-builder-service")
	slog.SetDefault(log)

	log.Info("starting transcript-builder-service")

	speakerRecognitionURL := requiredEnv("SPEAKER_RECOGNITION_URL")
	natsURL := requiredEnv("NATS_URL")
	listenSubject := requiredEnv("NATS_LISTEN_SUBJECT")
	publishSubject := requiredEnv("NATS_PUBLISH_SUBJECT")
	grpcAddress := requiredEnv("GRPC_ADDRESS")

	minioEndpoint := requiredEnv("MINIO_ENDPOINT")
	minioAccessKey := requiredEnv("MINIO_ACCESS_KEY")
	minioSecretKey := requiredEnv("MINIO_SECRET_KEY")
	minioBucket := requiredEnv("MINIO_BUCKET")

	diarizationService := diarization.NewHTTPDiarizationService(speakerRecognitionURL)
	builder := application.NewTranscriptionBuilderWithEmbedding(diarizationService, speakerRecognitionURL)

	minioStorage, err := infraStorage.NewMinIOStorage(context.Background(), infraStorage.MinIOConfig{
		Endpoint:  minioEndpoint,
		AccessKey: minioAccessKey,
		SecretKey: minioSecretKey,
		Bucket:    minioBucket,
		UseSSL:    os.Getenv("MINIO_USE_SSL") == "true",
	})
	if err != nil {
		log.Error("failed to init minio storage", "error", err)
		os.Exit(1)
	}
	log.Info("connected to minio", "endpoint", minioEndpoint)

	natsConn, err := connectNATS(natsURL)
	if err != nil {
		log.Error("failed to connect to nats", "error", err)
		os.Exit(1)
	}
	log.Info("connected to nats", "url", natsURL)

	pub := publisher.NewNatsPublisherWithStorage(natsConn, publishSubject, minioStorage)

	conn, err := grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("failed to create grpc connection", "addr", grpcAddress, "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Warn("failed to close grpc connection", "error", err)
		}
	}()
	log.Info("connected to audio-ingest-service gRPC", "addr", grpcAddress)

	client := proto.NewAudioIngestServiceClient(conn)
	audioSvc := audio_service.NewGrpcAudioService(client)
	svc := service.NewTranscriptBuilderService(builder, audioSvc, pub)

	listener, err := presentation.NewListener(natsURL, listenSubject, svc)
	if err != nil {
		log.Error("failed to create nats listener", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	log.Info("listening for events", "subject", listenSubject, "publishing_to", publishSubject)
	go listener.Listen(ctx)

	<-ctx.Done()
	log.Info("shutting down")
}

func requiredEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env var not set", "key", key)
		os.Exit(1)
	}
	return v
}

func connectNATS(url string) (*nats_go.Conn, error) {
	return nats_go.Connect(url)
}

