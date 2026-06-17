package main

import (
	"log/slog"
	"net/http"
	"os"

	"gateway/internal/config"
	"gateway/internal/logger"
	"gateway/internal/presentation"

	"github.com/gin-gonic/gin"
)

func main() {
	log := logger.New("gateway")
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if os.Getenv("LOG_LEVEL") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := presentation.NewRouter(cfg, log)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	log.Info("gateway started", "port", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
