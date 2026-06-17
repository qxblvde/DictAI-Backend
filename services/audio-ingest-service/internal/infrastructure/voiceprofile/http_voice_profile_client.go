package voiceprofile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPVoiceProfileClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPVoiceProfileClient(baseURL string) *HTTPVoiceProfileClient {
	return &HTTPVoiceProfileClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HTTPVoiceProfileClient) CreateLibraryProfile(ctx context.Context, userID, displayName string, audio io.Reader) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if displayName != "" {
		if err := w.WriteField("display_name", displayName); err != nil {
			return "", fmt.Errorf("write display_name: %w", err)
		}
	}

	fw, err := w.CreateFormFile("audio", "fragment.wav")
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, audio); err != nil {
		return "", fmt.Errorf("copy audio: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/voice-profiles/library", &body)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-User-Id", userID)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create library profile: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create library profile: status %d: %s", resp.StatusCode, b)
	}

	var result struct {
		VoiceProfileID string `json:"voice_profile_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode library profile response: %w", err)
	}
	return result.VoiceProfileID, nil
}

func (c *HTTPVoiceProfileClient) AssignProfile(ctx context.Context, userID, workspaceID, participantID, voiceProfileID string) error {
	body, _ := json.Marshal(map[string]string{
		"workspace_id":   workspaceID,
		"participant_id": participantID,
	})
	endpoint := c.baseURL + "/voice-profiles/profiles/" + url.PathEscape(voiceProfileID) + "/assign"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userID)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("assign profile: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("assign profile: status %d: %s", resp.StatusCode, b)
	}
	return nil
}

func closeBody(b io.ReadCloser) {
	if err := b.Close(); err != nil {
		log.Printf("close response body: %v", err)
	}
}
