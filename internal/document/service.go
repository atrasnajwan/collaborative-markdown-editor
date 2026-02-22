package document

import (
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/errors"
	"collaborative-markdown-editor/internal/sync"
	"collaborative-markdown-editor/internal/worker"
	"collaborative-markdown-editor/redis"
	"context"
	defError "errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service interface {
	CreateUserDocument(ctx context.Context, userID uint64, document *domain.Document) error
	RenameDocument(ctx context.Context, docID uint64, userID uint64, title string) (*domain.Document, error)
	CreateDocumentUpdate(ctx context.Context, id uint64, userID uint64, content []byte) error
	GetUserDocuments(ctx context.Context, userID uint64, page, pageSize int) (*PaginatedDocuments, error)
	GetSharedDocuments(ctx context.Context, userId uint64, page, pageSize int) (*PaginatedDocuments, error)
	GetDocumentByID(ctx context.Context, docID uint64, userID uint64) (*DocumentShowResponse, error)
	GetDocumentState(ctx context.Context, docID uint64) (*DocumentStateResponse, error)
	CreateDocumentSnapshot(ctx context.Context, docID uint64, state []byte) error
	FetchUserRole(ctx context.Context, docID, userID uint64) (string, error)
	ListCollaborators(ctx context.Context, docID uint64, requesterID uint64) ([]DocumentCollaboratorDTO, error)
	AddCollaborator(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64, role string) (*DocumentCollaboratorDTO, error)
	ChangeCollaboratorRole(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64, newRole string) (*DocumentCollaboratorDTO, error)
	RemoveCollaborator(ctx context.Context, docID uint64, requesterID uint64, targetUserID uint64) error
	DeleteDocument(ctx context.Context, docID uint64, userID uint64) error
}

type UserProvider interface {
	GetUserByID(ctx context.Context, id uint64) (*domain.User, error)
}

type DefaultService struct {
	repository   		DocumentRepository
	syncClient   		*sync.SyncClient
	userProvider 		UserProvider
	cache 				*redis.Cache
	snapshotThreshold 	uint64
	workerPool 			*worker.WorkerPool
}

func NewService(
	repository DocumentRepository,
	userProvider UserProvider,
	syncClient *sync.SyncClient,
	cache *redis.Cache,
	snapshotThreshold uint64,
	wp *worker.WorkerPool,
) Service {
	return &DefaultService{
		repository:   		repository,
		syncClient:   		syncClient,
		userProvider: 		userProvider,
		cache: 				cache,
		snapshotThreshold: 	snapshotThreshold,
		workerPool: 		wp,
	}
}

func (s *DefaultService) CreateUserDocument(ctx context.Context, userID uint64, document *domain.Document) error {
	// Create document for user
	err := s.repository.Create(ctx, userID, document)
	if err == nil {
		// increase cache key, so any new fetch will get new version
    	versionKey := fmt.Sprintf("user:%d:docs:version", userID)
    	s.cache.IncrementVersion(ctx, versionKey)
	}
	return err
}

func (s *DefaultService) RenameDocument(ctx context.Context, docID uint64, userID uint64, title string) (*domain.Document, error) {
    if title == "" {
        return nil, errors.BadRequest("Title cannot be empty", nil)
    }

    doc, err := s.repository.UpdateTitle(ctx, docID, userID, title)
    if err != nil {
        if defError.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.NotFound("Document not found", err)
        }
        return nil, err
    }

	collaborators, _ := s.repository.ListDocumentCollaborators(ctx, docID)

	// Submit to Worker Pool
	s.workerPool.Submit(func(bgCtx context.Context) error {
		// 5s timeout
		timeoutCtx, cancel := context.WithTimeout(bgCtx, 5*time.Second)
		defer cancel()
		// Invalidate cache
		for _, col := range collaborators {
			var versionKey string
            if col.Role == "owner" {
                versionKey = fmt.Sprintf("user:%d:docs:version", col.UserID)
            } else {
				// shared document
                versionKey = fmt.Sprintf("user:%d:docs:shared:version", col.UserID)
            }
            // invalidate
            s.cache.IncrementVersion(timeoutCtx, versionKey)
		}
		return nil
	})

    return doc, nil
}

