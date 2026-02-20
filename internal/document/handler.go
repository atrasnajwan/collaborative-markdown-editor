package document

import (
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/errors"
	"collaborative-markdown-editor/internal/utils"
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

type CreateOrRenameRequest struct {
	Title   string `json:"title" binding:"required,min=1,max=255"`
}

func (h *Handler) Create(c *gin.Context) {
	var form CreateOrRenameRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.Error(errors.NewValidationError(err))
		return
	}

	userID, _ := c.Get("user_id")

	doc := &domain.Document{
		Title:   form.Title,
	}

	if err := h.service.CreateUserDocument(c.Request.Context(), userID.(uint64), doc); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, doc)
}

func (h *Handler) Rename(c *gin.Context) {
    docIDStr := c.Param("id")
    docID, err := strconv.ParseUint(docIDStr, 10, 64)
    if err != nil {
        c.Error(err)
        return
    }

    userID, _ := c.Get("user_id")

    var input CreateOrRenameRequest
    if err := c.ShouldBindJSON(&input); err != nil {
        c.Error(errors.NewValidationError(err))
        return
    }

   	doc, err := h.service.RenameDocument(c.Request.Context(), docID, userID.(uint64), input.Title)
    if err != nil {
        c.Error(err)
        return
    }

    c.JSON(200, doc)
}

func (h *Handler) ShowUserDocuments(c *gin.Context) {
	userID, _ := c.Get("user_id")

	page, pageSize := utils.GetPaginationParams(c)
	result, err := h.service.GetUserDocuments(c.Request.Context(), userID.(uint64), page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}


func (h *Handler) ShowSharedDocuments(c *gin.Context) {
	userID, _ := c.Get("user_id")

	page, pageSize := utils.GetPaginationParams(c)
	docs, meta, err := h.service.GetSharedDocuments(c.Request.Context(), userID.(uint64), page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": docs, "meta": meta})
}

func (h *Handler) ShowDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docIDUint, err := strconv.ParseUint(docIDStr, 10, 64)
	if err != nil {
		c.Error(err)
		return
	}
	
	userID, _ := c.Get("user_id")

	doc, err := h.service.GetDocumentByID(c.Request.Context(), docIDUint, userID.(uint64))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) ShowUserRole(c *gin.Context) {
	docIDStr := c.Param("id")
	docIDUint, err := strconv.ParseUint(docIDStr, 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	userIDStr := c.Query("user_id")
	userIDUint, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	role, err := h.service.FetchUserRole(c.Request.Context(), docIDUint, userIDUint)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK,  gin.H{"role": role})
}

func (h *Handler) ShowDocumentState(c *gin.Context) {
	docIDStr := c.Param("id")
	docIDUint, err := strconv.ParseUint(docIDStr, 10, 64)
	if err != nil {
		c.Error(err)
		return
	}
	
	doc, err := h.service.GetDocumentState(c.Request.Context(), uint64(docIDUint))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) CreateUpdate(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := strconv.ParseUint(docIDStr, 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	userID, err := strconv.ParseUint(
		c.GetHeader("X-User-Id"),
		10, 64,
	)
	if err != nil {
		c.Error(errors.UnprocessableEntity("X-User-Id not found in header", err))
		return
	}

	updateBinary, err := io.ReadAll(c.Request.Body)
	if err != nil || len(updateBinary) == 0 {
		c.Error(errors.UnprocessableEntity("Can't read update binary or empty update", err))
		return
	}

	err = h.service.CreateDocumentUpdate(
		c.Request.Context(),
		docID,
		userID,
		updateBinary,	// raw Yjs binary
	)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) CreateSnapshot(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := strconv.ParseUint(docIDStr, 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	snapshotBinary, err := io.ReadAll(c.Request.Body)
	if err != nil || len(snapshotBinary) == 0 {
		c.Error(errors.UnprocessableEntity("Can't read snapshot binary or empty snapshot", err))
		return
	}

	err = h.service.CreateDocumentSnapshot(
		c.Request.Context(),
		docID,
		snapshotBinary,	// raw Yjs binary
	)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) ListCollaborators(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	userID, _ := c.Get("user_id")

	result, err := h.service.ListCollaborators(
		c.Request.Context(),
		docID,
		userID.(uint64),
	)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

type AddCollaboratorRequest struct {
	UserID uint64 `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=editor viewer"`
}

func (h *Handler) AddCollaborator(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	var req AddCollaboratorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError(err))
		return
	}

	requesterID, _ := c.Get("user_id")

	result, err := h.service.AddCollaborator(
		c.Request.Context(),
		docID,
		requesterID.(uint64),
		req.UserID,
		req.Role,
	)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

type ChangeCollaboratorRoleRequest struct {
	Role 			string `json:"role" binding:"required,oneof=editor viewer"`
	TargetUserID 	uint64 `json:"user_id" binding:"required"`
}

func (h *Handler) ChangeCollaboratorRole(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	var req ChangeCollaboratorRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError(err))
		return
	}

	requesterID, _ := c.Get("user_id")

	result, err := h.service.ChangeCollaboratorRole(
		c.Request.Context(),
		docID,
		requesterID.(uint64),
		req.TargetUserID,
		req.Role,
	)

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) RemoveCollaborator(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	requesterID, _ := c.Get("user_id")

	err = h.service.RemoveCollaborator(
		c.Request.Context(),
		docID,
		requesterID.(uint64),
		targetUserID,
	)

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "collaborator removed",
	})
}

func (h *Handler) DeleteDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := strconv.ParseUint(docIDStr, 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	userID, _ := c.Get("user_id")

	if err := h.service.DeleteDocument(c.Request.Context(), docID, userID.(uint64)); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
