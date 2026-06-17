package presentation

import (
	"log/slog"

	"gateway/internal/application"
	"gateway/internal/config"
	"gateway/internal/presentation/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg *config.Config, log *slog.Logger) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery(log))
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.Logging(log))
	r.Use(middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimit))

	r.Any("/auth/login", application.NewProxy(cfg, "auth", log))
	r.Any("/auth/register", application.NewProxy(cfg, "auth", log))

	protected := r.Group("/")
	protected.Use(middleware.JWTMiddleware(cfg.JWTSecret, log))

	protected.Any("/auth/refresh", application.NewProxy(cfg, "auth", log))
	protected.Any("/auth/change-password", application.NewProxy(cfg, "auth", log))
	protected.Any("/auth/change-email", application.NewProxy(cfg, "auth", log))
	protected.Any("/workspaces/*path", application.NewProxy(cfg, "workspace", log))
	protected.Any("/meetings/*path", application.NewProxy(cfg, "result", log))
	protected.Any("/ingest/*path", application.NewProxy(cfg, "audio", log))
	protected.Any("/voice-profiles/*path", application.NewProxy(cfg, "voice", log))

	return r
}
