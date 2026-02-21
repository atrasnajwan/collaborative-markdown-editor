package document

import (
	"bytes"
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/middleware"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mock implementation of the Service interface
type MockService struct {
	mock.Mock
}

func (m *MockService) CreateUserDocument(ctx context.Context, userID uint64, document *domain.Document) error {
	args := m.Called(ctx, userID, document)
	return args.Error(0)
}

func (m *MockService) RenameDocument(ctx context.Context, docID uint64, userID uint64, title string) (*domain.Document, error) {
	args := m.Called(ctx, docID, userID, title)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Document), args.Error(1)
}

func (m *MockService) CreateDocumentUpdate(ctx context.Context, id uint64, userID uint64, content []byte) error {
	args := m.Called(ctx, id, userID, content)
	return args.Error(0)
}

func (m *MockService) GetUserDocuments(ctx context.Context, userId uint64, page, pageSize int) (*PaginatedDocuments, error) {
	args := m.Called(ctx, userId, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaginatedDocuments), args.Error(1)
}

func (m *MockService) GetSharedDocuments(ctx context.Context, userId uint64, page, pageSize int) (*PaginatedDocuments, error) {
	args := m.Called(ctx, userId, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaginatedDocuments), args.Error(1)
}

func (m *MockService) GetDocumentByID(ctx context.Context, docID uint64, userID uint64) (*DocumentShowResponse, error) {
	args := m.Called(ctx, docID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DocumentShowResponse), args.Error(1)
}

func (m *MockService) GetDocumentState(ctx context.Context, docID uint64) (*DocumentStateResponse, error) {
	args := m.Called(ctx, docID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DocumentStateResponse), args.Error(1)
}

func (m *MockService) CreateDocumentSnapshot(ctx context.Context, docID uint64, state []byte) error {
	args := m.Called(ctx, docID, state)
	return args.Error(0)
}

func (m *MockService) FetchUserRole(ctx context.Context, docID, userID uint64) (string, error) {
	args := m.Called(ctx, docID, userID)
	return args.String(0), args.Error(1)
}

func (m *MockService) ListCollaborators(ctx context.Context, docID uint64, requesterID uint64) ([]DocumentCollaboratorDTO, error) {
	args := m.Called(ctx, docID, requesterID)
	if args.Get(0) == nil {
		return []DocumentCollaboratorDTO{}, args.Error(1)
	}
	return args.Get(0).([]DocumentCollaboratorDTO), args.Error(1)
}

func (m *MockService) AddCollaborator(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64, role string) (*DocumentCollaboratorDTO, error) {
	args := m.Called(ctx, docID, requesterID, targetUserID, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DocumentCollaboratorDTO), args.Error(1)
}

func (m *MockService) ChangeCollaboratorRole(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64, newRole string) (*DocumentCollaboratorDTO, error) {
	args := m.Called(ctx, docID, requesterID, targetUserID, newRole)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DocumentCollaboratorDTO), args.Error(1)
}

func (m *MockService) RemoveCollaborator(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64) error {
	args := m.Called(ctx, docID, requesterID, targetUserID)
	return args.Error(0)
}

func (m *MockService) DeleteDocument(ctx context.Context, docID uint64, userID uint64) error {
	args := m.Called(ctx, docID, userID)
	return args.Error(0)
}

func setupRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	return router
}

