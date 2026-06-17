package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Microservices/services/results-service/internal/domain"
)

type ListFilters struct {
	UserID    string
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	Limit     int
}

type ResultRepository struct {
	db *pgxpool.Pool
}

func NewResultRepository(db *pgxpool.Pool) *ResultRepository {
	return &ResultRepository{db: db}
}

func (r *ResultRepository) Create(ctx context.Context, result *domain.Result) error {
	query := `
        INSERT INTO results (audio_id, workspace_id, upload_user_id, summary_key, transcript_key, status)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING result_id, created_at
    `
	return r.db.QueryRow(ctx, query,
		result.AudioID,
		result.WorkspaceID,
		result.UploadUserID,
		result.SummaryKey,
		result.TranscriptKey,
		result.Status,
	).Scan(&result.ResultID, &result.CreatedAt)
}

func (r *ResultRepository) UpdateStatus(ctx context.Context, audioID, status string) error {
	tag, err := r.db.Exec(ctx, `UPDATE results SET status = $1 WHERE audio_id = $2`, status, audioID)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ResultRepository) UpdateKeysAndStatus(ctx context.Context, result *domain.Result) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE results SET summary_key = $1, transcript_key = $2, status = $3 WHERE audio_id = $4`,
		result.SummaryKey, result.TranscriptKey, result.Status, result.AudioID,
	)
	if err != nil {
		return fmt.Errorf("update keys and status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ResultRepository) List(ctx context.Context, f ListFilters) ([]domain.Result, int, error) {
	args := []any{f.UserID}
	conditions := []string{"r.upload_user_id = $1"}

	if f.StartDate != nil {
		args = append(args, f.StartDate)
		conditions = append(conditions, fmt.Sprintf("r.created_at >= $%d", len(args)))
	}
	if f.EndDate != nil {
		args = append(args, f.EndDate)
		conditions = append(conditions, fmt.Sprintf("r.created_at <= $%d", len(args)))
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	countQuery := "SELECT COUNT(*) FROM results r " + where
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count results: %w", err)
	}

	offset := (f.Page - 1) * f.Limit
	args = append(args, f.Limit, offset)
	dataQuery := fmt.Sprintf(`
        SELECT r.result_id, r.audio_id, r.workspace_id, COALESCE(w.name, '') AS workspace_name,
               r.upload_user_id, r.summary_key, r.transcript_key, r.status, r.created_at
        FROM results r
        LEFT JOIN workspaces w ON w.workspace_id = r.workspace_id
        %s
        ORDER BY r.created_at DESC
        LIMIT $%d OFFSET $%d
    `, where, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list results: %w", err)
	}
	defer rows.Close()

	var results []domain.Result
	for rows.Next() {
		var res domain.Result
		if err := rows.Scan(
			&res.ResultID,
			&res.AudioID,
			&res.WorkspaceID,
			&res.WorkspaceName,
			&res.UploadUserID,
			&res.SummaryKey,
			&res.TranscriptKey,
			&res.Status,
			&res.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan result row: %w", err)
		}
		results = append(results, res)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate result rows: %w", err)
	}

	return results, total, nil
}

func (r *ResultRepository) GetByAudioID(ctx context.Context, audioID string) (*domain.Result, error) {
	query := `
        SELECT result_id, audio_id, workspace_id, upload_user_id, summary_key, transcript_key, status, created_at
        FROM results
        WHERE audio_id = $1
        ORDER BY created_at DESC
        LIMIT 1
    `
	return r.scanResult(ctx, query, audioID)
}

func (r *ResultRepository) GetByAudioIDForUser(ctx context.Context, audioID, userID string) (*domain.Result, error) {
	query := `
        SELECT result_id, audio_id, workspace_id, upload_user_id, summary_key, transcript_key, status, created_at
        FROM results
        WHERE audio_id = $1 AND upload_user_id = $2
        ORDER BY created_at DESC
        LIMIT 1
    `
	return r.scanResult(ctx, query, audioID, userID)
}

func (r *ResultRepository) DeleteByAudioIDForUser(ctx context.Context, audioID, userID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM results WHERE audio_id = $1 AND upload_user_id = $2`, audioID, userID)
	if err != nil {
		return fmt.Errorf("delete result: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ResultRepository) scanResult(ctx context.Context, query string, args ...any) (*domain.Result, error) {
	res := &domain.Result{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&res.ResultID,
		&res.AudioID,
		&res.WorkspaceID,
		&res.UploadUserID,
		&res.SummaryKey,
		&res.TranscriptKey,
		&res.Status,
		&res.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get result by audio_id: %w", err)
	}
	return res, nil
}
