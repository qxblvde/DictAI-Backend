package domain

type UserRepository interface {
	CreateUser(user *User) error
	UpdateEmail(userID, newEmail string) error
	UpdatePassword(userID, newPasswordHash string) error
	DeleteUser(userID string) error
	GetUserByID(userID string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	CheckUserExists(email string) (bool, error)
}
