package workspace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPAccessChecker_CanUploadSuccess(t *testing.T) {
	workspaceID := "workspace-1"
	userID := "user-1"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/workspaces/"+workspaceID+"/participants", r.URL.Path)
		assert.Equal(t, userID, r.Header.Get("X-User-ID"))
		assert.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPAccessChecker(server.URL+"/", server.Client())

	allowed, err := checker.CanUpload(context.Background(), workspaceID, userID)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestHTTPAccessChecker_DeniedStatuses(t *testing.T) {
	tests := []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound}

	for _, status := range tests {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()

			checker := NewHTTPAccessChecker(server.URL, server.Client())
			allowed, err := checker.CanUpload(context.Background(), "workspace-1", "user-1")
			require.NoError(t, err)
			assert.False(t, allowed)
		})
	}
}

func TestHTTPAccessChecker_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := NewHTTPAccessChecker(server.URL, server.Client())
	allowed, err := checker.CanUpload(context.Background(), "workspace-1", "user-1")

	require.Error(t, err)
	assert.False(t, allowed)
	assert.Contains(t, err.Error(), "workspace service status: 500")
}

func TestHTTPAccessChecker_EmptyBaseURL(t *testing.T) {
	checker := NewHTTPAccessChecker("", nil)

	allowed, err := checker.CanUpload(context.Background(), "workspace-1", "user-1")
	require.Error(t, err)
	assert.False(t, allowed)
	assert.Contains(t, err.Error(), "workspace checker is not configured")
}

func TestHTTPAccessChecker_SkipsCallWhenInputIncomplete(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPAccessChecker(server.URL, server.Client())

	allowed, err := checker.CanUpload(context.Background(), "", "user-1")
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = checker.CanUpload(context.Background(), "workspace-1", "")
	require.NoError(t, err)
	assert.False(t, allowed)

	assert.Equal(t, 0, callCount)
}
