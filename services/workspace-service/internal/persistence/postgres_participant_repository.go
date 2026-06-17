package persistence

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/DictAI/Microservices/services/workspace-service/internal/domain"
)

type PostgresParticipantRepository struct {
	db *sql.DB
}

func (repository *PostgresParticipantRepository) AddWorkspaceParticipant(participant *domain.Participant) error {
	query := `
        INSERT INTO participants (workspace_id, name, email)
        VALUES ($1, $2, $3)
        RETURNING participant_id, voice_profile_id
    `
	var voiceProfileID sql.NullString
	err := repository.db.QueryRow(query, participant.WorkspaceID, participant.Name, participant.Email).Scan(&participant.ParticipantID, &voiceProfileID)
	if err != nil {
		return err
	}
	participant.VoiceProfileID = nullStringPtr(voiceProfileID)
	return nil
}

func (repository *PostgresParticipantRepository) GetWorkspaceParticipants(workspaceID string) ([]*domain.Participant, error) {
	var participants []*domain.Participant
	query := `
        SELECT participant_id, workspace_id, name, email, voice_profile_id
        FROM participants
        WHERE workspace_id = $1
    `

	rows, err := repository.db.Query(query, workspaceID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Warn("failed to close db rows", "error", err)
		}
	}()

	for rows.Next() {
		participant := &domain.Participant{}
		var voiceProfileID sql.NullString

		err = rows.Scan(&participant.ParticipantID, &participant.WorkspaceID, &participant.Name, &participant.Email, &voiceProfileID)
		if err != nil {
			return nil, err
		}
		participant.VoiceProfileID = nullStringPtr(voiceProfileID)

		participants = append(participants, participant)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return participants, nil
}

func (repository *PostgresParticipantRepository) DeleteParticipant(workspaceID, participantID string) error {
	query := `DELETE FROM participants WHERE workspace_id = $1 AND participant_id = $2`
	result, err := repository.db.Exec(query, workspaceID, participantID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (repository *PostgresParticipantRepository) UpdateParticipant(workspaceID, participantID, name, email string) (*domain.Participant, error) {
	setParts := []string{}
	args := []any{}

	if name != "" {
		args = append(args, name)
		setParts = append(setParts, fmt.Sprintf("name = $%d", len(args)))
	}
	if email != "" {
		args = append(args, email)
		setParts = append(setParts, fmt.Sprintf("email = $%d", len(args)))
	}

	args = append(args, workspaceID, participantID)
	query := fmt.Sprintf(`
        UPDATE participants SET %s
        WHERE workspace_id = $%d AND participant_id = $%d
        RETURNING participant_id, workspace_id, name, email, voice_profile_id
    `, strings.Join(setParts, ", "), len(args)-1, len(args))

	var p domain.Participant
	var voiceProfileID sql.NullString
	err := repository.db.QueryRow(query, args...).Scan(&p.ParticipantID, &p.WorkspaceID, &p.Name, &p.Email, &voiceProfileID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("update participant: %w", err)
	}
	p.VoiceProfileID = nullStringPtr(voiceProfileID)
	return &p, nil
}

func NewPostgresParticipantRepository(db *sql.DB) *PostgresParticipantRepository {
	return &PostgresParticipantRepository{db: db}
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}
