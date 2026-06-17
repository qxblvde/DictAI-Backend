package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/Microservices/services/auth-service/internal/data"
	"github.com/Microservices/services/auth-service/internal/logger"
	"github.com/Microservices/services/auth-service/internal/presentation"
	"github.com/Microservices/services/auth-service/internal/service"
	"github.com/gin-gonic/gin"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	log := logger.New("auth-service")
	slog.SetDefault(log)

	if os.Getenv("LOG_LEVEL") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	dbUser := os.Getenv("POSTGRES_USER")
	dbPass := os.Getenv("POSTGRES_PASSWORD")
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbName := os.Getenv("POSTGRES_DB")
	jwtSecret := os.Getenv("AUTH_SERVICE_JWT_SECRET")

	if dbUser == "" || dbPass == "" || dbHost == "" || dbPort == "" || dbName == "" || jwtSecret == "" {
		log.Error("missing required environment variables")
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

	userRepo := data.NewPostgresUserRepository(db)
	authService := service.NewAuthService(userRepo, jwtSecret)
	router := presentation.NewRouter(authService, log)

	log.Info("auth-service started", "port", "8080")
	if err := router.Run(":8080"); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
