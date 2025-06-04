package document

import (
	"collaborative-markdown-editor/internal/errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type FormCreate struct {
	Name    string `json:"name" binding:"required"`
	Content string `json:"content"`
}

func (h *Handler) Create(c *gin.Context) {
	var form FormCreate
	if err := c.ShouldBindJSON(&form); err != nil {
		errors.HandleError(c, errors.ErrInvalidInput(err))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		errors.HandleError(c, errors.ErrUnauthorized(nil).WithMessage("user not found"))
		return
	}

	doc := &Document{
		Name:    form.Name,
		Content: &form.Content,
	}

	if err := h.service.CreateUserDocument(userID.(uint), doc); err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"document": doc})
}
