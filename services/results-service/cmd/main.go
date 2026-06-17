package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Microservices/services/results-service/internal/config"
	"github.com/Microservices/services/results-service/internal/infrastructure/persistence"
	"github.com/Microservices/services/results-service/internal/infrastructure/storage"
	"github.com/Microservices/services/results-service/internal/presentation"
	"github.com/Microservices/services/results-service/internal/service"
)

func main() {
	log := newLogger()
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Error("postgres ping failed", "error", err)
		os.Exit(1)
	}

	minio, err := storage.New(ctx, storage.Config{
		Endpoint:  cfg.MinIOEndpoint,
		AccessKey: cfg.MinIOAccess,
		SecretKey: cfg.MinIOSecret,
		Bucket:    cfg.MinIOBucket,
		UseSSL:    cfg.MinIOUseSSL,
	})
	if err != nil {
		log.Error("failed to connect to minio", "error", err)
		os.Exit(1)
	}

	audioStore, err := storage.New(ctx, storage.Config{
		Endpoint:  cfg.MinIOEndpoint,
		AccessKey: cfg.MinIOAccess,
		SecretKey: cfg.MinIOSecret,
		Bucket:    cfg.MinIOAudioBucket,
		UseSSL:    cfg.MinIOUseSSL,
	})
	if err != nil {
		log.Error("failed to connect to audio minio bucket", "error", err)
		os.Exit(1)
	}

	repo := persistence.NewResultRepository(db)
	baseURL := "http://results-service:" + cfg.Port
	svc := service.New(repo, minio, audioStore, baseURL)
	router := presentation.NewRouter(svc, log)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Info("results-service started", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("results-service stopped")
}

func newLogger() *slog.Logger {
	level := slog.LevelInfo
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "text" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	return slog.New(handler).With("service", "results-service")
}
