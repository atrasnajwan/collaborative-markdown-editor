package document

type Service interface {
	CreateUserDocument(userID uint, document *Document) error
	UpdateDocumentContent(id uint, content string) error
	GetUserDocuments(userId uint, page, pageSize int) ([]Document, DocumentsMeta, error)
	GetDocumentByID(docID uint) (*Document, error)
}

type DefaultService struct {
	repository DocumentRepository
}

func NewService(repository DocumentRepository) Service {
	return &DefaultService{repository: repository}
}

func (s *DefaultService) CreateUserDocument(userId uint, document *Document) error {
	// Create document for user
	return s.repository.Create(userId, document)
}

func (s *DefaultService) UpdateDocumentContent(id uint, content string) error {
    err := s.repository.UpdateContent(id, content)
    if err != nil {
        return err
    }

    return nil
}

func (s *DefaultService) GetUserDocuments(userId uint, page, pageSize int) ([]Document, DocumentsMeta, error) {
	documentsData, err := s.repository.GetByUserID(userId, page, pageSize)

	if err != nil {
		return []Document{}, DocumentsMeta{}, err
	}

	return documentsData.Documents, documentsData.Meta, nil
}

func (s *DefaultService) GetDocumentByID(docID uint) (*Document, error) {
	return s.repository.FindByID(docID)
}
