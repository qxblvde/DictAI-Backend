package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Microservices/services/results-service/internal/domain"
	"github.com/Microservices/services/results-service/internal/infrastructure/persistence"
	"github.com/Microservices/services/results-service/internal/infrastructure/storage"
)

var ErrResultNotFound = errors.New("result not found")

type ResultService struct {
	repo       *persistence.ResultRepository
	storage    *storage.MinIOStorage
	audioStore *storage.MinIOStorage
	baseURL    string
}

func New(repo *persistence.ResultRepository, store *storage.MinIOStorage, audioStore *storage.MinIOStorage, baseURL string) *ResultService {
	return &ResultService{repo: repo, storage: store, audioStore: audioStore, baseURL: baseURL}
}

type UploadInput struct {
	AudioID       string
	WorkspaceID   string
	UploadUserID  string
	SummaryPDF    []byte
	TranscriptPDF []byte
}

type UploadOutput struct {
	ResultID      string
	SummaryURL    string
	TranscriptURL string
}

type ListFilters struct {
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	Limit     int
}

type ListOutput struct {
	Results    []ResultItem
	Total      int
	Page       int
	Limit      int
	TotalPages int
}

type ResultItem struct {
	ResultID      string    `json:"result_id"`
	AudioID       string    `json:"audio_id"`
	WorkspaceID   string    `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name"`
	UploadUserID  string    `json:"upload_user_id"`
	SummaryURL    string    `json:"summary_url"`
	TranscriptURL string    `json:"transcript_url"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *ResultService) List(ctx context.Context, userID string, f ListFilters) (*ListOutput, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}

	results, total, err := s.repo.List(ctx, persistence.ListFilters{
		UserID:    userID,
		StartDate: f.StartDate,
		EndDate:   f.EndDate,
		Page:      f.Page,
		Limit:     f.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list results: %w", err)
	}

	items := make([]ResultItem, 0, len(results))
	for _, r := range results {
		items = append(items, ResultItem{
			ResultID:      r.ResultID,
			AudioID:       r.AudioID,
			WorkspaceID:   r.WorkspaceID,
			WorkspaceName: r.WorkspaceName,
			UploadUserID:  r.UploadUserID,
			SummaryURL:    "/meetings/results/" + r.AudioID + "/summary",
			TranscriptURL: "/meetings/results/" + r.AudioID + "/transcript",
			Status:        r.Status,
			CreatedAt:     r.CreatedAt,
		})
	}

	totalPages := total / f.Limit
	if total%f.Limit != 0 {
		totalPages++
	}

	return &ListOutput{
		Results:    items,
		Total:      total,
		Page:       f.Page,
		Limit:      f.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *ResultService) CreatePending(ctx context.Context, audioID, workspaceID, uploadUserID string) error {
	result := &domain.Result{
		AudioID:      audioID,
		WorkspaceID:  workspaceID,
		UploadUserID: uploadUserID,
		Status:       "pending",
	}
	return s.repo.Create(ctx, result)
}

func (s *ResultService) SetStatus(ctx context.Context, audioID, status string) error {
	if err := s.repo.UpdateStatus(ctx, audioID, status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResultNotFound
		}
		return fmt.Errorf("set status: %w", err)
	}
	return nil
}

func (s *ResultService) Upload(ctx context.Context, in UploadInput) (*UploadOutput, error) {
	summaryKey := fmt.Sprintf("%s/%s/summary.pdf", in.WorkspaceID, in.AudioID)
	transcriptKey := fmt.Sprintf("%s/%s/transcript.pdf", in.WorkspaceID, in.AudioID)

	if err := s.storage.Put(ctx, summaryKey, bytes.NewReader(in.SummaryPDF), int64(len(in.SummaryPDF)), "application/pdf"); err != nil {
		return nil, fmt.Errorf("upload summary: %w", err)
	}
	if err := s.storage.Put(ctx, transcriptKey, bytes.NewReader(in.TranscriptPDF), int64(len(in.TranscriptPDF)), "application/pdf"); err != nil {
		return nil, fmt.Errorf("upload transcript: %w", err)
	}

	// Update existing pending record or create new done record
	existing, err := s.repo.GetByAudioID(ctx, in.AudioID)
	if err == nil && existing != nil {
		// Update keys and status
		existing.SummaryKey = summaryKey
		existing.TranscriptKey = transcriptKey
		existing.Status = "done"
		if err := s.repo.UpdateKeysAndStatus(ctx, existing); err != nil {
			return nil, fmt.Errorf("update result: %w", err)
		}
	} else {
		result := &domain.Result{
			AudioID:       in.AudioID,
			WorkspaceID:   in.WorkspaceID,
			UploadUserID:  in.UploadUserID,
			SummaryKey:    summaryKey,
			TranscriptKey: transcriptKey,
			Status:        "done",
		}
		if err := s.repo.Create(ctx, result); err != nil {
			return nil, fmt.Errorf("save result metadata: %w", err)
		}
	}

	return &UploadOutput{
		SummaryURL:    s.baseURL + "/results/" + in.AudioID + "/summary",
		TranscriptURL: s.baseURL + "/results/" + in.AudioID + "/transcript",
	}, nil
}

func (s *ResultService) Delete(ctx context.Context, userID, audioID string) error {
	result, err := s.repo.GetByAudioIDForUser(ctx, audioID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResultNotFound
		}
		return fmt.Errorf("load result metadata: %w", err)
	}

	if result.SummaryKey != "" {
		if err := s.storage.Delete(ctx, result.SummaryKey); err != nil {
			return fmt.Errorf("delete summary: %w", err)
		}
	}
	if result.TranscriptKey != "" {
		if err := s.storage.Delete(ctx, result.TranscriptKey); err != nil {
			return fmt.Errorf("delete transcript: %w", err)
		}
	}
	if err := s.repo.DeleteByAudioIDForUser(ctx, audioID, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResultNotFound
		}
		return fmt.Errorf("delete result metadata: %w", err)
	}
	return nil
}

func (s *ResultService) GetFile(ctx context.Context, audioID, fileType string) ([]byte, error) {
	result, err := s.repo.GetByAudioID(ctx, audioID)
	if err != nil {
		return nil, err
	}

	var key string
	switch fileType {
	case "summary":
		key = result.SummaryKey
	case "transcript":
		key = result.TranscriptKey
	default:
		return nil, fmt.Errorf("unknown file type: %s", fileType)
	}

	if key == "" {
		return nil, fmt.Errorf("file not ready yet")
	}

	rc, err := s.storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get file from storage: %w", err)
	}
	defer func() { _ = rc.Close() }()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(rc); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *ResultService) GetAudio(ctx context.Context, audioID string) ([]byte, string, error) {
	result, err := s.repo.GetByAudioID(ctx, audioID)
	if err != nil {
		return nil, "", ErrResultNotFound
	}

	// Try all possible paths
	paths := []string{
		"anonymous/" + audioID + ".mp3",
	}
	if result.WorkspaceID != "" {
		paths = append([]string{result.WorkspaceID + "/" + audioID + ".mp3"}, paths...)
	}

	for _, key := range paths {
		rc, err := s.audioStore.Get(ctx, key)
		if err == nil {
			defer func() { _ = rc.Close() }()
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(rc); err != nil {
				return nil, "", fmt.Errorf("read audio: %w", err)
			}
			return buf.Bytes(), "audio/mpeg", nil
		}
	}
	return nil, "", fmt.Errorf("audio not found")
}
