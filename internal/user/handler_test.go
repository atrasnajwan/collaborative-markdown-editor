package user

import (
	"bytes"
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/middleware"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mock implementation of the Service interface
type MockService struct {
	mock.Mock
}

func (m *MockService) Register(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockService) UpdateUser(ctx context.Context, userID uint64, req UpdateProfileRequest) (domain.SafeUser, error) {
	args := m.Called(ctx, userID, req)
	return args.Get(0).(domain.SafeUser), args.Error(1)
}

func (m *MockService) ChangePassword(ctx context.Context, userID uint64, req ChangePasswordRequest) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func (m *MockService) Login(ctx context.Context, email, password string) (*domain.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockService) Logout(ctx context.Context, userID uint64) {
	m.Called(ctx, userID)
}

func (m *MockService) GetUserByID(ctx context.Context, id uint64) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockService) DeactivateUser(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockService) SearchUsers(ctx context.Context, query string) ([]domain.SafeUser, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return []domain.SafeUser{}, args.Error(1)
	}
	return args.Get(0).([]domain.SafeUser), args.Error(1)
}

func setupRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	return router
}

func TestRegister_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("Register", mock.Anything, mock.MatchedBy(func(user *domain.User) bool {
		return user.Name == "Atras Najwan" &&
			user.Email == "atras@example.com" &&
			user.Password == "password123"
	})).Return(nil).Run(func(args mock.Arguments) {
		user := args.Get(1).(*domain.User)
		user.ID = 1
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
	})

	router.POST("/register", func(c *gin.Context) {
		handler.Register(c)
	})

	payload := FormRegister{
		Name:     "Atras Najwan",
		Email:    "atras@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotNil(t, response["user"])
	mockService.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	user := &domain.User{
		ID:           1,
		Name:         "Atras Najwan",
		Email:        "atras@example.com",
		IsActive:     true,
		TokenVersion: 0,
	}

	mockService.On("Login", mock.Anything, "atras@example.com", "password123").Return(user, nil)

	router.POST("/login", func(c *gin.Context) {
		handler.Login(c)
	})

	payload := FormLogin{
		Email:    "atras@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotNil(t, response["access_token"])
	assert.NotNil(t, response["user"])
	mockService.AssertExpectations(t)
}

func TestLogin_UserNotFound(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("Login", mock.Anything, "nothing@example.com", "password123").
		Return(nil, assert.AnError)

	router.POST("/login", func(c *gin.Context) {
		handler.Login(c)
	})

	payload := FormLogin{
		Email:    "nothing@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestLogout_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("Logout", mock.Anything, uint64(1)).Return()

	router.DELETE("/logout", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.Logout(c)
	})

	req := httptest.NewRequest("DELETE", "/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

func TestLogout_NoToken(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("Logout", mock.Anything, uint64(1)).Return()

	router.DELETE("/logout", func(c *gin.Context) {
		// Set a default user_id for logout, real middleware would handle this
		if _, exists := c.Get("user_id"); !exists {
			c.Set("user_id", uint64(1))
		}
		handler.Logout(c)
	})

	req := httptest.NewRequest("DELETE", "/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestGetProfile_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	user := &domain.User{
		ID:        1,
		Name:      "Atras Najwan",
		Email:     "atras@example.com",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockService.On("GetUserByID", mock.Anything, uint64(1)).Return(user, nil)

	router.GET("/profile", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.GetProfile(c)
	})

	req := httptest.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response domain.SafeUser
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Atras Najwan", response.Name)
	assert.Equal(t, "atras@example.com", response.Email)
	mockService.AssertExpectations(t)
}

func TestGetProfile_UserNotFound(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("GetUserByID", mock.Anything, uint64(999)).Return(nil, assert.AnError)

	router.GET("/profile", func(c *gin.Context) {
		c.Set("user_id", uint64(999))
		handler.GetProfile(c)
	})

	req := httptest.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestSearchUsers_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	results := []domain.SafeUser{
		{
			ID:    2,
			Name:  "Jane Foster",
			Email: "foster@example.com",
		},
		{
			ID:    3,
			Name:  "John Krasinsky",
			Email: "john.krasinsky@example.com",
		},
	}

	mockService.On("SearchUsers", mock.Anything, "john").Return(results, nil)

	router.GET("/search", func(c *gin.Context) {
		handler.SearchUsers(c)
	})

	req := httptest.NewRequest("GET", "/search?q=john", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response []domain.SafeUser
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 2, len(response))
	mockService.AssertExpectations(t)
}

func TestSearchUsers_NoQuery(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("SearchUsers", mock.Anything, "").Return([]domain.SafeUser{}, nil)

	router.GET("/search", func(c *gin.Context) {
		handler.SearchUsers(c)
	})

	req := httptest.NewRequest("GET", "/search", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response []domain.SafeUser
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 0, len(response))
	mockService.AssertExpectations(t)
}

func TestSearchUsers_EmptyResult(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("SearchUsers", mock.Anything, "nothing").Return([]domain.SafeUser{}, nil)

	router.GET("/search", func(c *gin.Context) {
		handler.SearchUsers(c)
	})

	req := httptest.NewRequest("GET", "/search?q=nothing", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response []domain.SafeUser
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 0, len(response))
	mockService.AssertExpectations(t)
}
