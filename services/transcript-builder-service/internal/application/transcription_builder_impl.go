package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/Microservices/services/transcript-builder-service/internal/application/interfaces"
	"github.com/Microservices/services/transcript-builder-service/internal/model"
	"github.com/Microservices/services/transcript-builder-service/internal/utils"
)

const (
	clusterSimilarityThreshold = 0.45
	minClusterSegmentDuration  = 5 * time.Second
	minEmbeddingDuration       = 7 * time.Second
	minFragmentDuration        = 2 * time.Second
)

type transcriptionBuilderImpl struct {
	diarizationService    interfaces.DiarizationService
	speakerRecognitionURL string
	httpClient            *http.Client
}

func NewTranscriptionBuilder(diarizationService interfaces.DiarizationService) interfaces.TranscriptionBuilder {
	return &transcriptionBuilderImpl{
		diarizationService: diarizationService,
	}
}

func NewTranscriptionBuilderWithEmbedding(diarizationService interfaces.DiarizationService, speakerRecognitionURL string) interfaces.TranscriptionBuilder {
	return &transcriptionBuilderImpl{
		diarizationService:    diarizationService,
		speakerRecognitionURL: strings.TrimRight(speakerRecognitionURL, "/"),
		httpClient:            &http.Client{Timeout: 120 * time.Second},
	}
}

func (t *transcriptionBuilderImpl) GetTranscriptionWithDiarization(
	workspaceId, userId string,
	segments []interfaces.TranscriptionSegment,
	audio io.ReadCloser,
) ([]model.ResultSegment, error) {
	defer func() { _ = audio.Close() }()

	audioData, err := io.ReadAll(audio)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio: %w", err)
	}

	sampleRate, bitDepth, numChannels, err := utils.GetWAVParamsFromBytes(audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to get WAV params: %w", err)
	}

	result := make([]model.ResultSegment, len(segments))
	for i, seg := range segments {
		audioSegment, err := utils.NewWAVSegment(
			audioData,
			seg.Interval.Start,
			seg.Interval.Duration,
			sampleRate,
			bitDepth,
			numChannels,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create audio segment: %w", err)
		}

		participantId, err := t.diarizationService.GetDiarization(workspaceId, userId, io.NopCloser(audioSegment))
		if err != nil {
			return nil, fmt.Errorf("failed to diarize audio segment: %w", err)
		}

		result[i] = model.ResultSegment{
			Interval:      seg.Interval,
			Text:          seg.Text,
			ParticipantId: participantId,
		}
	}

	return result, nil
}

func (t *transcriptionBuilderImpl) ClusterBySpeaker(
	segments []interfaces.TranscriptionSegment,
	audioData []byte,
	sessionID string,
) ([]interfaces.SpeakerCluster, error) {
	if t.speakerRecognitionURL == "" {
		return nil, fmt.Errorf("speaker recognition URL not configured")
	}

	sampleRate, bitDepth, numChannels, err := utils.GetWAVParamsFromBytes(audioData)
	if err != nil {
		return nil, fmt.Errorf("get WAV params: %w", err)
	}

	var centroids [][]float64
	var clusterLabels []string

	result := make([]interfaces.TranscriptionSegment, len(segments))
	for i, seg := range segments {
		// Build a window of consecutive segments to reach minEmbeddingDuration
		windowBytes, err := buildEmbeddingWindow(audioData, segments, i, sampleRate, bitDepth, numChannels)
		if err != nil {
			result[i] = seg
			result[i].Speaker = assignFallbackSpeaker(centroids, clusterLabels)
			continue
		}

		emb, err := t.getEmbedding(windowBytes)
		if err != nil {
			// On embedding failure, assign to last known cluster or SPEAKER_00
			result[i] = seg
			result[i].Speaker = assignFallbackSpeaker(centroids, clusterLabels)
			continue
		}

		isShort := seg.Interval.Duration < minClusterSegmentDuration
		_ = isShort

		// Find best matching cluster
		bestCluster := -1
		bestSim := clusterSimilarityThreshold - 0.001 // must exceed threshold
		for ci, centroid := range centroids {
			sim := cosineSimilarity(emb, centroid)
			slog.Debug("cluster similarity", "seg", i, "dur", seg.Interval.Duration.Seconds(), "cluster", ci, "sim", fmt.Sprintf("%.4f", sim))
			if sim > bestSim {
				bestSim = sim
				bestCluster = ci
			}
		}

		if bestCluster == -1 {
			// New cluster
			label := fmt.Sprintf("SPEAKER_%02d", len(centroids))
			slog.Debug("new cluster created", "seg", i, "dur", seg.Interval.Duration.Seconds(), "label", label, "best_sim_was", fmt.Sprintf("%.4f", bestSim))
			centroids = append(centroids, emb)
			clusterLabels = append(clusterLabels, label)
			bestCluster = len(centroids) - 1
		}

		result[i] = seg
		result[i].Speaker = clusterLabels[bestCluster]
	}

	// Group segments by speaker label and extract representative fragment
	type clusterAccum struct {
		segments []interfaces.TranscriptionSegment
		fragment []byte
	}
	clusterAccumMap := map[string]*clusterAccum{}
	for _, seg := range result {
		label := seg.Speaker
		if label == "" {
			label = "SPEAKER_00"
		}
		if _, ok := clusterAccumMap[label]; !ok {
			clusterAccumMap[label] = &clusterAccum{}
		}
		clusterAccumMap[label].segments = append(clusterAccumMap[label].segments, seg)
	}

	for label, accum := range clusterAccumMap {
		for _, seg := range accum.segments {
			if seg.Interval.Duration >= minFragmentDuration {
				wavSeg, err := utils.NewWAVSegment(audioData, seg.Interval.Start, seg.Interval.Duration, sampleRate, bitDepth, numChannels)
				if err == nil {
					fragBytes, err := io.ReadAll(wavSeg)
					if err == nil {
						clusterAccumMap[label].fragment = fragBytes
						break
					}
				}
			}
		}
		// Fallback: use first segment regardless of length
		if clusterAccumMap[label].fragment == nil && len(accum.segments) > 0 {
			seg := accum.segments[0]
			wavSeg, err := utils.NewWAVSegment(audioData, seg.Interval.Start, seg.Interval.Duration, sampleRate, bitDepth, numChannels)
			if err == nil {
				fragBytes, _ := io.ReadAll(wavSeg)
				clusterAccumMap[label].fragment = fragBytes
			}
		}
	}

	clusters := make([]interfaces.SpeakerCluster, 0, len(clusterAccumMap))
	for label, accum := range clusterAccumMap {
		clusters = append(clusters, interfaces.SpeakerCluster{
			Label:        label,
			FragmentData: accum.fragment,
			Segments:     accum.segments,
		})
	}
	sortClusters(clusters)

	return clusters, nil
}

