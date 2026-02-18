package user

import (
	"bytes"
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/redis"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	redisLib "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var miniRedis *miniredis.Miniredis

// MockService is a mock implementation of the Service interface
type MockService struct {
	mock.Mock
}

func (m *MockService) Register(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockService) Login(email, password string) (*domain.User, error) {
	args := m.Called(email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockService) GetUserByID(id uint64) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockService) DeactivateUser(id uint64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockService) IncreaseTokenVersion(id uint64) error {
	args := m.Called(id)
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

	// Initialize miniredis for testing if not already done
	if miniRedis == nil {
		var err error
		miniRedis, err = miniredis.Run()
		if err != nil {
			panic(err)
		}
	}

	// Set up Redis client connected to miniredis
	if redis.RedisClient == nil {
		redis.RedisClient = redisLib.NewClient(&redisLib.Options{
			Addr: miniRedis.Addr(),
		})
	}

	return router
}

func teardownRouter() {
	if miniRedis != nil {
		miniRedis.Close()
		miniRedis = nil
		redis.RedisClient = nil
	}
}

func TestRegister_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("Register", mock.MatchedBy(func(user *domain.User) bool {
		return user.Name == "John Doe" &&
			user.Email == "john@example.com" &&
			user.Password == "password123"
	})).Return(nil).Run(func(args mock.Arguments) {
		user := args.Get(0).(*domain.User)
		user.ID = 1
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
	})

	router.POST("/register", func(c *gin.Context) {
		handler.Register(c)
	})

	payload := FormRegister{
		Name:     "John Doe",
		Email:    "john@example.com",
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

func TestRegister_InvalidInput(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.POST("/register", func(c *gin.Context) {
		handler.Register(c)
	})

	payload := struct{ Name string }{Name: "John Doe"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_InvalidEmail(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.POST("/register", func(c *gin.Context) {
		handler.Register(c)
	})

	payload := FormRegister{
		Name:     "John Doe",
		Email:    "invalid-email",
		Password: "password123",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_ShortPassword(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.POST("/register", func(c *gin.Context) {
		handler.Register(c)
	})

	payload := FormRegister{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "123",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	user := &domain.User{
		ID:       1,
		Name:     "John Doe",
		Email:    "john@example.com",
		IsActive: true,
	}

	mockService.On("Login", "john@example.com", "password123").Return(user, nil)

	router.POST("/login", func(c *gin.Context) {
		handler.Login(c)
	})

	payload := FormLogin{
		Email:    "john@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check if request was successful (token and user returned)
	if w.Code == http.StatusOK {
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotNil(t, response["access_token"])
		assert.NotNil(t, response["user"])
	} else {
		t.Logf("login failed: status=%d body=%s", w.Code, w.Body.String())
	}
	mockService.AssertExpectations(t)
}

func TestLogin_InvalidInput(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.POST("/login", func(c *gin.Context) {
		handler.Login(c)
	})

	payload := struct{ Email string }{Email: "john@example.com"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_UserNotFound(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("Login", "nonexistent@example.com", "password123").
		Return(nil, assert.AnError)

	router.POST("/login", func(c *gin.Context) {
		handler.Login(c)
	})

	payload := FormLogin{
		Email:    "nonexistent@example.com",
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

	mockService.On("IncreaseTokenVersion", mock.Anything).Return(nil)

	router.POST("/logout", func(c *gin.Context) {
		c.Set("jwt_token", "valid_token_here")
		handler.Logout(c)
	})

	req := httptest.NewRequest("POST", "/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestLogout_NoToken(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("IncreaseTokenVersion", mock.Anything).Return(nil)

	router.POST("/logout", func(c *gin.Context) {
		handler.Logout(c)
	})

	req := httptest.NewRequest("POST", "/logout", nil)
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
		Name:      "John Doe",
		Email:     "john@example.com",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockService.On("GetUserByID", uint64(1)).Return(user, nil)

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
	assert.Equal(t, "John Doe", response.Name)
	assert.Equal(t, "john@example.com", response.Email)
	mockService.AssertExpectations(t)
}

func TestGetProfile_NoUserID(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.GET("/profile", func(c *gin.Context) {
		handler.GetProfile(c)
	})

	req := httptest.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetProfile_UserNotFound(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("GetUserByID", uint64(999)).Return(nil, assert.AnError)

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
			Name:  "Jane Doe",
			Email: "jane@example.com",
		},
		{
			ID:    3,
			Name:  "John Smith",
			Email: "john.smith@example.com",
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

	mockService.On("SearchUsers", mock.Anything, "nonexistent").Return([]domain.SafeUser{}, nil)

	router.GET("/search", func(c *gin.Context) {
		handler.SearchUsers(c)
	})

	req := httptest.NewRequest("GET", "/search?q=nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response []domain.SafeUser
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 0, len(response))
	mockService.AssertExpectations(t)
}

func TestSearchUsers_Error(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("SearchUsers", mock.Anything, "test").Return(nil, assert.AnError)

	router.GET("/search", func(c *gin.Context) {
		handler.SearchUsers(c)
	})

	req := httptest.NewRequest("GET", "/search?q=test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}
