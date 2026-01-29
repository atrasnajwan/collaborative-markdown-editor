package document

import (
	"collaborative-markdown-editor/internal/errors"
	"io"
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
	Title   string `json:"Title" binding:"required"`
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
		Title:   form.Title,
	}

	if err := h.service.CreateUserDocument(userID.(uint64), doc); err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, doc)
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

	docs, meta, err := h.service.GetUserDocuments(userID.(uint64), page, pageSize)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": docs, "meta": meta})
}

func (h *Handler) ShowDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docIDUint, err := strconv.ParseUint(docIDStr, 10, 64)
	
	if err != nil {
		errors.HandleError(c, errors.ErrInvalidInput(err).WithMessage("invalid document id"))
		return
	}
	
	userID, exists := c.Get("user_id")
	if !exists {
		errors.HandleError(c, errors.ErrUnauthorized(nil).WithMessage("user not found"))
		return
	}

	doc, err := h.service.GetDocumentByID(docIDUint, userID.(uint64))
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) ShowUserRole(c *gin.Context) {
	docIDStr := c.Param("id")
	docIDUint, err := strconv.ParseUint(docIDStr, 10, 64)
	
	if err != nil {
		errors.HandleError(c, errors.ErrInvalidInput(err).WithMessage("invalid document id"))
		return
	}
	userIDStr := c.Query("user_id")
	userIDUint, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		errors.HandleError(c, errors.ErrUnauthorized(err).WithMessage("user not found"))
		return
	}

	role, err := h.service.FetchUserRole(docIDUint, userIDUint)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK,  gin.H{"role": role})
}

func (h *Handler) ShowDocumentState(c *gin.Context) {
	docIDStr := c.Param("id")
	docIDUint, err := strconv.ParseUint(docIDStr, 10, 64)
	
	if err != nil {
		errors.HandleError(c, errors.ErrInvalidInput(err).WithMessage("invalid document id"))
		return
	}
	
	doc, err := h.service.GetDocumentState(uint64(docIDUint))
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) CreateUpdate(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := strconv.ParseUint(docIDStr, 10, 64)
	if err != nil {
		errors.HandleError(c, errors.ErrInvalidInput(err))
		return
	}

	userID, err := strconv.ParseUint(
		c.GetHeader("X-User-Id"),
		10, 64,
	)
	if err != nil {
		errors.HandleError(c, errors.ErrInvalidInput(err))
		return
	}

	updateBinary, err := io.ReadAll(c.Request.Body)
	if err != nil || len(updateBinary) == 0 {
		errors.HandleError(c, errors.ErrInvalidInput(err))
		return
	}

	err = h.service.CreateDocumentUpdate(
		c.Request.Context(),
		docID,
		userID,
		updateBinary,	// raw Yjs binary
	)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) ListCollaborators(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		errors.HandleError(c, errors.ErrUnauthorized(nil).WithMessage("user not found"))
		return
	}

	// viewer not allowed to
	role, err := h.service.FetchUserRole(docID, userID.(uint64))
	if err != nil {
		errors.HandleError(c, errors.ErrInternalServer(err))
		return
	}
	if role == "viwer" {
		errors.HandleError(c, errors.ErrForbidden(err))
			return
	}

	result, err := h.service.ListCollaborators(
		c.Request.Context(),
		docID,
		userID.(uint64),
	)
	if err != nil {
		errors.HandleError(c, errors.ErrInternalServer(err))
		return
	}

	c.JSON(http.StatusOK, result)
}
