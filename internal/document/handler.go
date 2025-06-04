package document

import (
	"collaborative-markdown-editor/internal/errors"
	"net/http"
	"strconv"

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

func (h *Handler) ShowUserDocuments(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		errors.HandleError(c, errors.ErrUnauthorized(nil).WithMessage("user not found"))
		return
	}

	// Parse query params with defaults
	page := 1
	pageSize := 10
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("per_page"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	docs, meta, err := h.service.GetUserDocuments(userID.(uint), page, pageSize)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": docs, "meta": meta})
}
