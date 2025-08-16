package document

import (
	"collaborative-markdown-editor/internal/errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

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
		Title:   form.Title,
		Content: &form.Content,
	}

	if err := h.service.CreateUserDocument(userID.(uint), doc); err != nil {
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

	docs, meta, err := h.service.GetUserDocuments(userID.(uint), page, pageSize)
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
	
	doc, err := h.service.GetDocumentByID(uint(docIDUint))
	if err != nil {
		errors.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, doc)
}

// New WebSocket handler for editing documents
func (h *Handler) EditDocument(c *gin.Context) {
	wsCon, err := upgrader.Upgrade(c.Writer, c.Request, nil); 
	if err != nil {
		errors.HandleError(c, errors.ErrInternalServer(err).WithMessage("Error upgrading to WebSocket"))
		return
	}

	defer wsCon.Close()

	docIDStr := c.Param("id")
	docIDUint, err := strconv.ParseUint(docIDStr, 10, 64)	
	if err != nil {
		errors.HandleError(c, errors.ErrInvalidInput(err).WithMessage("invalid document id"))
		return
	}

	userName, exist := c.Get("user_name")
	if !exist {
		log.Println("Username is not define")
	}

	_, err = h.service.GetDocumentByID(uint(docIDUint))
	if err != nil {
		errors.HandleError(c, errors.ErrNotFound(err).WithMessage("document not found"))
		return
	}

	fmt.Println("docID", docIDStr)

	for {
		_, _, err := wsCon.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}
		// log.Println(messageType, p)
	}

	fmt.Printf("%s disconnected\n", userName)
}
