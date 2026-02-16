package user

import (
	"collaborative-markdown-editor/auth"
	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/errors"
	"log"
	"net/http"

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

	user := &domain.User{
		Name:     form.Name,
		Email:    form.Email,
		Password: form.Password,
		IsActive: true,
	}

	if err := h.service.Register(user); err != nil {
		errors.HandleError(c, errors.ErrUnprocessableEntity(err))
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
		errors.HandleError(c, errors.ErrUnprocessableEntity(err))
		return
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.TokenVersion)
	if err != nil {
		errors.HandleError(c, errors.ErrInternalServer(err))
		return
	}
	refreshToken, err := auth.GenerateRefreshToken(user.ID, user.TokenVersion)
	if err != nil {
		errors.HandleError(c, errors.ErrInternalServer(err))
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
		errors.HandleError(c, errors.ErrUnauthorized(err))
		return
	}

	token, err := auth.VerifyJWT(refreshToken)
	if err != nil {
		errors.HandleError(c, errors.ErrUnauthorized(err).WithMessage("Invalid token or expired!"))
		return
	}

	userID, tokenVersion, err := auth.GetDataFromToken(token) 
	if err != nil {
		errors.HandleError(c, errors.ErrUnauthorized(err).WithMessage("Invalid token"))
		return
	}

	user, err := h.service.GetUserByID(userID)
	if err != nil {
		errors.HandleError(c, errors.ErrUnauthorized(err).WithMessage("User not found"))
		return
	}

	// Check token version
	if user.TokenVersion != tokenVersion {
		errors.HandleError(c, errors.ErrUnauthorized(nil).WithMessage("Invalid token!"))
		return
	}

	// Issue new access token
	newAccessToken, err := auth.GenerateAccessToken(user.ID, user.TokenVersion)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": newAccessToken,
	})
}


// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	userID := c.GetUint("user_id")

	err := h.service.IncreaseTokenVersion(uint64(userID))
	if err != nil {
		log.Printf("%v\n", err.Error())
	}
	// Clear refresh cookie
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)
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

func (h *Handler) SearchUsers(c *gin.Context) {
	query := c.Query("q")

	users, err := h.service.SearchUsers(
		c.Request.Context(),
		query,
	)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, users)
}
