package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/Microservices/services/summarization-service/internal/application/service"
	"github.com/Microservices/services/summarization-service/internal/infrastructure/claude"
	"github.com/Microservices/services/summarization-service/internal/infrastructure/publisher"
	"github.com/Microservices/services/summarization-service/internal/infrastructure/results"
	"github.com/Microservices/services/summarization-service/internal/infrastructure/workspace"
	"github.com/Microservices/services/summarization-service/internal/presentation"
)

func main() {
	setupLogging()
	slog.Info("starting summarization-service")

	natsURL := requiredEnv("NATS_URL")
	diarizedSubject := requiredEnv("NATS_DIARIZED_SUBJECT")
	publishSubject := requiredEnv("SUMMARIZATION_PUBLISH_SUBJECT")
	resultsServiceURL := requiredEnv("RESULTS_SERVICE_URL")
	anthropicAPIKey := requiredEnv("ANTHROPIC_API_KEY")
	anthropicBaseURL := os.Getenv("ANTHROPIC_BASE_URL")
	anthropicModel := envOrDefault("ANTHROPIC_MODEL", "claude-opus-4-7")
	workspaceServiceURL := requiredEnv("WORKSPACE_SERVICE_URL")
	publicResultsURL := os.Getenv("PUBLIC_RESULTS_URL")

	summarizer := claude.NewClaudeSummarizer(anthropicAPIKey, anthropicBaseURL, anthropicModel)
	resultsClient := results.NewHTTPClient(resultsServiceURL)
	workspaceClient := workspace.NewHTTPClient(workspaceServiceURL)

	natsPublisher, err := publisher.NewNatsPublisher(natsURL, publishSubject)
	if err != nil {
		log.Fatal("failed to create nats publisher: ", err)
	}

	svc := service.New(summarizer, resultsClient, natsPublisher, workspaceClient, publicResultsURL, resultsServiceURL)

	listener, err := presentation.NewListener(natsURL, diarizedSubject, svc)
	if err != nil {
		log.Fatal("failed to create listener: ", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go listener.Listen(ctx)

	slog.Info("summarization-service ready", "subject", diarizedSubject)
	<-ctx.Done()
	slog.Info("shutting down summarization-service")
}

func setupLogging() {
	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}
	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(handler))
}

func requiredEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is not set", name)
	}
	return v
}

func envOrDefault(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
