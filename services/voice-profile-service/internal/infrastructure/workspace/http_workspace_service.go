package workspace

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	workspacecontract "github.com/Microservices/services/voice-profile-service/internal/application/contracts/workspace"
)

type HTTPService struct {
	baseURL string
	client  *http.Client
}

func NewHTTPService(baseURL string) workspacecontract.Service {
	return &HTTPService{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (h *HTTPService) GetParticipants(userId, workspaceId string) ([]string, error) {
	url := fmt.Sprintf("%s/workspaces/%s/participants", h.baseURL, workspaceId)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-User-Id", userId)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("workspace service returned status %d: %s", resp.StatusCode, string(body))
	}

	var payload []struct {
		ParticipantID string `json:"participant_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	participants := make([]string, 0, len(payload))
	for _, p := range payload {
		if p.ParticipantID != "" {
			participants = append(participants, p.ParticipantID)
		}
	}

	return participants, nil
}
