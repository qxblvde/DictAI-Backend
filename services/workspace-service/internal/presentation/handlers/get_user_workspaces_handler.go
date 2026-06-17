package handlers

import (
	"net/http"
	"strconv"

	"github.com/DictAI/Microservices/services/workspace-service/internal/service"
	"github.com/gin-gonic/gin"
)

func NewGetUserWorkspacesHandler(svc *service.WorkspaceService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.GetHeader("X-User-Id")
		if userID == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user unauthorized"})
			return
		}

		page := 1
		limit := 20

		if v := ctx.Query("page"); v != "" {
			p, err := strconv.Atoi(v)
			if err != nil || p < 1 {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "page must be a positive integer"})
				return
			}
			page = p
		}

		if v := ctx.Query("limit"); v != "" {
			l, err := strconv.Atoi(v)
			if err != nil || l < 1 {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
				return
			}
			limit = l
		}

		workspaces, total, totalPages, err := svc.GetWorkspacesByUser(userID, page, limit)
		if err != nil {
			handleError(ctx, err)
			return
		}

		type workspaceItem struct {
			WorkspaceID string `json:"workspace_id"`
			OwnerID     string `json:"owner_user_id"`
			Name        string `json:"name"`
			CreatedAt   string `json:"created_at"`
		}

		items := make([]workspaceItem, 0, len(workspaces))
		for _, w := range workspaces {
			items = append(items, workspaceItem{
				WorkspaceID: w.WorkspaceID,
				OwnerID:     w.OwnerID,
				Name:        w.Name,
				CreatedAt:   w.CreatedAt.Format("2006-01-02T15:04:05Z"),
			})
		}

		ctx.JSON(http.StatusOK, gin.H{
			"data": items,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": totalPages,
			},
		})
	}
}
