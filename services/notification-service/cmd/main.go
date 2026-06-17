package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/Microservices/services/notification-service/internal/application/service"
	"github.com/Microservices/services/notification-service/internal/infrastructure/auth"
	"github.com/Microservices/services/notification-service/internal/infrastructure/email"
	"github.com/Microservices/services/notification-service/internal/infrastructure/workspace"
	"github.com/Microservices/services/notification-service/internal/presentation"
)

func main() {
	setupLogging()
	slog.Info("starting notification-service")

	natsURL := requiredEnv("NATS_URL")
	summarizedSubject := requiredEnv("NATS_SUMMARIZED_SUBJECT")
	workspaceServiceURL := requiredEnv("WORKSPACE_SERVICE_URL")
	authServiceURL := requiredEnv("AUTH_SERVICE_URL")
	smtpHost := requiredEnv("SMTP_HOST")
	smtpPort := envOrDefault("SMTP_PORT", "587")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := requiredEnv("SMTP_FROM")

	sender := email.NewSMTPSender(smtpHost, smtpPort, smtpUser, smtpPass, smtpFrom)
	workspaceClient := workspace.NewHTTPClient(workspaceServiceURL)
	authClient := auth.NewHTTPClient(authServiceURL)
	svc := service.New(sender, workspaceClient, authClient)

	listener, err := presentation.NewListener(natsURL, summarizedSubject, svc)
	if err != nil {
		log.Fatal("failed to create listener: ", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go listener.Listen(ctx)

	slog.Info("notification-service ready", "subject", summarizedSubject)
	<-ctx.Done()
	slog.Info("shutting down notification-service")
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
