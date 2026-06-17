package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Microservices/services/summarization-service/internal/application/interfaces"
)

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{baseURL: baseURL, httpClient: &http.Client{}}
}

type participantResponse struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
}

type workspaceResponse struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
}

func (c *HTTPClient) GetParticipants(ctx context.Context, workspaceID, userID string) ([]interfaces.Participant, error) {
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

	var raw []participantResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode participants: %w", err)
	}

	participants := make([]interfaces.Participant, len(raw))
	for i, p := range raw {
		participants[i] = interfaces.Participant{
			ParticipantID: p.ParticipantID,
			Name:          p.Name,
		}
	}
	return participants, nil
}

func (c *HTTPClient) GetWorkspace(ctx context.Context, workspaceID, userID string) (*interfaces.WorkspaceInfo, error) {
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

	var raw workspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode workspace: %w", err)
	}

	return &interfaces.WorkspaceInfo{
		WorkspaceID: raw.WorkspaceID,
		Name:        raw.Name,
	}, nil
}
