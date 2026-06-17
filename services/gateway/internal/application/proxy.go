package application

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"gateway/internal/config"

	"github.com/gin-gonic/gin"
)

func NewProxy(cfg *config.Config, service string, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		l := log.With("request_id", reqID, "target_service", service)

		targetStr, err := resolveServiceURL(cfg, service)
		if err != nil {
			l.Error("unknown proxy target", "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		targetURL, err := url.Parse(targetStr)
		if err != nil {
			l.Error("invalid target url", "url", targetStr, "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "invalid target url"})
			return
		}

		proxy := &httputil.ReverseProxy{
			Rewrite: func(preq *httputil.ProxyRequest) {
				preq.SetURL(targetURL)
				preq.Out.Header.Set("X-Request-ID", reqID)

				userID, exists := c.Get("user_id")
				if exists {
					preq.Out.Header.Set("X-User-Id", fmt.Sprintf("%v", userID))
					l.Debug("forwarding user id", "user_id", userID)
				} else {
					l.Debug("no user_id in context (public route)")
				}

				preq.SetXForwarded()
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				w.WriteHeader(http.StatusBadGateway)
				l.Error("proxy error", "error", err)
			},
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func resolveServiceURL(cfg *config.Config, service string) (string, error) {
	switch service {
	case "workspace":
		return cfg.WorkspaceURL, nil
	case "result":
		return cfg.ResultURL, nil
	case "audio":
		return cfg.AudioURL, nil
	case "voice":
		return cfg.VoiceURL, nil
	case "auth":
		return cfg.AuthURL, nil
	default:
		return "", fmt.Errorf("unknown service: %s", service)
	}
}
