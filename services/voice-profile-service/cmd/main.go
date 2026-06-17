package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/Microservices/services/voice-profile-service/internal/application/service"
	"github.com/Microservices/services/voice-profile-service/internal/infrastructure/database"
	"github.com/Microservices/services/voice-profile-service/internal/infrastructure/py_embedding_service"
	"github.com/Microservices/services/voice-profile-service/internal/infrastructure/workspace"
	"github.com/Microservices/services/voice-profile-service/internal/logger"
	"github.com/Microservices/services/voice-profile-service/internal/presentation"
	"github.com/gin-gonic/gin"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	log := logger.New("voice-profile-service")
	slog.SetDefault(log)

	if os.Getenv("LOG_LEVEL") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	endpoint := normalizeEmbeddingEndpoint(log, os.Getenv("PY_SPEACH_TO_VECTOR_SERVICE_ENDPOINT"))
	embd := py_embedding_service.NewPyService(endpoint)

	dbUser := os.Getenv("POSTGRES_USER")
	dbPass := os.Getenv("POSTGRES_PASSWORD")
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbName := os.Getenv("POSTGRES_DB")
	if dbUser == "" || dbPass == "" || dbHost == "" || dbPort == "" || dbName == "" {
		log.Error("missing required database environment variables")
		os.Exit(1)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Error("failed to open db connection", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Warn("failed to close db connection", "error", err)
		}
	}()

	if err := db.Ping(); err != nil {
		log.Error("db ping failed", "host", dbHost, "port", dbPort, "error", err)
		os.Exit(1)
	}
	log.Info("connected to postgres", "host", dbHost, "port", dbPort, "db", dbName)

	repo := database.NewVoiceProfilePostgresRepository(db)

	workspaceServiceURL := os.Getenv("WORKSPACE_SERVICE_URL")
	if workspaceServiceURL == "" {
		workspaceServiceURL = "http://workspace-service:8080"
	}
	workspaceSvc := workspace.NewHTTPService(workspaceServiceURL)

	svc := service.NewVoiceProfileService(repo, embd, workspaceSvc)
	router := presentation.NewRouter(svc, log)

	port := os.Getenv("VOICE_PROFILE_SERVICE_PORT")
	if port == "" {
		port = os.Getenv("VOICE_PROFILE_PORT")
	}
	if port == "" {
		port = "8080"
	}

	log.Info("voice-profile-service started", "port", port)
	if err := router.Run(":" + port); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func normalizeEmbeddingEndpoint(log *slog.Logger, raw string) string {
	endpoint := strings.TrimSpace(raw)
	if endpoint == "" || endpoint == "..." {
		endpoint = "http://speaker-recognition-service:8000/embedding"
	}

	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		log.Error("invalid PY_SPEACH_TO_VECTOR_SERVICE_ENDPOINT", "value", raw, "error", err)
		os.Exit(1)
	}

	if u.Path == "" || u.Path == "/" {
		u.Path = "/embedding"
	}

	return u.String()
}
