package presentation

import (
	"log/slog"

	"audio-ingest-service/internal/presentation/handler"
	"audio-ingest-service/internal/presentation/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter(
	uploadHandler *handler.UploadHandler,
	anonUploadHandler *handler.AnonymousUploadHandler,
	anonStatusHandler *handler.AnonymousStatusHandler,
	anonFragmentHandler *handler.AnonymousFragmentHandler,
	anonConfirmHandler *handler.AnonymousConfirmHandler,
	jwtSecret string,
	log *slog.Logger,
) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery(log))
	r.Use(middleware.Logging(log))

	protected := r.Group("/ingest")
	//protected.Use(middleware.JWTMiddleware(jwtSecret))
	protected.POST("/audio/upload", uploadHandler.Upload)
	protected.POST("/audio/upload/anonymous", anonUploadHandler.Upload)
	protected.GET("/audio/anonymous/:session_id", anonStatusHandler.GetStatus)
	protected.GET("/audio/anonymous/:session_id/speakers/:speaker_id/fragment", anonFragmentHandler.GetFragment)
	protected.POST("/audio/anonymous/:session_id/confirm", anonConfirmHandler.Confirm)

	return r
}
