package document

type Service interface {
	CreateUserDocument(userID uint, document *Document) error
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

