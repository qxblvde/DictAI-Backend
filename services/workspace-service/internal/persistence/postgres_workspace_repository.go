package persistence

import (
	"database/sql"
	"fmt"

	"github.com/DictAI/Microservices/services/workspace-service/internal/domain"
)

type PostgresWorkspaceRepository struct {
	db *sql.DB
}

func (repository *PostgresWorkspaceRepository) CreateWorkspace(workspace *domain.Workspace) error {
	query := `
		INSERT INTO workspaces (owner_user_id, name)
		VALUES ($1, $2)
		RETURNING workspace_id, created_at
    `
	return repository.db.QueryRow(query, workspace.OwnerID, workspace.Name).Scan(&workspace.WorkspaceID, &workspace.CreatedAt)
}

func (repository *PostgresWorkspaceRepository) FindWorkspace(workspaceID string) (*domain.Workspace, error) {
	var workspace domain.Workspace

	query := `
		SELECT workspace_id, owner_user_id, name, created_at 
		FROM workspaces 
		WHERE workspace_id = $1
    `

	err := repository.db.QueryRow(query, workspaceID).Scan(&workspace.WorkspaceID, &workspace.OwnerID, &workspace.Name, &workspace.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &workspace, nil
}

func (repository *PostgresWorkspaceRepository) GetByOwnerID(ownerID string, limit, offset int) ([]*domain.Workspace, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM workspaces WHERE owner_user_id = $1`
	if err := repository.db.QueryRow(countQuery, ownerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count workspaces: %w", err)
	}

	query := `
		SELECT workspace_id, owner_user_id, name, created_at
		FROM workspaces
		WHERE owner_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := repository.db.Query(query, ownerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []*domain.Workspace
	for rows.Next() {
		var w domain.Workspace
		if err := rows.Scan(&w.WorkspaceID, &w.OwnerID, &w.Name, &w.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan workspace row: %w", err)
		}
		workspaces = append(workspaces, &w)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate workspace rows: %w", err)
	}

	return workspaces, total, nil
}

func (repository *PostgresWorkspaceRepository) UpdateWorkspace(workspaceID, name string) (*domain.Workspace, error) {
	query := `
		UPDATE workspaces SET name = $1
		WHERE workspace_id = $2
		RETURNING workspace_id, owner_user_id, name, created_at
	`
	var w domain.Workspace
	err := repository.db.QueryRow(query, name, workspaceID).Scan(&w.WorkspaceID, &w.OwnerID, &w.Name, &w.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("update workspace: %w", err)
	}
	return &w, nil
}

func (repository *PostgresWorkspaceRepository) DeleteWorkspace(workspaceID string) error {
	result, err := repository.db.Exec(`DELETE FROM workspaces WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
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

func NewPostgresWorkspaceRepository(db *sql.DB) *PostgresWorkspaceRepository {
	return &PostgresWorkspaceRepository{db: db}
}
