package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/DictAI/Microservices/services/workspace-service/internal/logger"
	"github.com/DictAI/Microservices/services/workspace-service/internal/persistence"
	"github.com/DictAI/Microservices/services/workspace-service/internal/presentation"
	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	log := logger.New("workspace-service")
	slog.SetDefault(log)

	if os.Getenv("LOG_LEVEL") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	dbUser := os.Getenv("POSTGRES_USER")
	dbPass := os.Getenv("POSTGRES_PASSWORD")
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbName := os.Getenv("POSTGRES_DB")
	servicePort := os.Getenv("WORKSPACE_SERVICE_PORT")

	if dbUser == "" || dbPass == "" || dbHost == "" || dbPort == "" || dbName == "" {
		log.Error("missing required database environment variables")
		os.Exit(1)
	}
	if servicePort == "" {
		log.Error("WORKSPACE_SERVICE_PORT not set")
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

	workspaceRepo := persistence.NewPostgresWorkspaceRepository(db)
	participantRepo := persistence.NewPostgresParticipantRepository(db)
	workspaceService := service.NewWorkspaceService(workspaceRepo, participantRepo)

	router := presentation.NewRouter(workspaceService, log)

	log.Info("workspace-service started", "port", servicePort)
	if err := router.Run(":" + servicePort); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
