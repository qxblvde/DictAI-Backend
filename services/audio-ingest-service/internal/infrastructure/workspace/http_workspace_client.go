package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPWorkspaceClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPWorkspaceClient(baseURL string) *HTTPWorkspaceClient {
	return &HTTPWorkspaceClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *HTTPWorkspaceClient) CreateWorkspace(ctx context.Context, userID, name string) (string, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/workspaces/", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userID)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create workspace: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create workspace: status %d: %s", resp.StatusCode, b)
	}

	var result struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode workspace response: %w", err)
	}
	return result.WorkspaceID, nil
}

func (c *HTTPWorkspaceClient) AddParticipant(ctx context.Context, userID, workspaceID, name, email string) (string, error) {
	body, _ := json.Marshal(map[string]string{"name": name, "email": email})
	endpoint := c.baseURL + "/workspaces/" + url.PathEscape(workspaceID) + "/participants"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userID)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("add participant: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("add participant: status %d: %s", resp.StatusCode, b)
	}

	var result struct {
		ParticipantID string `json:"participant_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode participant response: %w", err)
	}
	return result.ParticipantID, nil
}

func closeBody(b io.ReadCloser) {
	if err := b.Close(); err != nil {
		log.Printf("close response body: %v", err)
	}
}
