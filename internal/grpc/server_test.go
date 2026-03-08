package grpc

import (
	"context"
	"testing"

	"collaborative-markdown-editor/internal/document"
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/grpc/internalpb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
)

// simple mock implementing document.Service for use in gRPC tests
type mockDocService struct {
	mock.Mock
}

func (m *mockDocService) CreateUserDocument(ctx context.Context, userID uint64, doc *domain.Document) error {
	args := m.Called(ctx, userID, doc)
	return args.Error(0)
}

func (m *mockDocService) RenameDocument(ctx context.Context, docID uint64, userID uint64, title string) (*domain.Document, error) {
	args := m.Called(ctx, docID, userID, title)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Document), args.Error(1)
}

func (m *mockDocService) CreateDocumentUpdate(ctx context.Context, id, userID uint64, content []byte) error {
	args := m.Called(ctx, id, userID, content)
	return args.Error(0)
}

func (m *mockDocService) GetUserDocuments(ctx context.Context, userID uint64, page, pageSize int) (*document.PaginatedDocuments, error) {
	args := m.Called(ctx, userID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*document.PaginatedDocuments), args.Error(1)
}

func (m *mockDocService) GetSharedDocuments(ctx context.Context, userID uint64, page, pageSize int) (*document.PaginatedDocuments, error) {
	args := m.Called(ctx, userID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*document.PaginatedDocuments), args.Error(1)
}

func (m *mockDocService) GetDocumentByID(ctx context.Context, docID uint64, userID uint64) (*document.DocumentShowResponse, error) {
	args := m.Called(ctx, docID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*document.DocumentShowResponse), args.Error(1)
}

func (m *mockDocService) GetDocumentState(ctx context.Context, docID uint64) (*document.DocumentStateResponse, error) {
	args := m.Called(ctx, docID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*document.DocumentStateResponse), args.Error(1)
}

func (m *mockDocService) CreateDocumentSnapshot(ctx context.Context, docID uint64, state []byte) error {
	args := m.Called(ctx, docID, state)
	return args.Error(0)
}

func (m *mockDocService) FetchUserRole(ctx context.Context, docID, userID uint64) (string, error) {
	args := m.Called(ctx, docID, userID)
	return args.String(0), args.Error(1)
}

func (m *mockDocService) ListCollaborators(ctx context.Context, docID uint64, requesterID uint64) ([]document.DocumentCollaboratorDTO, error) {
	args := m.Called(ctx, docID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]document.DocumentCollaboratorDTO), args.Error(1)
}

func (m *mockDocService) AddCollaborator(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64, role string) (*document.DocumentCollaboratorDTO, error) {
	args := m.Called(ctx, docID, requesterID, targetUserID, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*document.DocumentCollaboratorDTO), args.Error(1)
}

func (m *mockDocService) ChangeCollaboratorRole(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64, newRole string) (*document.DocumentCollaboratorDTO, error) {
	args := m.Called(ctx, docID, requesterID, targetUserID, newRole)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*document.DocumentCollaboratorDTO), args.Error(1)
}

func (m *mockDocService) RemoveCollaborator(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64) error {
	args := m.Called(ctx, docID, requesterID, targetUserID)
	return args.Error(0)
}

func (m *mockDocService) DeleteDocument(ctx context.Context, docID uint64, userID uint64) error {
	args := m.Called(ctx, docID, userID)
	return args.Error(0)
}

// tests start here
func TestAuthInterceptor(t *testing.T) {
	svc := &mockDocService{}
	s := NewServer(svc, "secret123")

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	_, err := s.authInterceptor(context.Background(), nil, nil, handler)
	assert.Error(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-internal-secret", "wrong"))
	_, err = s.authInterceptor(ctx, nil, nil, handler)
	assert.Error(t, err)

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-internal-secret", "secret123"))
	res, err := s.authInterceptor(ctx, nil, nil, handler)
	assert.NoError(t, err)
	assert.Equal(t, "ok", res)
}

func TestGetUserRole(t *testing.T) {
	svc := &mockDocService{}
	s := NewServer(svc, "unused")

	svc.On("FetchUserRole", mock.Anything, uint64(1), uint64(2)).Return("editor", nil)

	resp, err := s.GetUserRole(context.Background(), &internalpb.PermissionRequest{DocId: 1, UserId: 2})
	assert.NoError(t, err)
	assert.Equal(t, "editor", resp.Role)

	svc.AssertExpectations(t)
}

func TestGetDocumentState(t *testing.T) {
	svc := &mockDocService{}
	s := NewServer(svc, "unused")

	state := &document.DocumentStateResponse{
		Snapshot:    []byte{0x01},
		SnapshotSeq: 5,
		Updates:     []document.DocumentUpdateDTO{{Seq: 1, Binary: []byte{0x02}}},
	}
	svc.On("GetDocumentState", mock.Anything, uint64(99)).Return(state, nil)

	resp, err := s.GetDocumentState(context.Background(), &internalpb.DocumentIDRequest{Id: 99})
	assert.NoError(t, err)
	assert.Equal(t, state.Snapshot, resp.Snapshot)
	assert.Equal(t, state.SnapshotSeq, resp.SnapshotSeq)
	assert.Len(t, resp.Updates, 1)
	assert.Equal(t, uint64(1), resp.Updates[0].Seq)
	assert.Equal(t, []byte{0x02}, resp.Updates[0].Binary)

	svc.AssertExpectations(t)
}

func TestCreateUpdateAndSnapshot(t *testing.T) {
	svc := &mockDocService{}
	s := NewServer(svc, "unused")

	svc.On("CreateDocumentUpdate", mock.Anything, uint64(7), uint64(8), []byte{0xAA}).Return(nil)
	svc.On("CreateDocumentSnapshot", mock.Anything, uint64(7), []byte{0xBB}).Return(nil)

	_, err := s.CreateUpdate(context.Background(), &internalpb.UpdateRequest{DocId: 7, UserId: 8, Update: []byte{0xAA}})
	assert.NoError(t, err)

	_, err = s.CreateSnapshot(context.Background(), &internalpb.SnapshotRequest{DocId: 7, Snapshot: []byte{0xBB}})
	assert.NoError(t, err)

	svc.AssertExpectations(t)
}
