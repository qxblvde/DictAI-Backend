package service

import (
	"net/mail"
	"time"

	"github.com/Microservices/services/auth-service/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	rp        domain.UserRepository
	jwtSecret string
}

func NewAuthService(rp domain.UserRepository, secret string) *AuthService {
	return &AuthService{rp: rp, jwtSecret: secret}
}

func (svc *AuthService) RegisterUser(email, password string) (*domain.User, error) {
	err := svc.isPasswordValid(password)
	if err != nil {
		return nil, err
	}

	err = svc.isEmailValid(email)
	if err != nil {
		return nil, err
	}

	exists, err := svc.rp.CheckUserExists(email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{Email: email, Password: string(hashedPassword)}
	if err := svc.rp.CreateUser(user); err != nil {
		return nil, err
	}

	return user, err
}

func (svc *AuthService) LoginUser(email, password string) (string, error) {
	err := svc.isEmailValid(email)
	if err != nil {
		return "", err
	}

	exists, err := svc.rp.CheckUserExists(email)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrUserNotFound
	}

	user, err := svc.rp.GetUserByEmail(email)
	if err != nil {
		return "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", ErrWrongPassword
	}

	token, err := svc.createToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (svc *AuthService) RefreshToken(userID string) (string, error) {
	return svc.createToken(userID)
}

func (svc *AuthService) ChangeEmail(userID, newEmail string) error {
	err := svc.isEmailValid(newEmail)
	if err != nil {
		return err
	}

	exists, err := svc.rp.CheckUserExists(newEmail)
	if err != nil {
		return err
	}
	if exists {
		return ErrEmailAlreadyExists
	}

	user := &domain.User{ID: userID, Email: newEmail}
	err = svc.rp.UpdateEmail(user.ID, user.Email)
	if err != nil {
		return err
	}
	return nil
}

func (svc *AuthService) ChangePassword(userID, newPassword string) error {
	err := svc.isPasswordValid(newPassword)
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &domain.User{ID: userID, Password: string(hashedPassword)}
	err = svc.rp.UpdatePassword(user.ID, user.Password)
	if err != nil {
		return err
	}
	return nil
}

func (svc *AuthService) createToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	tokenString, err := token.SignedString([]byte(svc.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (svc *AuthService) isEmailValid(email string) error {
	if _, err := mail.ParseAddress(email); err != nil || email == "" {
		return ErrInvalidEmail
	}

	return nil
}

func (svc *AuthService) isPasswordValid(password string) error {
	if password == "" || len(password) < 8 {
		return ErrInvalidPassword
	}

	return nil
}

func (svc *AuthService) GetUserByID(userID string) (*domain.User, error) {
	return svc.rp.GetUserByID(userID)
}
