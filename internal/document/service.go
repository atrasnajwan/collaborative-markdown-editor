package document

import (
	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/errors"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type Service interface {
	CreateUserDocument(userID uint64, document *domain.Document) error
	CreateDocumentUpdate(ctx context.Context, id uint64, userID uint64, content []byte) error
	GetUserDocuments(userId uint64, page, pageSize int) ([]domain.Document, DocumentsMeta, error)
	GetDocumentByID(docID uint64, userID uint64) (*DocumentShowResponse, error)
	GetDocumentState(docID uint64) (*DocumentStateResponse, error)
	CreateDocumentSnapshot(ctx context.Context, docID uint64, state []byte) error
	FetchUserRole(docID, userID uint64) (string, error)
	ListCollaborators(ctx context.Context, docID uint64, requesterID uint64) ([]DocumentCollaboratorDTO, error)
	AddCollaborator(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64, role string) (*DocumentCollaboratorDTO, error)
	ChangeCollaboratorRole(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64, newRole string) (*DocumentCollaboratorDTO, error)
	RemoveCollaborator(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64) error
}

type UserProvider interface {
	GetUserByID(id uint64) (*domain.User, error)
}

type DefaultService struct {
	repository   DocumentRepository
	httpClient   *http.Client
	userProvider UserProvider
}

func NewService(repository DocumentRepository, userProvider UserProvider) Service {
	return &DefaultService{
		repository:   repository,
		httpClient:   &http.Client{Timeout: 5 * time.Second},
		userProvider: userProvider,
	}
}

func (s *DefaultService) CreateUserDocument(userId uint64, document *domain.Document) error {
	// Create document for user
	return s.repository.Create(userId, document)
}

type DocumentsData struct {
	Documents []domain.Document
	Meta      DocumentsMeta `json:"total_page"`
}

func (s *DefaultService) GetUserDocuments(userId uint64, page, pageSize int) ([]domain.Document, DocumentsMeta, error) {
	documents, meta, err := s.repository.ListDocumentByUserID(userId, page, pageSize)

	if err != nil {
		return []domain.Document{}, DocumentsMeta{}, err
	}

	return documents, meta, nil
}

type DocumentShowResponse struct {
	ID        uint64    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Role      string    `json:"role"`
}

func (s *DefaultService) GetDocumentByID(docID uint64, userID uint64) (*DocumentShowResponse, error) {
	doc, err := s.repository.FindByID(docID)
	if err != nil {
		return nil, err
	}

	var role string
	role, err = s.repository.GetUserRole(docID, userID)
	if err != nil {
		return nil, err
	}

	return &DocumentShowResponse{
		ID:        doc.ID,
		Title:     doc.Title,
		Role:      role,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}, nil
}

func (s *DefaultService) FetchUserRole(docID, userID uint64) (string, error) {
	return s.repository.GetUserRole(docID, userID)
}

// context to detect if connection is safe, and cancel downstream if fail
func (s *DefaultService) CreateDocumentUpdate(ctx context.Context, docID uint64, userID uint64, content []byte) error {
	// viewer not allowed to
	role, err := s.FetchUserRole(docID, userID)
	if err != nil {
		return err
	}
	if role == "viewer" {
		return errors.ErrForbidden(nil)
	}

	err = s.repository.CreateUpdate(docID, userID, content)
	if err != nil {
		return err
	}

	if s.shouldSnapshot(docID) {
		state, err := s.fetchStateFromSyncServer(ctx, docID)
		if err != nil {
			return err
		}

		// cancel if the request timeout
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		return s.repository.CreateSnapshot(ctx, docID, state)
	}

	return nil
}

func (s *DefaultService) CreateDocumentSnapshot(ctx context.Context, docID uint64, state []byte) error {
	return s.repository.CreateSnapshot(ctx, docID, state)
}

func (s *DefaultService) shouldSnapshot(docID uint64) bool {
	const snapshotEvery = 200

	var lastSnapshotSeq uint64
	var currentSeq uint64

	err := s.repository.LastSnapshotSeq(docID, &lastSnapshotSeq)
	if err != nil {
		return false
	}

	err = s.repository.CurrentSeq(docID, &currentSeq)
	if err != nil {
		return false
	}

	return currentSeq-lastSnapshotSeq >= snapshotEvery
}

type DocumentUpdateDTO struct {
	Seq    uint64 `json:"seq"`
	Binary []byte `json:"binary"`
}

type DocumentStateResponse struct {
	Snapshot    []byte              `json:"snapshot"`
	SnapshotSeq uint64              `json:"snapshot_seq"`
	Updates     []DocumentUpdateDTO `json:"updates"`
}

func (s *DefaultService) GetDocumentState(docID uint64) (*DocumentStateResponse, error) {

	var snapshot domain.DocumentSnapshot
	err := s.repository.LastSnapshot(docID, &snapshot)

	var snapshotSeq uint64
	var snapshotState []byte

	if err == nil {
		snapshotSeq = snapshot.Seq
		snapshotState = snapshot.SnapshotBinary
	}

	var updates []domain.DocumentUpdate
	err = s.repository.UpdatesFromSnapshot(docID, snapshotSeq, &updates)
	if err != nil {
		return nil, err
	}

	return &DocumentStateResponse{
		Snapshot:    snapshotState,
		SnapshotSeq: snapshotSeq,
		Updates:     toDocumentUpdateDTOs(updates),
	}, nil
}

