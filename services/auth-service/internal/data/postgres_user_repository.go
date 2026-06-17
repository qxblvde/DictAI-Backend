package data

import (
	"github.com/Microservices/services/auth-service/internal/domain"
)

type PostgresUserRepository struct {
	db DBExecutor
}

func (p *PostgresUserRepository) CreateUser(user *domain.User) error {
	query := `
        INSERT INTO users (email, password_hash)
        VALUES ($1, $2)
        RETURNING user_id, created_at
    `
	return p.db.QueryRow(query, user.Email, user.Password).Scan(&user.ID, &user.CreatedAt)
}

func (p *PostgresUserRepository) UpdateEmail(userID, newEmail string) error {
	query := `UPDATE users SET email = $1 WHERE user_id = $2`
	_, err := p.db.Exec(query, newEmail, userID)
	return err
}

func (p *PostgresUserRepository) UpdatePassword(userID, newPasswordHash string) error {
	query := `UPDATE users SET password_hash = $1 WHERE user_id = $2`
	_, err := p.db.Exec(query, newPasswordHash, userID)
	return err
}

func (p *PostgresUserRepository) DeleteUser(userID string) error {
	query := `DELETE FROM users WHERE user_id=$1`
	_, err := p.db.Exec(query, userID)
	return err
}

func (p *PostgresUserRepository) GetUserByID(userID string) (*domain.User, error) {
	var user domain.User
	query := `SELECT user_id, email, password_hash, created_at FROM users WHERE user_id=$1`
	err := p.db.QueryRow(query, userID).Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (p *PostgresUserRepository) GetUserByEmail(email string) (*domain.User, error) {
	var user domain.User
	query := `SELECT user_id, email, password_hash, created_at FROM users WHERE email=$1`
	err := p.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (p *PostgresUserRepository) CheckUserExists(email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`
	err := p.db.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func NewPostgresUserRepository(dbExecutor DBExecutor) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: dbExecutor,
	}
}
