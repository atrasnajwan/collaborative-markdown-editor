package user

import (
	"collaborative-markdown-editor/auth"
	"collaborative-markdown-editor/internal/errors"
	"collaborative-markdown-editor/redis"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for users
type Handler struct {
	service Service
}

// NewHandler creates a new user handler
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// FormLogin represents login form data
type FormLogin struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// FormRegister represents registration form data
type FormRegister struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var form FormRegister
	if err := c.ShouldBindJSON(&form); err != nil {
		errors.HandleError(c, errors.ErrInvalidInput(err))
		return
	}

	user := &User{
		Name:     form.Name,
		Email:    form.Email,
		Password: form.Password,
		IsActive: true,
	}

	if err := h.service.Register(user); err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user.ToSafeUser()})
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var form FormLogin
	if err := c.ShouldBindJSON(&form); err != nil {
		errors.HandleError(c, errors.ErrInvalidInput(err))
		return
	}

	user, err := h.service.Login(form.Email, form.Password)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	// Generate JWT token
	token, err := auth.GenerateJWT(user.ID)
	if err != nil {
		errors.HandleError(c, errors.ErrInternalServer(err))
		return
	}

	// Store in Redis with expiration
	redis.RedisClient.Set(redis.Ctx, token, user.ID, time.Hour*24*3)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user.ToSafeUser(),
	})
}

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	jwtToken, exists := c.Get("jwt_token")
	if exists {
		redis.RedisClient.Del(redis.Ctx, jwtToken.(string))
	}

	c.Status(http.StatusNoContent)
}

// GetProfile handles getting the current user's profile
func (h *Handler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		errors.HandleError(c, errors.ErrUnauthorized(nil).WithMessage("user not found"))
		return
	}

	user, err := h.service.GetUserByID(userID.(uint64))
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, user.ToSafeUser())
}
