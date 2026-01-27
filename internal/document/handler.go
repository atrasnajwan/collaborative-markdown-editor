package document

import (
	"collaborative-markdown-editor/internal/errors"
	// "fmt"
	"io"
	// "log"
	"net/http"
	"strconv"
	// "sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// all origin
		return true
	},
}
type FormCreate struct {
	Title   string `json:"Title" binding:"required"`
}
type FormSave struct {
	Content string `json:"content" binding:"required"`
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
	userID, exists := c.Get("user_id")
	if !exists {
		errors.HandleError(c, errors.ErrUnauthorized(nil).WithMessage("user not found"))
		return
	}

	updateBinary, err := io.ReadAll(c.Request.Body)
	if err != nil || len(updateBinary) == 0 {
		errors.HandleError(c, errors.ErrInvalidInput(err))
		return
	}

	err = h.service.CreateDocumentUpdate(
		docID,
		userID.(uint64),
		updateBinary,	// raw Yjs binary
	)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

func (h *Handler) CreateSnapshot(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := strconv.ParseUint(docIDStr, 10, 64)
	if err != nil {
		errors.HandleError(c, errors.ErrInvalidInput(err))
		return
	}

	snapshot, err := io.ReadAll(c.Request.Body)
	if err != nil || len(snapshot) == 0 {
		errors.HandleError(c, errors.ErrInvalidInput(err))
		return
	}

	err = h.service.CreateSnapshot(c.Request.Context(), docID, snapshot)
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

