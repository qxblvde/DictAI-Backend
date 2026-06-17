package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"audio-ingest-service/internal/application"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubStorage struct {
	putErr error
}

func (s *stubStorage) Put(_ context.Context, _ string, body io.Reader) error {
	if s.putErr != nil {
		return s.putErr
	}

	_, err := io.ReadAll(body)
	return err
}

func (s *stubStorage) Get(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (s *stubStorage) Delete(_ context.Context, _ string) error {
	return nil
}

type stubPublisher struct {
	publishErr error
}

func (p *stubPublisher) PublishAudioUploaded(_ context.Context, _ application.UploadedEvent) error {
	return p.publishErr
}

type stubAccessChecker struct {
	allowed bool
	err     error
}

func (c *stubAccessChecker) CanUpload(_ context.Context, _, _ string) (bool, error) {
	if c.err != nil {
		return false, c.err
	}
	return c.allowed, nil
}

func TestUploadHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := application.NewUploadService(
		&stubStorage{},
		&stubPublisher{},
		&stubAccessChecker{allowed: true},
	)
	h := NewUploadHandler(service, slog.Default())

	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		h.Upload(c)
	})

	body, contentType := buildMultipartBody(t, "audio", "meeting.mp3", uuid.NewString())
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-Id", "user-1")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.NotEmpty(t, response["audio_id"])
	_, err := uuid.Parse(response["audio_id"])
	require.NoError(t, err)
}

func TestUploadHandler_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := application.NewUploadService(
		&stubStorage{},
		&stubPublisher{},
		&stubAccessChecker{allowed: false},
	)
	h := NewUploadHandler(service, slog.Default())

	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		h.Upload(c)
	})

	body, contentType := buildMultipartBody(t, "audio", "meeting.mp3", uuid.NewString())
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-Id", "user-1")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "workspace access denied")
}

func TestUploadHandler_BadRequestWhenFileMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := application.NewUploadService(
		&stubStorage{},
		&stubPublisher{},
		&stubAccessChecker{allowed: true},
	)
	h := NewUploadHandler(service, slog.Default())

	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		h.Upload(c)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("workspace_id", uuid.NewString()))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-Id", "user-1")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "audio file is required")
}

func TestUploadHandler_InternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := application.NewUploadService(
		&stubStorage{},
		&stubPublisher{publishErr: errors.New("nats is down")},
		&stubAccessChecker{allowed: true},
	)
	h := NewUploadHandler(service, slog.Default())

	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		h.Upload(c)
	})

	body, contentType := buildMultipartBody(t, "audio", "meeting.mp3", uuid.NewString())
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-Id", "user-1")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to upload audio")
}

func buildMultipartBody(t *testing.T, fileField, filename, workspaceID string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("workspace_id", workspaceID))

	part, err := writer.CreateFormFile(fileField, filename)
	require.NoError(t, err)

	_, err = part.Write([]byte("audio-bytes"))
	require.NoError(t, err)

	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}