func (t *transcriptionBuilderImpl) getEmbedding(wavBytes []byte) ([]float64, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", "segment.wav")
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := fw.Write(wavBytes); err != nil {
		return nil, fmt.Errorf("write wav bytes: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close multipart: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, t.speakerRecognitionURL+"/embedding", &body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding failed: status %d: %s", resp.StatusCode, b)
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embedding: %w", err)
	}
	return result.Embedding, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return -1
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return -1
	}
	return dot / (sqrt64(normA) * sqrt64(normB))
}

func sqrt64(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton's method — avoids importing math
	z := x
	for i := 0; i < 50; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

func assignFallbackSpeaker(centroids [][]float64, labels []string) string {
	if len(labels) == 0 {
		return "SPEAKER_00"
	}
	return labels[len(labels)-1]
}

func sortClusters(clusters []interfaces.SpeakerCluster) {
	for i := 1; i < len(clusters); i++ {
		for j := i; j > 0 && clusters[j].Label < clusters[j-1].Label; j-- {
			clusters[j], clusters[j-1] = clusters[j-1], clusters[j]
		}
	}
}

// buildEmbeddingWindow concatenates WAV segments starting at idx, expanding
// forward until total duration >= minEmbeddingDuration or no more segments.
func buildEmbeddingWindow(audioData []byte, segments []interfaces.TranscriptionSegment, idx, sampleRate, bitDepth, numChannels int) ([]byte, error) {
	var total time.Duration
	var buf bytes.Buffer

	for j := idx; j < len(segments) && total < minEmbeddingDuration; j++ {
		seg := segments[j]
		wavSeg, err := utils.NewWAVSegment(audioData, seg.Interval.Start, seg.Interval.Duration, sampleRate, bitDepth, numChannels)
		if err != nil {
			break
		}
		chunk, err := io.ReadAll(wavSeg)
		if err != nil {
			break
		}
		// Skip WAV header for all but the first chunk (44 bytes)
		if j == idx {
			buf.Write(chunk)
		} else if len(chunk) > 44 {
			buf.Write(chunk[44:])
		}
		total += seg.Interval.Duration
	}

	if buf.Len() == 0 {
		return nil, fmt.Errorf("no audio data")
	}

	// Fix WAV header data size
	data := buf.Bytes()
	if len(data) >= 44 {
		dataSize := uint32(len(data) - 44)
		fileSize := uint32(len(data) - 8)
		data[4] = byte(fileSize)
		data[5] = byte(fileSize >> 8)
		data[6] = byte(fileSize >> 16)
		data[7] = byte(fileSize >> 24)
		data[40] = byte(dataSize)
		data[41] = byte(dataSize >> 8)
		data[42] = byte(dataSize >> 16)
		data[43] = byte(dataSize >> 24)
	}
	return data, nil
}
