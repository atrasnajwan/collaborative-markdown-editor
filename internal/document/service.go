package document

type Service interface {
	CreateUserDocument(userID uint64, document *Document) error
	UpdateDocumentContent(id uint64, userID uint64, content []byte) error
	GetUserDocuments(userId uint64, page, pageSize int) ([]Document, DocumentsMeta, error)
	GetDocumentByID(docID uint64) (*Document, error)
}

type DefaultService struct {
	repository DocumentRepository
}

func NewService(repository DocumentRepository) Service {
	return &DefaultService{repository: repository}
}

func (s *DefaultService) CreateUserDocument(userId uint64, document *Document) error {
	// Create document for user
	return s.repository.Create(userId, document)
}

func (s *DefaultService) UpdateDocumentContent(id uint64, userID uint64, content []byte) error {
   return s.repository.UpdateContent(id, userID, content)
}

func (s *DefaultService) GetUserDocuments(userId uint64, page, pageSize int) ([]Document, DocumentsMeta, error) {
	documentsData, err := s.repository.GetByUserID(userId, page, pageSize)

	if err != nil {
		return []Document{}, DocumentsMeta{}, err
	}

	return documentsData.Documents, documentsData.Meta, nil
}

func (s *DefaultService) GetDocumentByID(docID uint64) (*Document, error) {
	return s.repository.FindByID(docID)
}