// TestCreateDocument_Success tests successful document creation
func TestCreateDocument_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("CreateUserDocument", mock.Anything, uint64(1), mock.MatchedBy(func(doc *domain.Document) bool {
		return doc.Title == "Test Document"
	})).Return(nil).Run(func(args mock.Arguments) {
		doc := args.Get(2).(*domain.Document)
		doc.ID = 1
	})

	router.POST("/documents", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.Create(c)
	})

	payload := CreateOrRenameRequest{Title: "Test Document"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/documents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

// TestCreateDocument_InvalidInput tests document creation with invalid input
func TestCreateDocument_InvalidInput(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.POST("/documents", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.Create(c)
	})

	payload := struct{}{}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/documents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 422 for validation errors (missing title)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// TestShowUserDocuments_WithPagination tests user documents with pagination
func TestShowUserDocuments_WithPagination(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)
	docs := []DocumentShowResponse{{ID: 1, Title: "Doc 1"}}
	result := &PaginatedDocuments{
		Data: docs,
		Meta: DocumentsMeta{CurrentPage: 2, TotalPage: 3, Total: 25, PerPage: 15},
	}

	mockService.On("GetUserDocuments", mock.Anything, uint64(1), 2, 15).Return(result, nil)

	router.GET("/documents", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.ShowUserDocuments(c)
	})

	req := httptest.NewRequest("GET", "/documents?page=2&per_page=15", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// TestShowSharedDocuments_Success tests retrieving shared documents
func TestShowSharedDocuments_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	docs := []DocumentShowResponse{
		{ID: 1, Title: "Shared Doc 1", Role: "editor"},
		{ID: 2, Title: "Shared Doc 2", Role: "viewer"},
	}
	result := &PaginatedDocuments{
		Data: docs,
		Meta: DocumentsMeta{CurrentPage: 1, TotalPage: 1, Total: 2, PerPage: 10},
	}

	mockService.On("GetSharedDocuments", mock.Anything, uint64(1), 1, 10).Return(result, nil)

	router.GET("/documents/shared", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.ShowSharedDocuments(c)
	})

	req := httptest.NewRequest("GET", "/documents/shared", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

// TestShowDocument_Success tests retrieving a single document
func TestShowDocument_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	doc := &DocumentShowResponse{
		ID:        1,
		Title:     "Test Doc",
		Role:      "editor",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockService.On("GetDocumentByID", mock.Anything, uint64(1), uint64(1)).Return(doc, nil)

	router.GET("/documents/:id", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.ShowDocument(c)
	})

	req := httptest.NewRequest("GET", "/documents/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response DocumentShowResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, uint64(1), response.ID)
	mockService.AssertExpectations(t)
}

// TestShowDocument_InvalidID tests retrieving document with invalid ID
func TestShowDocument_InvalidID(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.GET("/documents/:id", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.ShowDocument(c)
	})

	req := httptest.NewRequest("GET", "/documents/invalid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestShowUserRole_Success tests retrieving user role in document
func TestShowUserRole_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("FetchUserRole", mock.Anything, uint64(1), uint64(2)).Return("editor", nil)

	router.GET("/documents/:id/role", func(c *gin.Context) {
		handler.ShowUserRole(c)
	})

	req := httptest.NewRequest("GET", "/documents/1/role?user_id=2", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "editor", response["role"])
	mockService.AssertExpectations(t)
}

// TestShowUserRole_InvalidDocID tests user role with invalid doc ID
func TestShowUserRole_InvalidDocID(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.GET("/documents/:id/role", func(c *gin.Context) {
		handler.ShowUserRole(c)
	})

	req := httptest.NewRequest("GET", "/documents/invalid/role?user_id=2", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestShowDocumentState_Success tests retrieving document state
func TestShowDocumentState_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	state := &DocumentStateResponse{
		Snapshot:    []byte("document state"),
		SnapshotSeq: 1,
		Updates:     []DocumentUpdateDTO{},
	}

	mockService.On("GetDocumentState", mock.Anything, uint64(1)).Return(state, nil)

	router.GET("/documents/:id/state", func(c *gin.Context) {
		handler.ShowDocumentState(c)
	})

	req := httptest.NewRequest("GET", "/documents/1/state", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// TestCreateUpdate_Success tests creating a document update
func TestCreateUpdate_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	updateBinary := []byte("test update binary")
	mockService.On("CreateDocumentUpdate", mock.Anything, uint64(1), uint64(2), updateBinary).Return(nil)

	router.POST("/documents/:id/updates", func(c *gin.Context) {
		handler.CreateUpdate(c)
	})

	req := httptest.NewRequest("POST", "/documents/1/updates", bytes.NewBuffer(updateBinary))
	req.Header.Set("X-User-Id", "2")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

// TestCreateUpdate_InvalidDocID tests creating update with invalid doc ID
func TestCreateUpdate_InvalidDocID(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.POST("/documents/:id/updates", func(c *gin.Context) {
		handler.CreateUpdate(c)
	})

	req := httptest.NewRequest("POST", "/documents/invalid/updates", bytes.NewBuffer([]byte("test")))
	req.Header.Set("X-User-Id", "2")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestCreateUpdate_InvalidUserID tests creating update with invalid user ID
func TestCreateUpdate_InvalidUserID(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.POST("/documents/:id/updates", func(c *gin.Context) {
		handler.CreateUpdate(c)
	})

	req := httptest.NewRequest("POST", "/documents/1/updates", bytes.NewBuffer([]byte("test")))
	req.Header.Set("X-User-Id", "invalid")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 422 for unprocessable entity (invalid user ID format)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// TestCreateSnapshot_Success tests creating a document snapshot
func TestCreateSnapshot_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	snapshotBinary := []byte("test snapshot binary")
	mockService.On("CreateDocumentSnapshot", mock.Anything, uint64(1), snapshotBinary).Return(nil)

	router.POST("/documents/:id/snapshots", func(c *gin.Context) {
		handler.CreateSnapshot(c)
	})

	req := httptest.NewRequest("POST", "/documents/1/snapshots", bytes.NewBuffer(snapshotBinary))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

// TestListCollaborators_Success tests listing document collaborators
func TestListCollaborators_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	collaborators := []DocumentCollaboratorDTO{
		{User: UserDTO{ID: 2, Name: "User 2", Email: "user2@example.com"}, Role: "editor"},
		{User: UserDTO{ID: 3, Name: "User 3", Email: "user3@example.com"}, Role: "viewer"},
	}

	mockService.On("ListCollaborators", mock.Anything, uint64(1), uint64(1)).Return(collaborators, nil)

	router.GET("/documents/:id/collaborators", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.ListCollaborators(c)
	})

	req := httptest.NewRequest("GET", "/documents/1/collaborators", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// TestAddCollaborator_Success tests adding a collaborator
func TestAddCollaborator_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	collaborator := &DocumentCollaboratorDTO{User: UserDTO{ID: 2, Name: "User 2", Email: "user2@example.com"}, Role: "editor"}
	mockService.On("AddCollaborator", mock.Anything, uint64(1), uint64(1), uint64(2), "editor").Return(collaborator, nil)

	router.POST("/documents/:id/collaborators", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.AddCollaborator(c)
	})

	payload := AddCollaboratorRequest{UserID: 2, Role: "editor"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/documents/1/collaborators", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

// TestAddCollaborator_InvalidRole tests adding collaborator with invalid role
func TestAddCollaborator_InvalidRole(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.POST("/documents/:id/collaborators", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.AddCollaborator(c)
	})

	payload := AddCollaboratorRequest{UserID: 2, Role: "invalid_role"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/documents/1/collaborators", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 422 for validation error (invalid role)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// TestChangeCollaboratorRole_Success tests changing collaborator role
func TestChangeCollaboratorRole_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	collaborator := &DocumentCollaboratorDTO{User: UserDTO{ID: 2, Name: "User 2", Email: "user2@example.com"}, Role: "viewer"}
	mockService.On("ChangeCollaboratorRole", mock.Anything, uint64(1), uint64(1), uint64(2), "viewer").Return(collaborator, nil)

	router.PATCH("/documents/:id/collaborators", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.ChangeCollaboratorRole(c)
	})

	payload := ChangeCollaboratorRoleRequest{TargetUserID: 2, Role: "viewer"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PATCH", "/documents/1/collaborators", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// TestRemoveCollaborator_Success tests removing a collaborator
func TestRemoveCollaborator_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("RemoveCollaborator", mock.Anything, uint64(1), uint64(1), uint64(2)).Return(nil)

	router.DELETE("/documents/:id/collaborators/:userId", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.RemoveCollaborator(c)
	})

	req := httptest.NewRequest("DELETE", "/documents/1/collaborators/2", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// TestDeleteDocument_Success tests deleting a document
func TestDeleteDocument_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	mockService.On("DeleteDocument", mock.Anything, uint64(1), uint64(1)).Return(nil)

	router.DELETE("/documents/:id", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.DeleteDocument(c)
	})

	req := httptest.NewRequest("DELETE", "/documents/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

// TestDeleteDocument_InvalidID tests deleting document with invalid ID
func TestDeleteDocument_InvalidID(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)
	router := setupRouter(handler)

	router.DELETE("/documents/:id", func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		handler.DeleteDocument(c)
	})

	req := httptest.NewRequest("DELETE", "/documents/invalid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
