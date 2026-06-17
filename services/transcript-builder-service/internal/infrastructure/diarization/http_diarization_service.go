package diarization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type HTTPDiarizationService struct {
	baseURL string
	client  *http.Client
}

func NewHTTPDiarizationService(baseURL string) *HTTPDiarizationService {
	return &HTTPDiarizationService{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (h *HTTPDiarizationService) GetDiarization(workspaceId, userId string, audioSegment io.ReadCloser) (string, error) {
	defer func() { _ = audioSegment.Close() }()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if err := w.WriteField("workspace_id", workspaceId); err != nil {
		return "", fmt.Errorf("write workspace_id field: %w", err)
	}

	fw, err := w.CreateFormFile("file", "segment.wav")
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, audioSegment); err != nil {
		return "", fmt.Errorf("copy audio segment: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, h.baseURL+"/identify-speaker", &body)
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if userId != "" {
		req.Header.Set("X-User-Id", userId)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("identify-speaker request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("identify-speaker failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ParticipantId *string `json:"participant_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.ParticipantId == nil {
		return "", nil
	}
	return *result.ParticipantId, nil
}
