package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Microservices/services/voice-profile-service/internal/application/contracts/persistence"
)

type voiceProfilePostgresRepository struct {
	e DBExecutor
}

func NewVoiceProfilePostgresRepository(executor DBExecutor) persistence.VoiceProfileRepository {
	return &voiceProfilePostgresRepository{executor}
}

func (v *voiceProfilePostgresRepository) CreateProfile(embeddings [192]float32, ownerUserId, participantId, displayName string) (string, error) {
	query := `
        INSERT INTO voice_profiles (embedding, participant_id, owner_user_id, display_name)
        VALUES ($1::vector, $2, $3, COALESCE(NULLIF($4, ''), 'Voice profile'))
        ON CONFLICT (participant_id) DO UPDATE
            SET embedding = EXCLUDED.embedding,
                owner_user_id = EXCLUDED.owner_user_id,
                display_name = COALESCE(NULLIF(EXCLUDED.display_name, ''), voice_profiles.display_name),
                created_at = NOW()
        RETURNING voice_profile_id
    `

	vectorLiteral := embeddingToVectorLiteral(embeddings)

	var voiceProfile string
	err := v.e.QueryRow(query, vectorLiteral, participantId, ownerUserId, displayName).Scan(&voiceProfile)

	return voiceProfile, err
}

func (v *voiceProfilePostgresRepository) CreateLibraryProfile(embeddings [192]float32, ownerUserId, displayName string) (string, error) {
	query := `
        INSERT INTO voice_profiles (embedding, owner_user_id, display_name)
        VALUES ($1::vector, $2, COALESCE(NULLIF($3, ''), 'Voice profile'))
        RETURNING voice_profile_id
    `

	vectorLiteral := embeddingToVectorLiteral(embeddings)

	var voiceProfile string
	err := v.e.QueryRow(query, vectorLiteral, ownerUserId, displayName).Scan(&voiceProfile)

	return voiceProfile, err
}

func (v *voiceProfilePostgresRepository) AssignProfile(ownerUserId, workspaceId, participantId, voiceProfileId string) error {
	query := `
        UPDATE participants p
        SET voice_profile_id = vp.voice_profile_id
        FROM workspaces w, voice_profiles vp
        WHERE p.workspace_id = w.workspace_id
          AND p.workspace_id = $2
          AND p.participant_id = $3
          AND w.owner_user_id = $1
          AND vp.voice_profile_id = $4
          AND vp.owner_user_id = $1
    `
	result, err := v.e.Exec(query, ownerUserId, workspaceId, participantId, voiceProfileId)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("voice profile access denied or participant not found")
	}
	return nil
}

func (v *voiceProfilePostgresRepository) ListProfiles(ownerUserId string) ([]persistence.ProfileSummary, error) {
	query := `
        SELECT voice_profile_id, participant_id, owner_user_id, COALESCE(display_name, 'Voice profile'), created_at
        FROM voice_profiles
        WHERE owner_user_id = $1
        ORDER BY created_at DESC
    `
	rows, err := v.e.Query(query, ownerUserId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]persistence.ProfileSummary, 0)
	for rows.Next() {
		var profile persistence.ProfileSummary
		var participantID sql.NullString
		var ownerID sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&profile.VoiceProfileID, &participantID, &ownerID, &profile.DisplayName, &createdAt); err != nil {
			return nil, err
		}
		profile.ParticipantID = nullStringPtr(participantID)
		profile.OwnerUserID = nullStringPtr(ownerID)
		profile.CreatedAt = createdAt
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (v *voiceProfilePostgresRepository) GetProfile(participantId string) ([192]float32, error) {
	query := `
        SELECT embedding
        FROM (
            SELECT vp.embedding, 0 AS priority
            FROM participants p
            JOIN voice_profiles vp ON vp.voice_profile_id = p.voice_profile_id
            WHERE p.participant_id = $1
            UNION ALL
            SELECT embedding, 1 AS priority
            FROM voice_profiles
            WHERE participant_id = $1
        ) profile_candidates
        ORDER BY priority
        LIMIT 1
    `

	var rawEmbedding string
	err := v.e.QueryRow(query, participantId).Scan(&rawEmbedding)
	if err != nil {
		return [192]float32{}, err
	}

	parsed, err := parseVectorLiteral(rawEmbedding)
	if err != nil {
		return [192]float32{}, err
	}

	return parsed, nil
}

func embeddingToVectorLiteral(embeddings [192]float32) string {
	parts := make([]string, 0, len(embeddings))
	for _, v := range embeddings {
		parts = append(parts, strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func parseVectorLiteral(raw string) ([192]float32, error) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")

	items := strings.Split(trimmed, ",")
	if len(items) != 192 {
		return [192]float32{}, fmt.Errorf("invalid embedding length from db: expected 192, got %d", len(items))
	}

	var out [192]float32
	for i := range items {
		f, err := strconv.ParseFloat(strings.TrimSpace(items[i]), 32)
		if err != nil {
			return [192]float32{}, fmt.Errorf("parse embedding at index %d: %w", i, err)
		}
		out[i] = float32(f)
	}

	return out, nil
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}
