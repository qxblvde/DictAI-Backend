package service

import (
	"errors"
	"testing"

	"github.com/Microservices/services/auth-service/internal/domain"
	"github.com/Microservices/services/auth-service/internal/domain/mocks"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func setupTest(t *testing.T) (*AuthService, *mocks.MockUserRepository, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockRepo := mocks.NewMockUserRepository(ctrl)
	svc := NewAuthService(mockRepo, "test-secret")
	return svc, mockRepo, ctrl
}

func TestRegisterUser_Success(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(false, nil)
	mockRepo.EXPECT().CreateUser(gomock.Any()).Return(nil)

	user, err := svc.RegisterUser("test@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password123")); err != nil {
		t.Error("expected password to be hashed correctly")
	}
}

func TestRegisterUser_InvalidPassword_TooShort(t *testing.T) {
	svc, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	_, err := svc.RegisterUser("test@example.com", "short")
	if !errors.Is(err, ErrInvalidPassword) {
		t.Errorf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestRegisterUser_InvalidPassword_Empty(t *testing.T) {
	svc, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	_, err := svc.RegisterUser("test@example.com", "")
	if !errors.Is(err, ErrInvalidPassword) {
		t.Errorf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestRegisterUser_InvalidEmail(t *testing.T) {
	svc, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	_, err := svc.RegisterUser("not-an-email", "password123")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestRegisterUser_EmptyEmail(t *testing.T) {
	svc, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	_, err := svc.RegisterUser("", "password123")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestRegisterUser_EmailAlreadyExists(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(true, nil)

	_, err := svc.RegisterUser("test@example.com", "password123")
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestRegisterUser_CheckUserExistsError(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	dbErr := errors.New("db connection error")
	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(false, dbErr)

	_, err := svc.RegisterUser("test@example.com", "password123")
	if err != dbErr {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestRegisterUser_CreateUserError(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	dbErr := errors.New("db insert error")
	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(false, nil)
	mockRepo.EXPECT().CreateUser(gomock.Any()).Return(dbErr)

	_, err := svc.RegisterUser("test@example.com", "password123")
	if err != dbErr {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestLoginUser_Success(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	hashedPw, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Password: string(hashedPw),
	}

	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(true, nil)
	mockRepo.EXPECT().GetUserByEmail("test@example.com").Return(user, nil)

	token, err := svc.LoginUser("test@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestLoginUser_InvalidEmail(t *testing.T) {
	svc, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	_, err := svc.LoginUser("bad-email", "password123")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestLoginUser_UserNotFound(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(false, nil)

	_, err := svc.LoginUser("test@example.com", "password123")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestLoginUser_WrongPassword(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	hashedPw, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Password: string(hashedPw),
	}

	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(true, nil)
	mockRepo.EXPECT().GetUserByEmail("test@example.com").Return(user, nil)

	_, err := svc.LoginUser("test@example.com", "wrong-password")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestLoginUser_GetUserByEmailError(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	dbErr := errors.New("db error")
	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(true, nil)
	mockRepo.EXPECT().GetUserByEmail("test@example.com").Return(nil, dbErr)

	_, err := svc.LoginUser("test@example.com", "password123")
	if err != dbErr {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	svc, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	token, err := svc.RefreshToken("user-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestChangeEmail_Success(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().CheckUserExists("new@example.com").Return(false, nil)
	mockRepo.EXPECT().UpdateEmail("user-123", "new@example.com").Return(nil)

	err := svc.ChangeEmail("user-123", "new@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestChangeEmail_InvalidEmail(t *testing.T) {
	svc, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	err := svc.ChangeEmail("user-123", "not-valid")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestChangeEmail_EmailAlreadyExists(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().CheckUserExists("taken@example.com").Return(true, nil)

	err := svc.ChangeEmail("user-123", "taken@example.com")
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestChangeEmail_UpdateError(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	dbErr := errors.New("db error")
	mockRepo.EXPECT().CheckUserExists("new@example.com").Return(false, nil)
	mockRepo.EXPECT().UpdateEmail("user-123", "new@example.com").Return(dbErr)

	err := svc.ChangeEmail("user-123", "new@example.com")
	if err != dbErr {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestChangePassword_Success(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().UpdatePassword(gomock.Eq("user-123"), gomock.Any()).Return(nil)

	err := svc.ChangePassword("user-123", "newpassword123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestChangePassword_TooShort(t *testing.T) {
	svc, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	err := svc.ChangePassword("user-123", "short")
	if !errors.Is(err, ErrInvalidPassword) {
		t.Errorf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestChangePassword_Empty(t *testing.T) {
	svc, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	err := svc.ChangePassword("user-123", "")
	if !errors.Is(err, ErrInvalidPassword) {
		t.Errorf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestChangePassword_UpdateError(t *testing.T) {
	svc, mockRepo, ctrl := setupTest(t)
	defer ctrl.Finish()

	dbErr := errors.New("db error")
	mockRepo.EXPECT().UpdatePassword(gomock.Eq("user-123"), gomock.Any()).Return(dbErr)

	err := svc.ChangePassword("user-123", "newpassword123")
	if err != dbErr {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestIsEmailValid(t *testing.T) {
	svc := &AuthService{}
	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user@domain.org", true},
		{"", false},
		{"not-email", false},
		{"@missing.com", false},
	}

	for _, tt := range tests {
		err := svc.isEmailValid(tt.email)
		if tt.valid && err != nil {
			t.Errorf("expected %q to be valid, got error: %v", tt.email, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("expected %q to be invalid, got nil error", tt.email)
		}
	}
}

func TestIsPasswordValid(t *testing.T) {
	svc := &AuthService{}
	tests := []struct {
		password string
		valid    bool
	}{
		{"password123", true},
		{"12345678", true},
		{"1234567", false},
		{"", false},
	}

	for _, tt := range tests {
		err := svc.isPasswordValid(tt.password)
		if tt.valid && err != nil {
			t.Errorf("expected %q to be valid, got error: %v", tt.password, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("expected %q to be invalid, got nil error", tt.password)
		}
	}
}
