package py_embedding_service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/Microservices/services/voice-profile-service/internal/application/contracts/embeddings"
)

type pyService struct {
	endpoint string
}

func NewPyService(endpoint string) embeddings.EmbeddingService {
	return &pyService{
		endpoint: endpoint,
	}
}

func (p pyService) Get(audio io.ReadCloser) ([192]float32, error) {
	var zero [192]float32

	if audio == nil {
		return zero, fmt.Errorf("audio is nil")
	}

	audioBytes, err := io.ReadAll(audio)
	if err != nil {
		return zero, fmt.Errorf("reading audio: %w", err)
	}

	const maxAttempts = 10
	const retryDelay = 5 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, status, err := p.doRequest(audioBytes)
		if err != nil {
			return zero, err
		}
		if status == http.StatusServiceUnavailable {
			log.Printf("embedding service unavailable (attempt %d/%d), retrying in %s", attempt, maxAttempts, retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		return result, nil
	}

	return zero, fmt.Errorf("embedding service unavailable after %d attempts", maxAttempts)
}

func (p pyService) doRequest(audioBytes []byte) ([192]float32, int, error) {
	var zero [192]float32

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	fw, err := mw.CreateFormFile("file", "audio.wav")
	if err != nil {
		return zero, 0, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := fw.Write(audioBytes); err != nil {
		return zero, 0, fmt.Errorf("writing audio: %w", err)
	}
	if err := mw.Close(); err != nil {
		return zero, 0, fmt.Errorf("closing multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", p.endpoint, &body)
	if err != nil {
		return zero, 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return zero, 0, fmt.Errorf("sending request to python embedding service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("Unable to close: ", err)
		}
	}()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return zero, http.StatusServiceUnavailable, nil
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return zero, resp.StatusCode, fmt.Errorf("python embedding service returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return zero, resp.StatusCode, fmt.Errorf("decoding response JSON: %w", err)
	}

	if len(parsed.Embedding) != 192 {
		return zero, resp.StatusCode, fmt.Errorf("invalid embedding length: expected 192 floats, got %d", len(parsed.Embedding))
	}

	var out [192]float32
	for i := 0; i < 192; i++ {
		out[i] = float32(parsed.Embedding[i])
	}

	return out, resp.StatusCode, nil
}
