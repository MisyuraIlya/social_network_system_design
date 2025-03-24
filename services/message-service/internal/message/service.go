package message

type IMessageService interface {
	SendMessage(dialogID, senderID uint, content string) (*Message, error)
	GetMessages(dialogID uint) ([]Message, error)
}

type MessageService struct {
	repo IMessageRepository
}

func NewMessageService(r IMessageRepository) IMessageService {
	return &MessageService{repo: r}
}

func (s *MessageService) SendMessage(dialogID, senderID uint, content string) (*Message, error) {
	msg := &Message{
		DialogID: dialogID,
		SenderID: senderID,
		Content:  content,
	}
	return s.repo.Create(msg)
}

func (s *MessageService) GetMessages(dialogID uint) ([]Message, error) {
	return s.repo.FindByDialogID(dialogID)
}