func toDocumentUpdateDTOs(updates []domain.DocumentUpdate) []DocumentUpdateDTO {
	dtos := make([]DocumentUpdateDTO, 0, len(updates))

	for _, u := range updates {
		dtos = append(dtos, DocumentUpdateDTO{
			Seq:    u.Seq,
			Binary: u.UpdateBinary,
		})
	}

	return dtos
}

type StateResponse struct {
	Binary string `json:"binary"`
}

// call sync server to get current doc state
func (s *DefaultService) fetchStateFromSyncServer(ctx context.Context, docID uint64) ([]byte, error) {
	url := fmt.Sprintf("%s/internal/documents/%d/state", config.AppConfig.SyncServerAddress, docID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sync server returned %d", resp.StatusCode)
	}

	var payload StateResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(payload.Binary)
}

type UserDTO struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type DocumentCollaboratorDTO struct {
	User UserDTO `json:"user"`
	Role string  `json:"role"`
}

func (s *DefaultService) ListCollaborators(ctx context.Context, docID uint64, requesterID uint64) ([]DocumentCollaboratorDTO, error) {
	rows, err := s.repository.ListDocumentCollaborators(ctx, docID)
	if err != nil {
		return nil, err
	}

	// viewer not allowed to
	role, err := s.FetchUserRole(docID, requesterID)
	if err != nil {
		return nil, err
	}
	if role == "viewer" {
		return nil, errors.ErrForbidden(nil)
	}

	// Map to API DTO
	result := make([]DocumentCollaboratorDTO, 0, len(rows))
	for _, r := range rows {
		result = append(result, DocumentCollaboratorDTO{
			User: UserDTO{
				ID:    r.UserID,
				Name:  r.Name,
				Email: r.Email,
			},
			Role: r.Role,
		})
	}

	return result, nil
}

func (s *DefaultService) AddCollaborator(
	ctx context.Context,
	docID uint64,
	requesterID uint64,
	targetUserID uint64,
	role string,
) (*DocumentCollaboratorDTO, error) {
	// only owner can add
	requesterRole, err := s.repository.GetUserRole(docID, requesterID)
	if err != nil || requesterRole != "owner" {
		return nil, errors.ErrForbidden(nil)
	}

	// Prevent self-add
	if requesterID == targetUserID {
		return nil, errors.ErrInvalidInput(nil).WithMessage("can't add yourself")
	}

	// Ensure target user exists
	user, err := s.userProvider.GetUserByID(targetUserID)
	if err != nil {
		return nil, errors.ErrInvalidInput(nil).WithMessage("can't find user")
	}

	if err := s.repository.AddCollaborator(ctx, docID, targetUserID, role); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.ErrAlreadyExists(nil).WithMessage("user already added")
		}
		return nil, err
	}

	// 5. Response DTO
	return &DocumentCollaboratorDTO{
		User: UserDTO{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
		Role: role,
	}, nil
}

func (s *DefaultService) ChangeCollaboratorRole(
	ctx context.Context,
	docID uint64,
	requesterID uint64,
	targetUserID uint64,
	newRole string,
) (*DocumentCollaboratorDTO, error) {
	// must be owner
	requesterRole, err := s.repository.GetUserRole(docID, requesterID)
	if err != nil || requesterRole != "owner" {
		return nil, errors.ErrForbidden(nil)
	}

	// Prevent self-demotion
	if requesterID == targetUserID {
		return nil, errors.ErrInvalidInput(nil).WithMessage("can't add yourself")
	}

	// Ensure target collaborator exists
	currentRole, err := s.repository.GetUserRole(docID, targetUserID)
	if err != nil {
		return nil, errors.ErrInvalidInput(nil).WithMessage("can't find user")
	}

	//  No-op check
	if currentRole == newRole {
		return nil, errors.ErrUnprocessableEntity(nil).WithMessage("user role already match")
	}

	if err := s.repository.UpdateCollaboratorRole(
		ctx,
		docID,
		targetUserID,
		newRole,
	); err != nil {
		return nil, err
	}

	user, err := s.userProvider.GetUserByID(targetUserID)
	if err != nil {
		return nil, err
	}

	return &DocumentCollaboratorDTO{
		User: UserDTO{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
		Role: newRole,
	}, nil
}

func (s *DefaultService) RemoveCollaborator(
	ctx context.Context,
	docID uint64,
	requesterID uint64,
	targetUserID uint64,
) error {
	// must be owner
	requesterRole, err := s.repository.GetUserRole(docID, requesterID)
	if err != nil || requesterRole != "owner" {
		return errors.ErrForbidden(nil)
	}

	// Prevent owner removing themselves
	if requesterID == targetUserID {
		return errors.ErrInvalidInput(nil).WithMessage("can't remove yourself")
	}

	// Ensure target exists
	if _, err := s.repository.GetUserRole(docID, targetUserID); err != nil {
		return errors.ErrInvalidInput(nil).WithMessage("can't find user")
	}

	if err := s.repository.RemoveCollaborator(ctx, docID, targetUserID); err != nil {
		return err
	}

	return nil
}
