package transcriber

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Whisper struct {
	endpoint string
	client   *http.Client
}

func NewWhisper(endpoint string) *Whisper {
	return &Whisper{
		endpoint: strings.TrimRight(endpoint, "/"),
		client:   &http.Client{},
	}
}

// Transcript sends raw audio as application/octet-stream to the transcription service
// and returns segments formatted as lines: "<start>-<end>: <text>" (start/end with 2 decimals).
func (w *Whisper) Transcript(ctx context.Context, audio io.ReadCloser) (string, error) {
	if err := w.waitForReady(ctx); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.endpoint+"/transcribe", audio)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close http response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("transcribe failed: status %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		Success  bool   `json:"success"`
		Language string `json:"language"`
		Segments []struct {
			Start float64 `json:"start"`
			End   float64 `json:"end"`
			Text  string  `json:"text"`
		} `json:"segments"`
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return "", err
	}
	if !out.Success {
		return "", fmt.Errorf("transcription service returned success=false")
	}

	var b strings.Builder
	for i, s := range out.Segments {
		if i > 0 {
			b.WriteByte('\n')
		}
		_, err := fmt.Fprintf(&b, "%.2f-%.2f: %s", s.Start, s.End, strings.TrimSpace(s.Text))
		if err != nil {
			return "", err
		}
	}

	return b.String(), nil
}

func (w *Whisper) waitForReady(ctx context.Context) error {
	base := strings.TrimRight(w.endpoint, "/")
	base = strings.TrimSuffix(base, "/transcribe")
	healthURL := base + "/health"

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return err
		}

		resp, err := w.client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			switch resp.StatusCode {
			case http.StatusOK:
				return nil
			case http.StatusAccepted, http.StatusServiceUnavailable:
				// model is loading, retry below
			default:
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
