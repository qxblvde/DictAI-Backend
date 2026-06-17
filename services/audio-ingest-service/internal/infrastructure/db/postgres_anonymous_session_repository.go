package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"audio-ingest-service/internal/domain"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresAnonymousSessionRepository struct {
	db *sql.DB
}

func NewPostgresAnonymousSessionRepository(db *sql.DB) *PostgresAnonymousSessionRepository {
	return &PostgresAnonymousSessionRepository{db: db}
}

func (r *PostgresAnonymousSessionRepository) Create(ctx context.Context, s *domain.AnonymousSession) error {
	q := `
		INSERT INTO anonymous_sessions (session_id, audio_id, owner_user_id, status, created_at, expires_at)
		VALUES (gen_random_uuid(), $1, $2, 'processing', NOW(), NOW() + INTERVAL '7 days')
		RETURNING session_id, created_at, expires_at`
	return r.db.QueryRowContext(ctx, q, s.AudioID, s.OwnerUserID).
		Scan(&s.SessionID, &s.CreatedAt, &s.ExpiresAt)
}

func (r *PostgresAnonymousSessionRepository) GetByID(ctx context.Context, sessionID string) (*domain.AnonymousSession, error) {
	q := `SELECT session_id, audio_id, owner_user_id, status, created_at, expires_at
		  FROM anonymous_sessions WHERE session_id = $1`
	var s domain.AnonymousSession
	err := r.db.QueryRowContext(ctx, q, sessionID).
		Scan(&s.SessionID, &s.AudioID, &s.OwnerUserID, &s.Status, &s.CreatedAt, &s.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &s, nil
}

func (r *PostgresAnonymousSessionRepository) GetByAudioID(ctx context.Context, audioID string) (*domain.AnonymousSession, error) {
	q := `SELECT session_id, audio_id, owner_user_id, status, created_at, expires_at
		  FROM anonymous_sessions WHERE audio_id = $1`
	var s domain.AnonymousSession
	err := r.db.QueryRowContext(ctx, q, audioID).
		Scan(&s.SessionID, &s.AudioID, &s.OwnerUserID, &s.Status, &s.CreatedAt, &s.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session by audio: %w", err)
	}
	return &s, nil
}

func (r *PostgresAnonymousSessionRepository) UpdateStatus(ctx context.Context, sessionID string, status domain.SessionStatus) error {
	q := `UPDATE anonymous_sessions SET status = $1 WHERE session_id = $2`
	_, err := r.db.ExecContext(ctx, q, string(status), sessionID)
	return err
}

func (r *PostgresAnonymousSessionRepository) AddSpeaker(ctx context.Context, sp *domain.AnonymousSpeaker) error {
	segsJSON, err := json.Marshal(sp.Segments)
	if err != nil {
		return fmt.Errorf("marshal segments: %w", err)
	}
	q := `INSERT INTO anonymous_speakers (session_id, label, fragment_key, segments)
		  VALUES ($1, $2, $3, $4)
		  RETURNING speaker_id, created_at`
	return r.db.QueryRowContext(ctx, q, sp.SessionID, sp.Label, sp.FragmentKey, segsJSON).
		Scan(&sp.SpeakerID, &sp.CreatedAt)
}

func (r *PostgresAnonymousSessionRepository) GetSpeakers(ctx context.Context, sessionID string) ([]domain.AnonymousSpeaker, error) {
	q := `SELECT speaker_id, session_id, label, fragment_key, segments, created_at
		  FROM anonymous_speakers WHERE session_id = $1 ORDER BY label`
	rows, err := r.db.QueryContext(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query speakers: %w", err)
	}
	defer rows.Close()

	var speakers []domain.AnonymousSpeaker
	for rows.Next() {
		var sp domain.AnonymousSpeaker
		var segsJSON []byte
		if err := rows.Scan(&sp.SpeakerID, &sp.SessionID, &sp.Label, &sp.FragmentKey, &segsJSON, &sp.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan speaker: %w", err)
		}
		if err := json.Unmarshal(segsJSON, &sp.Segments); err != nil {
			return nil, fmt.Errorf("unmarshal segments: %w", err)
		}
		speakers = append(speakers, sp)
	}
	return speakers, rows.Err()
}

func (r *PostgresAnonymousSessionRepository) GetSpeakerByID(ctx context.Context, speakerID string) (*domain.AnonymousSpeaker, error) {
	q := `SELECT speaker_id, session_id, label, fragment_key, segments, created_at
		  FROM anonymous_speakers WHERE speaker_id = $1`
	var sp domain.AnonymousSpeaker
	var segsJSON []byte
	err := r.db.QueryRowContext(ctx, q, speakerID).
		Scan(&sp.SpeakerID, &sp.SessionID, &sp.Label, &sp.FragmentKey, &segsJSON, &sp.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get speaker: %w", err)
	}
	if err := json.Unmarshal(segsJSON, &sp.Segments); err != nil {
		return nil, fmt.Errorf("unmarshal segments: %w", err)
	}
	return &sp, nil
}

func (r *PostgresAnonymousSessionRepository) CleanupExpired(ctx context.Context) ([]string, error) {
	// Collect keys before marking expired
	q := `SELECT s.audio_id, sp.fragment_key
		  FROM anonymous_sessions s
		  LEFT JOIN anonymous_speakers sp ON sp.session_id = s.session_id
		  WHERE s.expires_at < NOW() AND s.status IN ('processing','ready')`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query expired: %w", err)
	}
	defer rows.Close()

	keySet := map[string]struct{}{}
	for rows.Next() {
		var audioID string
		var fragmentKey sql.NullString
		if err := rows.Scan(&audioID, &fragmentKey); err != nil {
			return nil, fmt.Errorf("scan expired row: %w", err)
		}
		keySet[fmt.Sprintf("anonymous/%s.mp3", audioID)] = struct{}{}
		if fragmentKey.Valid {
			keySet[fragmentKey.String] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE anonymous_sessions SET status = 'expired'
		 WHERE expires_at < NOW() AND status IN ('processing','ready')`)
	if err != nil {
		return nil, fmt.Errorf("mark expired: %w", err)
	}

	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	return keys, nil
}
