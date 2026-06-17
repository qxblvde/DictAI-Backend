package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-gonic/gin"
)

var errorResponses = map[error]struct {
	status     int
	errMessage string
}{
	service.ErrInvalidWorkspaceName:          {http.StatusBadRequest, "invalid workspace name"},
	service.ErrWorkspaceNotFound:             {http.StatusNotFound, "workspace not found"},
	service.ErrCreateUserForbidden:           {http.StatusForbidden, "user not owner of workspace"},
	service.ErrDeleteParticipantForbidden:    {http.StatusForbidden, "only workspace owner can remove participants"},
	service.ErrUpdateParticipantForbidden:    {http.StatusForbidden, "only workspace owner can update participants"},
	service.ErrDeleteWorkspaceForbidden:      {http.StatusForbidden, "only workspace owner can delete workspace"},
	service.ErrUpdateWorkspaceForbidden:      {http.StatusForbidden, "only workspace owner can update workspace"},
	service.ErrParticipantNotFound:           {http.StatusNotFound, "participant not found in workspace"},
	service.ErrInvalidUserName:               {http.StatusBadRequest, "invalid user name"},
	service.ErrInvalidEmail:                  {http.StatusBadRequest, "invalid email"},
	service.ErrEmailNotUnique:                {http.StatusConflict, "email already exists in this workspace"},
	service.ErrViewParticipantsForbidden:     {http.StatusForbidden, "no rights to view participants"},
}

func handleError(ctx *gin.Context, err error) {
	for key, val := range errorResponses {
		if errors.Is(err, key) {
			ctx.AbortWithStatusJSON(val.status, gin.H{"error": val.errMessage})
			return
		}
	}
	slog.Error("unhandled error in workspace handler",
		"error", err,
		"path", ctx.Request.URL.Path,
		"request_id", ctx.GetHeader("X-Request-ID"),
	)
	ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