type PaginatedDocuments struct {
    Data []DocumentShowResponse `json:"data"`
    Meta DocumentsMeta `json:"meta"`
}

func (s *DefaultService) GetUserDocuments(ctx context.Context, userID uint64, page, pageSize int) (*PaginatedDocuments, error) {
	// Get the current data version for this user's documents
    versionKey := fmt.Sprintf("user:%d:docs:version", userID)
    v := s.cache.GetVersion(ctx, versionKey)

	cacheKey := fmt.Sprintf("docs:u:%d:v:%d:p:%d:ps:%d", userID, v, page, pageSize)

	var result PaginatedDocuments
	// get data from cache
    found, _ := s.cache.Get(ctx, cacheKey, &result)
    if found {
        return &result, nil
    }

	documents, meta, err := s.repository.ListDocumentByUserID(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	result = PaginatedDocuments{Data: documents, Meta: meta}
	// set value to cache
	go s.cache.Set(context.Background(), cacheKey, result, 24*time.Hour)

	return &result, nil
}

func (s *DefaultService) GetSharedDocuments(ctx context.Context, userID uint64, page, pageSize int) (*PaginatedDocuments, error) {
	// Get the current data version for this user shared documents
    versionKey := fmt.Sprintf("user:%d:docs:shared:version", userID)
    v := s.cache.GetVersion(ctx, versionKey)

	cacheKey := fmt.Sprintf("docs:shared:u:%d:v:%d:p:%d:ps:%d", userID, v, page, pageSize)

	var result PaginatedDocuments
	// get data from cache
    found, _ := s.cache.Get(ctx, cacheKey, &result)
    if found {
        return &result, nil
    }

	documents, meta, err := s.repository.ListSharedDocuments(ctx, userID, page, pageSize)

	if err != nil {
		return nil, err
	}
	result = PaginatedDocuments{Data: documents, Meta: meta}
	// set value to cache
	go s.cache.Set(context.Background(), cacheKey, result, 24*time.Hour)

	return &result, nil
}


type DocumentShowResponse struct {
	ID        uint64    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Role      string    `json:"role"`
	OwnerName string	`json:"owner_name"`
	OwnerId   uint64	`json:"owner_id"`
}

func (s *DefaultService) GetDocumentByID(ctx context.Context, docID uint64, userID uint64) (*DocumentShowResponse, error) {
	doc, err := s.repository.FindByID(ctx, docID)
	if err != nil {
		return nil, err
	}

	var role string
	role, err = s.repository.GetUserRole(ctx, docID, userID)
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

func (s *DefaultService) FetchUserRole(ctx context.Context, docID, userID uint64) (string, error) {
	return s.repository.GetUserRole(ctx, docID, userID)
}

// context to detect if connection is safe, and cancel downstream if fail
func (s *DefaultService) CreateDocumentUpdate(ctx context.Context, docID uint64, userID uint64, content []byte) error {
	// viewer not allowed to
	role, err := s.FetchUserRole(ctx, docID, userID)
	if err != nil {
		return err
	}
	if role == "viewer" {
		return errors.Forbidden("Viewer can't create update!", nil)
	}

	err = s.repository.CreateUpdate(ctx, docID, userID, content)
	if err != nil {
		return err
	}

	// Throttled Invalidation
    s.workerPool.Submit(func(bgCtx context.Context) error {
		timeoutCtx, cancel := context.WithTimeout(bgCtx, 10*time.Second)
		defer cancel()
		
		// Use docID in the lock so edits to Doc A don't block invalidation for Doc B
        lockKey := fmt.Sprintf("invalidation_cooldown:d:%d", docID)
        
        //  we only invalidate once every 1 minute per document
        isNew, _ := s.cache.SetNX(timeoutCtx, lockKey, "1", time.Minute)
        
        if isNew {
			collaborators, err := s.repository.ListDocumentCollaborators(timeoutCtx, docID)
			if err != nil {
				return err
			}

			for _, col := range collaborators {
				var versionKey string
				if col.Role == "owner" {
					versionKey = fmt.Sprintf("user:%d:docs:version", col.UserID)
				} else {
					// shared document
					versionKey = fmt.Sprintf("user:%d:docs:shared:version", col.UserID)
				}
				// invalidate
				s.cache.IncrementVersion(timeoutCtx, versionKey)
			}

        }
        return nil
    })

	if s.shouldSnapshot(ctx, docID) {
		// run on the background
		s.workerPool.Submit(func(bgCtx context.Context) error {
            return s.handleBackgroundSnapshot(bgCtx, docID)
        })
	}

	return nil
}

// run snapshot on the background
func (s *DefaultService) handleBackgroundSnapshot(ctx context.Context, docID uint64) error {
    lockKey := fmt.Sprintf("lock:snapshot:%d", docID)
    
    // This prevents multiple snapshots for the same document overlapping
    locked, err := s.cache.SetNX(ctx, lockKey, "processing", 30*time.Second)
    if err != nil || !locked {
        return nil // Already being processed by another worker
    }
    defer s.cache.Invalidate(ctx, lockKey)

    // Re-verify if snapshot is still needed
    if !s.shouldSnapshot(ctx, docID) {
        return nil
    }

    // set timeout for fetch from sync + save to DB to 10s
    timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

    state, err := s.syncClient.FetchDocumentState(timeoutCtx, docID)
    if err != nil {
        return err
    }

    return s.repository.CreateSnapshot(timeoutCtx, docID, state)
}

func (s *DefaultService) CreateDocumentSnapshot(ctx context.Context, docID uint64, state []byte) error {
	return s.repository.CreateSnapshot(ctx, docID, state)
}

func (s *DefaultService) shouldSnapshot(ctx context.Context, docID uint64) bool {
	var lastSnapshotSeq uint64
	var currentSeq uint64

	err := s.repository.LastSnapshotSeq(ctx, docID, &lastSnapshotSeq)
	if err != nil {
		return false
	}

	err = s.repository.CurrentSeq(ctx, docID, &currentSeq)
	if err != nil {
		return false
	}

	return currentSeq-lastSnapshotSeq >= s.snapshotThreshold
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

func (s *DefaultService) GetDocumentState(ctx context.Context, docID uint64) (*DocumentStateResponse, error) {

	var snapshot domain.DocumentSnapshot
	err := s.repository.LastSnapshot(ctx, docID, &snapshot)

	var snapshotSeq uint64
	var snapshotState []byte

	if err == nil {
		snapshotSeq = snapshot.Seq
		snapshotState = snapshot.SnapshotBinary
	}

	var updates []domain.DocumentUpdate
	err = s.repository.UpdatesFromSnapshot(ctx, docID, snapshotSeq, &updates)
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
	role, err := s.FetchUserRole(ctx, docID, requesterID)
	if err != nil {
		return nil, err
	}
	if role == "viewer" {
		return nil, errors.Forbidden("Viewer can't show collaborators", nil)
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
	requesterRole, err := s.repository.GetUserRole(ctx, docID, requesterID)
	if err != nil {
		return nil, err
	}
	if requesterRole != "owner" {
		return nil, errors.Forbidden("Only owner can add new collaborator!", nil)
	}

	// Prevent self-add
	if requesterID == targetUserID {
		return nil, errors.UnprocessableEntity("Can't add yourself!", nil)
	}

	// Ensure target user exists
	user, err := s.userProvider.GetUserByID(ctx, targetUserID)
	if err != nil {
		return nil, errors.UnprocessableEntity("Can't find user!", nil)
	}

	if err := s.repository.AddCollaborator(ctx, docID, targetUserID, role); err != nil {
		if defError.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.Conflict("User already added!", err)
		}
		return nil, err
	}

	// invalidate cache
	versionKey := fmt.Sprintf("user:%d:docs:shared:version", targetUserID)
    s.cache.IncrementVersion(ctx, versionKey)

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
	docID,
	requesterID,
	targetUserID uint64,
	newRole string,
) (*DocumentCollaboratorDTO, error) {
	// must be owner
	requesterRole, err := s.repository.GetUserRole(ctx, docID, requesterID)
	if err != nil {
		return nil, err
	}

	if requesterRole != "owner" {
		return nil, errors.Forbidden("Only owner can change role!", nil)
	}

	// Prevent self-demotion
	if requesterID == targetUserID {
		return nil, errors.UnprocessableEntity("Can't add yourself!", nil)
	}

	// Ensure target collaborator exists
	currentRole, err := s.repository.GetUserRole(ctx, docID, targetUserID)
	if err != nil {
		return nil, errors.UnprocessableEntity("Can't find user!", err)
	}

	//  No-op check
	if currentRole == newRole {
		return nil, errors.UnprocessableEntity("User role already match", nil)
	}

	if err := s.repository.UpdateCollaboratorRole(
		ctx,
		docID,
		targetUserID,
		newRole,
	); err != nil {
		return nil, err
	}
	
	// invalidate cache
	versionKey := fmt.Sprintf("user:%d:docs:shared:version", targetUserID)
    s.cache.IncrementVersion(ctx, versionKey)
	
	// Submit to Worker Pool
	s.workerPool.Submit(func(bgCtx context.Context) error {
		// 5s timeout
		timeoutCtx, cancel := context.WithTimeout(bgCtx, 5*time.Second)
		defer cancel()

		// send update to sync-server
		return s.syncClient.UpdateUserPermission(
			timeoutCtx,
			docID,
			targetUserID,
			newRole,
		)
	})

	user, err := s.userProvider.GetUserByID(ctx, targetUserID)
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
	requesterRole, err := s.repository.GetUserRole(ctx, docID, requesterID)
	if err != nil {
		return err
	}
	if requesterRole != "owner" {
		return errors.Forbidden("Only owner can remove collaborator", nil)
	}

	// Prevent owner removing themselves
	if requesterID == targetUserID {
		return errors.UnprocessableEntity("Can't remove yourself", nil)
	}

	// Ensure target exists
	if _, err := s.repository.GetUserRole(ctx, docID, targetUserID); err != nil {
		return errors.UnprocessableEntity("Can't find user", err)
	}

	if err := s.repository.RemoveCollaborator(ctx, docID, targetUserID); err != nil {
		return err
	}

	// invalidate cache
	versionKey := fmt.Sprintf("user:%d:docs:shared:version", targetUserID)
    s.cache.IncrementVersion(ctx, versionKey)
	
	// Submit to Worker Pool
	s.workerPool.Submit(func(bgCtx context.Context) error {
		// 5s timeout
		timeoutCtx, cancel := context.WithTimeout(bgCtx, 5*time.Second)
		defer cancel()

		// send update to sync-server
		return s.syncClient.UpdateUserPermission(
			timeoutCtx,
			docID,
			targetUserID,
			"none",
		)
	})

	return nil
}

func (s *DefaultService) DeleteDocument(ctx context.Context, docID uint64, userID uint64) error {
	var collab domain.DocumentCollaborator
	err := s.repository.GetCollaborator(ctx, docID, userID, &collab)

	if err != nil {
		return errors.UnprocessableEntity("You're not collaborator", err)
	}

	if collab.Role != "owner" {
		return errors.Forbidden("Only owner can delete document", nil)
	}

	collaborators, _ := s.repository.ListDocumentCollaborators(ctx, docID)
	err = s.repository.DeleteDocument(ctx, docID)
	if err != nil {
		return err
	}

	
	// Submit to Worker Pool
	s.workerPool.Submit(func(bgCtx context.Context) error {
		// 10s timeout
		timeoutCtx, cancel := context.WithTimeout(bgCtx, 10*time.Second)
		defer cancel()
		
		// Invalidate cache
		for _, col := range collaborators {
			var versionKey string
            if col.Role == "owner" {
                versionKey = fmt.Sprintf("user:%d:docs:version", col.UserID)
            } else {
				// shared document
                versionKey = fmt.Sprintf("user:%d:docs:shared:version", col.UserID)
            }
            // invalidate
            s.cache.IncrementVersion(timeoutCtx, versionKey)
		}

		// send update to sync-server
		return s.syncClient.RemoveDocument(
			timeoutCtx,
			docID,
		)
	})

	return  nil
}
