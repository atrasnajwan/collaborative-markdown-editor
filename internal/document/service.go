package document

import (
	"collaborative-markdown-editor/internal/config"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Service interface {
	CreateUserDocument(userID uint64, document *Document) error
	CreateDocumentUpdate(ctx context.Context, id uint64, userID uint64, content []byte) error
	GetUserDocuments(userId uint64, page, pageSize int) ([]Document, DocumentsMeta, error)
	GetDocumentByID(docID uint64) (*Document, error)
	CreateSnapshot(ctx context.Context, docID uint64, state []byte) error
	GetDocumentState(docID uint64) (*DocumentStateResponse, error)
}

type DefaultService struct {
	repository DocumentRepository
	httpClient   *http.Client
}

func NewService(repository DocumentRepository) Service {
	return &DefaultService{
			repository: repository,
			httpClient: &http.Client{ Timeout: 5 * time.Second },
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

func (s *DefaultService) GetDocumentByID(docID uint64) (*Document, error) {
	return s.repository.FindByID(docID)
}

// context to detect if connection is safe, and cancel downstream if fail
func (s *DefaultService) CreateDocumentUpdate(ctx context.Context, id uint64, userID uint64, content []byte) error {
	err := s.repository.CreateUpdate(id, userID, content)
	if err != nil {
		return err
	}

	if s.shouldSnapshot(id) {
        state, err := s.fetchStateFromWS(ctx, id)
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

// context to detect if connection is safe, and cancel downstream if fail
func (s *DefaultService) CreateSnapshot(ctx context.Context, docID uint64, state []byte) error {
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
	log.Println("curr seq", currentSeq)
    return currentSeq - lastSnapshotSeq >= snapshotEvery
}

type DocumentUpdateDTO struct {
	Seq    uint64 `json:"seq"`
	Binary []byte `json:"binary"`
}

type DocumentStateResponse struct {
	Title       string              `json:"title"`
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

	document, err := s.GetDocumentByID(docID)
    if err != nil {
        return nil, err
    }

	return &DocumentStateResponse{
		Title:       document.Title,
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
