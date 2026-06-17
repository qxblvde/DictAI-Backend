package results

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/Microservices/services/summarization-service/internal/application/interfaces"
)

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

type uploadResponse struct {
	ResultID      string `json:"result_id"`
	SummaryURL    string `json:"summary_url"`
	TranscriptURL string `json:"transcript_url"`
}

func (c *HTTPClient) Upload(ctx context.Context, in interfaces.UploadInput) (*interfaces.UploadOutput, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	_ = w.WriteField("audio_id", in.AudioID)
	_ = w.WriteField("workspace_id", in.WorkspaceID)
	_ = w.WriteField("upload_user_id", in.UploadUserID)

	summaryPart, err := w.CreateFormFile("summary", "summary.pdf")
	if err != nil {
		return nil, fmt.Errorf("create summary part: %w", err)
	}
	if _, err := summaryPart.Write(in.SummaryPDF); err != nil {
		return nil, fmt.Errorf("write summary: %w", err)
	}

	transcriptPart, err := w.CreateFormFile("transcript", "transcript.pdf")
	if err != nil {
		return nil, fmt.Errorf("create transcript part: %w", err)
	}
	if _, err := transcriptPart.Write(in.TranscriptPDF); err != nil {
		return nil, fmt.Errorf("write transcript: %w", err)
	}

	_ = w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/meetings/results", &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("results-service returned status %d", resp.StatusCode)
	}

	var out uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &interfaces.UploadOutput{
		ResultID:      out.ResultID,
		SummaryURL:    out.SummaryURL,
		TranscriptURL: out.TranscriptURL,
	}, nil
}
