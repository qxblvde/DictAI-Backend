package workspace

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPAccessChecker struct {
	baseURL string
	client  *http.Client
}

func NewHTTPAccessChecker(baseURL string, client *http.Client) *HTTPAccessChecker {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	return &HTTPAccessChecker{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (c *HTTPAccessChecker) CanUpload(ctx context.Context, workspaceID, userID string) (bool, error) {
	if c.baseURL == "" {
		return false, errorsNotConfigured("workspace service URL is empty")
	}
	if workspaceID == "" || userID == "" {
		return false, nil
	}

	endpoint := c.baseURL + "/workspaces/" + url.PathEscape(workspaceID) + "/participants"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("build workspace request: %w", err)
	}

	req.Header.Set("X-User-ID", userID)

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("call workspace service: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("workspace access checker: close response body: %v", closeErr)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return false, nil
	default:
		if resp.StatusCode >= http.StatusInternalServerError {
			return false, fmt.Errorf("workspace service status: %d", resp.StatusCode)
		}
		return false, nil
	}
}

func errorsNotConfigured(message string) error {
	return fmt.Errorf("workspace checker is not configured: %s", message)
}
