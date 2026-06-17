package application

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"gateway/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestProxy_ForwardsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/audio/test", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("backend response"))
		if err != nil {
			panic(err)
		}
	}))
	defer backend.Close()

	cfg := &config.Config{
		AudioURL: backend.URL,
	}

	router := gin.New()
	router.Any("/audio/*path", NewProxy(cfg, "audio", slog.Default()))

	gateway := httptest.NewServer(router)
	defer gateway.Close()

	resp, err := http.Get(gateway.URL + "/audio/test")
	assert.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("Error while closing resp.Body: %v", err)
		}
	}()

	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "backend response")
}
