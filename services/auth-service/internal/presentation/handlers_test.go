package presentation

import (
	"bytes"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Microservices/services/auth-service/internal/domain"
	"github.com/Microservices/services/auth-service/internal/domain/mocks"
	"github.com/Microservices/services/auth-service/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter(t *testing.T) (*gin.Engine, *mocks.MockUserRepository, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockRepo := mocks.NewMockUserRepository(ctrl)
	svc := service.NewAuthService(mockRepo, "test-secret")
	router := NewRouter(svc, slog.Default())
	return router, mockRepo, ctrl
}

func performRequest(router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		err := json.NewEncoder(&buf).Encode(body)
		if err != nil {
			log.Fatalf("failed to encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestRegister_Success(t *testing.T) {
	router, mockRepo, ctrl := setupRouter(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(false, nil)
	mockRepo.EXPECT().CreateUser(gomock.Any()).Return(nil)

	w := performRequest(router, "POST", "/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_InvalidJSON(t *testing.T) {
	router, _, ctrl := setupRouter(t)
	defer ctrl.Finish()

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	router, _, ctrl := setupRouter(t)
	defer ctrl.Finish()

	w := performRequest(router, "POST", "/auth/register", map[string]string{
		"email":    "not-email",
		"password": "password123",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	router, _, ctrl := setupRouter(t)
	defer ctrl.Finish()

	w := performRequest(router, "POST", "/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": "short",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	router, mockRepo, ctrl := setupRouter(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(true, nil)

	w := performRequest(router, "POST", "/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_Success(t *testing.T) {
	router, mockRepo, ctrl := setupRouter(t)
	defer ctrl.Finish()

	hashedPw, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{ID: "user-123", Email: "test@example.com", Password: string(hashedPw)}

	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(true, nil)
	mockRepo.EXPECT().GetUserByEmail("test@example.com").Return(user, nil)

	w := performRequest(router, "POST", "/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["token"] == "" {
		t.Error("expected token in response")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	router, mockRepo, ctrl := setupRouter(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(false, nil)

	w := performRequest(router, "POST", "/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	router, mockRepo, ctrl := setupRouter(t)
	defer ctrl.Finish()

	hashedPw, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	user := &domain.User{ID: "user-123", Email: "test@example.com", Password: string(hashedPw)}

	mockRepo.EXPECT().CheckUserExists("test@example.com").Return(true, nil)
	mockRepo.EXPECT().GetUserByEmail("test@example.com").Return(user, nil)

	w := performRequest(router, "POST", "/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "wrong-password",
	})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRefresh_Success(t *testing.T) {
	router, _, ctrl := setupRouter(t)
	defer ctrl.Finish()

	w := performRequest(router, "POST", "/auth/refresh", map[string]string{
		"user_id": "user-123",
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["token"] == "" {
		t.Error("expected token in response")
	}
}

func TestChangeEmail_Success(t *testing.T) {
	router, mockRepo, ctrl := setupRouter(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().CheckUserExists("new@example.com").Return(false, nil)
	mockRepo.EXPECT().UpdateEmail("user-123", "new@example.com").Return(nil)

	w := performRequest(router, "POST", "/auth/change-email", map[string]string{
		"user_id":   "user-123",
		"new_email": "new@example.com",
	})

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChangeEmail_InvalidEmail(t *testing.T) {
	router, _, ctrl := setupRouter(t)
	defer ctrl.Finish()

	w := performRequest(router, "POST", "/auth/change-email", map[string]string{
		"user_id":   "user-123",
		"new_email": "invalid",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChangePassword_Success(t *testing.T) {
	router, mockRepo, ctrl := setupRouter(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().UpdatePassword(gomock.Eq("user-123"), gomock.Any()).Return(nil)

	w := performRequest(router, "POST", "/auth/change-password", map[string]string{
		"user_id":      "user-123",
		"new_password": "newpassword123",
	})

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChangePassword_TooShort(t *testing.T) {
	router, _, ctrl := setupRouter(t)
	defer ctrl.Finish()

	w := performRequest(router, "POST", "/auth/change-password", map[string]string{
		"user_id":      "user-123",
		"new_password": "short",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
