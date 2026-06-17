package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Participant struct {
	ParticipantID string
	Name          string
	Email         string
}

type WorkspaceInfo struct {
	Name string
}

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{baseURL: baseURL, httpClient: &http.Client{}}
}

func (c *HTTPClient) GetParticipants(ctx context.Context, workspaceID, userID string) ([]Participant, error) {
	url := fmt.Sprintf("%s/workspaces/%s/participants", c.baseURL, workspaceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-User-Id", userID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch participants: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("workspace-service returned status %d", resp.StatusCode)
	}

	var raw []struct {
		ParticipantID string `json:"participant_id"`
		Name          string `json:"name"`
		Email         string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode participants: %w", err)
	}

	out := make([]Participant, len(raw))
	for i, p := range raw {
		out[i] = Participant{ParticipantID: p.ParticipantID, Name: p.Name, Email: p.Email}
	}
	return out, nil
}

func (c *HTTPClient) GetWorkspace(ctx context.Context, workspaceID, userID string) (*WorkspaceInfo, error) {
	url := fmt.Sprintf("%s/workspaces/%s", c.baseURL, workspaceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-User-Id", userID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch workspace: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("workspace-service returned status %d", resp.StatusCode)
	}

	var raw struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode workspace: %w", err)
	}
	return &WorkspaceInfo{Name: raw.Name}, nil
}
