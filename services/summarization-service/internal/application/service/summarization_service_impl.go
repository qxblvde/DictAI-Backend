package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Microservices/services/summarization-service/internal/application/interfaces"
	"github.com/Microservices/services/summarization-service/internal/infrastructure/typst"
)

const unknownParticipantName = "Неизвестный участник"

type SummarizationService struct {
	summarizer      interfaces.Summarizer
	resultsClient   interfaces.ResultsClient
	publisher       interfaces.Publisher
	workspaceClient interfaces.WorkspaceClient
	publicBaseURL   string
	resultsBaseURL  string
}

func New(
	summarizer interfaces.Summarizer,
	resultsClient interfaces.ResultsClient,
	publisher interfaces.Publisher,
	workspaceClient interfaces.WorkspaceClient,
	publicBaseURL string,
	resultsBaseURL string,
) *SummarizationService {
	return &SummarizationService{
		summarizer:      summarizer,
		resultsClient:   resultsClient,
		publisher:       publisher,
		workspaceClient: workspaceClient,
		publicBaseURL:   publicBaseURL,
		resultsBaseURL:  resultsBaseURL,
	}
}

func (s *SummarizationService) Process(ctx context.Context, audioID, workspaceID, uploadUserID string, segments []interfaces.Segment) error {
	if audioID == "" || workspaceID == "" || uploadUserID == "" {
		return errors.New("audioID, workspaceID, uploadUserID are required")
	}
	if len(segments) == 0 {
		return errors.New("segments are empty")
	}

	participants, err := s.workspaceClient.GetParticipants(ctx, workspaceID, uploadUserID)
	if err != nil {
		slog.Warn("failed to fetch participants, using IDs", "error", err)
	}

	nameMap := make(map[string]string, len(participants))
	for _, p := range participants {
		nameMap[p.ParticipantID] = p.Name
	}

	// Resolve uploader name
	uploaderName := uploadUserID
	if name, ok := nameMap[uploadUserID]; ok {
		uploaderName = name
	}

	// Resolve workspace name
	workspaceName := workspaceID
	if ws, wsErr := s.workspaceClient.GetWorkspace(ctx, workspaceID, uploadUserID); wsErr == nil {
		workspaceName = ws.Name
	} else {
		slog.Warn("failed to fetch workspace name", "error", wsErr)
	}

	enriched := make([]interfaces.Segment, len(segments))
	for i, seg := range segments {
		enriched[i] = seg
		if seg.ParticipantID == "" {
			enriched[i].ParticipantName = unknownParticipantName
			continue
		}
		if name, ok := nameMap[seg.ParticipantID]; ok && name != "" {
			enriched[i].ParticipantName = name
		}
	}

	meta := interfaces.MeetingMeta{
		WorkspaceName: workspaceName,
		UploaderName:  uploaderName,
		Date:          time.Now().Format("02.01.2006 15:04"),
	}

	result, err := s.summarizer.SummarizeWithMeta(ctx, enriched, meta)
	if err != nil {
		return err
	}

	summaryPDF, err := typst.Compile(ctx, typst.SummaryDoc(meta, result.SummaryFragment))
	if err != nil {
		return fmt.Errorf("compile summary pdf: %w", err)
	}

	transcriptPDF, err := typst.Compile(ctx, typst.TranscriptDoc(meta, result.TranscriptFragment))
	if err != nil {
		return fmt.Errorf("compile transcript pdf: %w", err)
	}

	uploaded, err := s.resultsClient.Upload(ctx, interfaces.UploadInput{
		AudioID:       audioID,
		WorkspaceID:   workspaceID,
		UploadUserID:  uploadUserID,
		SummaryPDF:    summaryPDF,
		TranscriptPDF: transcriptPDF,
	})
	if err != nil {
		return err
	}

	slog.Info("results uploaded", "audio_id", audioID, "summary_url", uploaded.SummaryURL)

	summaryURL := s.toPublicURL(uploaded.SummaryURL)
	transcriptURL := s.toPublicURL(uploaded.TranscriptURL)
	if err := s.publisher.Publish(audioID, workspaceID, uploadUserID, summaryURL, transcriptURL); err != nil {
		return err
	}
	s.setResultStatus(audioID, "done")
	return nil
}

func (s *SummarizationService) toPublicURL(internal string) string {
	if s.publicBaseURL == "" || internal == "" {
		return internal
	}
	u, err := url.Parse(internal)
	if err != nil {
		return internal
	}
	pub, err := url.Parse(s.publicBaseURL)
	if err != nil {
		return internal
	}
	u.Scheme = pub.Scheme
	u.Host = pub.Host
	// strip /results/ prefix — gateway exposes it under /results/
	u.Path = strings.TrimPrefix(u.Path, "")
	return u.String()
}

func (s *SummarizationService) setResultStatus(audioID, status string) {
	if s.resultsBaseURL == "" {
		return
	}
	body := []byte(`{"status":"` + status + `"}`)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPatch,
		s.resultsBaseURL+"/meetings/results/"+audioID+"/status", bytes.NewReader(body))
	if err != nil {
		slog.Warn("failed to build set-status request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("failed to set result status", "audio_id", audioID, "status", status, "error", err)
		return
	}
	_ = resp.Body.Close()
}
