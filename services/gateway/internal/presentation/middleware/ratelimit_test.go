package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RateLimit(1, 1))

	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}
