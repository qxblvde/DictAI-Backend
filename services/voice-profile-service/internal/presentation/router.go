package presentation

import (
	"log/slog"

	voiceprofile "github.com/Microservices/services/voice-profile-service/internal/application/contracts/voiceprofile"
	"github.com/Microservices/services/voice-profile-service/internal/presentation/handlers"
	"github.com/Microservices/services/voice-profile-service/internal/presentation/middleware"
	"github.com/gin-gonic/gin"
)

func NewRouter(svc voiceprofile.VoiceProfileService, log *slog.Logger) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery(log))
	r.Use(middleware.Logging(log))

	g := r.Group("/voice-profiles")

	g.GET("/profiles", handlers.NewListProfilesHandler(svc, log))
	g.POST("/profiles", handlers.NewUploadAudioHandler(svc, log))
	g.POST("/library", handlers.NewCreateLibraryProfileHandler(svc, log))
	g.POST("/profiles/:voice_profile_id/assign", handlers.NewAssignProfileHandler(svc, log))
	g.GET("/profiles/:participant_id", handlers.NewGetProfileHandler(svc, log))

	return r
}
