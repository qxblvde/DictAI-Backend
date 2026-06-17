package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"audio-ingest-service/internal/application"
	"audio-ingest-service/internal/config"
	infradb "audio-ingest-service/internal/infrastructure/db"
	"audio-ingest-service/internal/infrastructure/event"
	"audio-ingest-service/internal/infrastructure/storage"
	"audio-ingest-service/internal/infrastructure/voiceprofile"
	"audio-ingest-service/internal/infrastructure/workspace"
	"audio-ingest-service/internal/logger"
	"audio-ingest-service/internal/presentation"
	grpcserver "audio-ingest-service/internal/presentation/grpcserver"
	"audio-ingest-service/internal/presentation/handler"
	"audio-ingest-service/internal/presentation/listener"

	audiopb "audio-ingest-service/proto"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func main() {
	log := logger.New("audio-ingest-service")
	slog.SetDefault(log)

	if os.Getenv("LOG_LEVEL") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	cfg := config.Load()

	natsConn, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		log.Error("failed to connect to nats", "url", cfg.NATSURL, "error", err)
		os.Exit(1)
	}
	defer natsConn.Close()
	log.Info("connected to nats", "url", cfg.NATSURL)

	objectStorage, err := storage.NewMinIOStorage(context.Background(), storage.MinIOConfig{
		Endpoint:  cfg.MinIOEndpoint,
		AccessKey: cfg.MinIOAccessKey,
		SecretKey: cfg.MinIOSecretKey,
		Bucket:    cfg.MinIOBucket,
		UseSSL:    cfg.MinIOUseSSL,
	})
	if err != nil {
		log.Error("failed to init minio storage", "endpoint", cfg.MinIOEndpoint, "error", err)
		os.Exit(1)
	}
	log.Info("connected to minio", "endpoint", cfg.MinIOEndpoint, "bucket", cfg.MinIOBucket)

	db, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		log.Error("failed to open postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.PingContext(context.Background()); err != nil {
		log.Error("failed to ping postgres", "error", err)
		os.Exit(1)
	}
	log.Info("connected to postgres")

	sessionRepo := infradb.NewPostgresAnonymousSessionRepository(db)

	eventPublisher := event.NewNATSPublisher(natsConn)
	workspaceChecker := workspace.NewHTTPAccessChecker(cfg.WorkspaceServiceURL, &http.Client{Timeout: 5 * time.Second})
	workspaceClient := workspace.NewHTTPWorkspaceClient(cfg.WorkspaceServiceURL)
	voiceProfileClient := voiceprofile.NewHTTPVoiceProfileClient(cfg.VoiceProfileServiceURL)

	uploadService := application.NewUploadService(objectStorage, eventPublisher, workspaceChecker)
	anonUploadService := application.NewAnonymousUploadService(objectStorage, eventPublisher, sessionRepo)
	confirmService := application.NewConfirmService(sessionRepo, objectStorage, workspaceClient, voiceProfileClient, eventPublisher)

	grpcAudioServer := grpcserver.NewAudioIngestServiceServer(objectStorage, workspaceChecker)

	anonDiarizedListener, err := listener.NewAnonymousDiarizedListener(natsConn, sessionRepo)
	if err != nil {
		log.Error("failed to create anonymous_diarized listener", "error", err)
		os.Exit(1)
	}

	router := presentation.NewRouter(
		handler.NewUploadHandler(uploadService, log, cfg.ResultsServiceURL),
		handler.NewAnonymousUploadHandler(anonUploadService, log, cfg.ResultsServiceURL),
		handler.NewAnonymousStatusHandler(sessionRepo, log),
		handler.NewAnonymousFragmentHandler(sessionRepo, objectStorage, log),
		handler.NewAnonymousConfirmHandler(confirmService, log),
		cfg.JWTSecret,
		log,
	)

	netListener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Error("failed to listen on grpc port", "port", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer()
	audiopb.RegisterAudioIngestServiceServer(grpcSrv, grpcAudioServer)
	go func() {
		log.Info("audio-ingest-service gRPC started", "port", cfg.GRPCPort)
		if err := grpcSrv.Serve(netListener); err != nil {
			log.Error("gRPC server stopped", "error", err)
			os.Exit(1)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go anonDiarizedListener.Listen(ctx)
	go runCleanup(ctx, sessionRepo, objectStorage, log)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("audio-ingest-service started", "port", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("HTTP server stopped", "error", err)
		os.Exit(1)
	}
}

func runCleanup(ctx context.Context, repo interface {
	CleanupExpired(ctx context.Context) ([]string, error)
}, stor interface {
	Delete(ctx context.Context, key string) error
}, log *slog.Logger) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			keys, err := repo.CleanupExpired(ctx)
			if err != nil {
				log.Error("cleanup expired sessions failed", "error", err)
				continue
			}
			for _, key := range keys {
				if err := stor.Delete(ctx, key); err != nil {
					log.Warn("failed to delete expired object", "key", key, "error", err)
				}
			}
			if len(keys) > 0 {
				log.Info("cleaned up expired anonymous sessions", "deleted_objects", len(keys))
			}
		}
	}
}
