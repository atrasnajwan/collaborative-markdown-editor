package user

import (
	"collaborative-markdown-editor/internal/auth"
	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/errors"
	"collaborative-markdown-editor/redis"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for users
type Handler struct {
	service Service
	cache *redis.Cache
}

// NewHandler creates a new user handler
func NewHandler(service Service, cache *redis.Cache) *Handler {
	return &Handler{service: service, cache: cache}
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
		c.Error(errors.NewValidationError(err))
		return
	}

	user := &domain.User{
		Name:     form.Name,
		Email:    form.Email,
		Password: form.Password,
		IsActive: true,
	}

	if err := h.service.Register(user); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user.ToSafeUser()})
}

type UpdateProfileRequest struct {
    Name     *string `json:"name" binding:"omitempty,min=2"`
    Email    *string `json:"email" binding:"omitempty,email"`
}

func (h *Handler) UpdateProfile(c *gin.Context) {
    userID, _ := c.Get("user_id")

    var req UpdateProfileRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.Error(errors.NewValidationError(err))
        return
    }

    user, err := h.service.UpdateUser(c.Request.Context(), userID.(uint64), req)
    if err != nil {
        c.Error(err)
        return
    }

    c.JSON(http.StatusOK, user)
}

type ChangePasswordRequest struct {
    CurrentPassword string `json:"current_password" binding:"required"`
    NewPassword     string `json:"new_password" binding:"required,min=8"`
}

func (h *Handler) ChangePassword(c *gin.Context) {
    userID, _ := c.Get("user_id")

    var req ChangePasswordRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.Error(errors.NewValidationError(err))
        return
    }

    err := h.service.ChangePassword(c.Request.Context(), userID.(uint64), req)
    if err != nil {
        c.Error(err)
        return
    }

    c.Status(http.StatusNoContent)
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var form FormLogin
	if err := c.ShouldBindJSON(&form); err != nil {
		c.Error(errors.NewValidationError(err))
		return
	}

	user, err := h.service.Login(form.Email, form.Password)
	if err != nil {
		c.Error(err)
		return
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.TokenVersion)
	if err != nil {
		c.Error(err)
		return
	}
	refreshToken, err := auth.GenerateRefreshToken(user.ID, user.TokenVersion)
	if err != nil {
		c.Error(err)
		return
	}

	// Set refresh token as HttpOnly cookie
	c.SetCookie(
		"refresh_token",
		refreshToken,
		7*24*3600,
		"/",
		"",
		config.AppConfig.Environment == "production",  // Secure
		true,  // HttpOnly
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token":	accessToken,
		"user": 		user.ToSafeUser(),
	})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.Error(errors.NewValidationError(err))
		return
	}

	token, err := auth.VerifyJWT(refreshToken)
	if err != nil {
		c.Error(errors.Unauthorized("Invalid token or expired!", err))
		return
	}

	userID, tokenVersion, err := auth.GetDataFromToken(token) 
	if err != nil {
		c.Error(errors.Unauthorized("Invalid token!", err))
		return
	}

	user, err := h.service.GetUserByID(userID)
	if err != nil {
		c.Error(errors.UnprocessableEntity("User not found!", err))
		return
	}

	// Check token version
	if user.TokenVersion != tokenVersion {
		c.Error(errors.Unauthorized("Session expired!", nil))
		return
	}

	// Issue new access token
	newAccessToken, err := auth.GenerateAccessToken(user.ID, user.TokenVersion)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": newAccessToken,
	})
}


// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	userID, _ := c.Get("user_id")

	err := h.service.IncreaseTokenVersion(userID.(uint64))
	if err != nil {
		log.Printf("%v\n", err.Error())
	}
	// Clear refresh cookie
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)
	// invalidate cache/redis
	cacheKey := fmt.Sprintf("user:version:%d", userID)
	h.cache.Invalidate(c.Request.Context(), cacheKey)
	
	c.Status(http.StatusNoContent)
}

// GetProfile handles getting the current user's profile
func (h *Handler) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.service.GetUserByID(userID.(uint64))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, user.ToSafeUser())
}

func (h *Handler) SearchUsers(c *gin.Context) {
	query := c.Query("q")

	users, err := h.service.SearchUsers(
		c.Request.Context(),
		query,
	)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, users)
}
