package presentation

import (
	"log/slog"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/Microservices/services/results-service/internal/presentation/handlers"
	"github.com/Microservices/services/results-service/internal/presentation/middleware"
	"github.com/Microservices/services/results-service/internal/service"
)

func NewRouter(svc *service.ResultService, log *slog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(middleware.Recovery(log))
	router.Use(cors.Default())
	router.Use(middleware.Logging(log))

	h := handlers.NewResultHandler(svc)

	meetings := router.Group("/meetings")
	meetings.GET("/results", h.List)
	meetings.POST("/results", h.Upload)
	meetings.POST("/results/pending", h.CreatePending)
	meetings.PATCH("/results/:audio_id/status", h.SetStatus)
	meetings.GET("/results/:audio_id/summary", h.GetSummary)
	meetings.GET("/results/:audio_id/transcript", h.GetTranscript)
	meetings.GET("/results/:audio_id/audio", h.GetAudio)
	meetings.DELETE("/results/:audio_id", h.Delete)

	return router
}
