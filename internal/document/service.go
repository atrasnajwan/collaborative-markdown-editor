package document

import (
	"collaborative-markdown-editor/internal/config"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Service interface {
	CreateUserDocument(userID uint64, document *Document) error
	CreateDocumentUpdate(ctx context.Context, id uint64, userID uint64, content []byte) error
	GetUserDocuments(userId uint64, page, pageSize int) ([]Document, DocumentsMeta, error)
	GetDocumentByID(docID uint64, userID uint64) (*DocumentShowResponse, error)
	GetDocumentState(docID uint64) (*DocumentStateResponse, error)
}

type DefaultService struct {
	repository DocumentRepository
	httpClient *http.Client
}

func NewService(repository DocumentRepository) Service {
	return &DefaultService{
		repository: repository,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *DefaultService) CreateUserDocument(userId uint64, document *Document) error {
	// Create document for user
	return s.repository.Create(userId, document)
}

type DocumentsData struct {
	Documents []Document
	Meta      DocumentsMeta `json:"total_page"`
}

func (s *DefaultService) GetUserDocuments(userId uint64, page, pageSize int) ([]Document, DocumentsMeta, error) {
	documents, meta, err := s.repository.GetByUserID(userId, page, pageSize)

	if err != nil {
		return []Document{}, DocumentsMeta{}, err
	}

	return documents, meta, nil
}

type DocumentShowResponse struct {
	ID			uint64		`json:"id"`
	Title       string		`json:"title"`
	CreatedAt	time.Time	`json:"created_at"`
	UpdatedAt	time.Time	`json:"updated_at"`
	Role		string		`json:"role"`
}

func (s *DefaultService) GetDocumentByID(docID uint64, userID uint64) (*DocumentShowResponse, error) {
	doc, err := s.repository.FindByID(docID)
	if err != nil {
		return nil, err
	}

	var role string
	err = s.repository.GetUserRole(docID, userID, &role)
	if err != nil {
		return nil, err
	}
	if role == "" {
		role = "none"
	}
	
	return &DocumentShowResponse{
		ID:  		doc.ID,
		Title: 		doc.Title,
		Role: 		role,
		CreatedAt: 	doc.CreatedAt,
		UpdatedAt: 	doc.UpdatedAt,
	}, nil
}
}

// context to detect if connection is safe, and cancel downstream if fail
func (s *DefaultService) CreateDocumentUpdate(ctx context.Context, id uint64, userID uint64, content []byte) error {
	err := s.repository.CreateUpdate(id, userID, content)
	if err != nil {
		return err
	}

	if s.shouldSnapshot(id) {
		state, err := s.fetchStateFromSyncServer(ctx, id)
		if err != nil {
			return err
		}

		// cancel if the request timeout
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		return s.repository.CreateSnapshot(ctx, id, state)
	}

	return nil
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

	var snapshot DocumentSnapshot
	err := s.repository.LastSnapshot(docID, &snapshot)

	var snapshotSeq uint64
	var snapshotState []byte

	if err == nil {
		snapshotSeq = snapshot.Seq
		snapshotState = snapshot.SnapshotBinary
	}

	var updates []DocumentUpdate
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

func toDocumentUpdateDTOs(updates []DocumentUpdate) []DocumentUpdateDTO {
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
		return nil, fmt.Errorf("node returned %d", resp.StatusCode)
	}

	var payload StateResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(payload.Binary)
}
