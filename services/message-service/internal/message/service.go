package message

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"message-service/internal/chat"
	"message-service/internal/kafka"
	"message-service/internal/media"
	"message-service/internal/redisx"
)

type Service interface {
	Send(ctx context.Context, userID string, in SendReq) (*Message, error)
	SendWithUpload(ctx context.Context, userID string, chatID int64, fileName string, fileData []byte, text string, bearer string) (*Message, error)
	MarkSeen(messageID int64, userID string) error
	ListByChat(chatID int64, limit, offset int) ([]Message, error)
}

type service struct {
	repo  Repository
	chats chat.Service
	rds   *redisx.Client
	kafka *kafka.Writer
	media *media.Client
}

func NewService(r Repository, cs chat.Service, rds *redisx.Client, kw *kafka.Writer, mc *media.Client) Service {
	return &service{repo: r, chats: cs, rds: rds, kafka: kw, media: mc}
}

func (s *service) Send(ctx context.Context, userID string, in SendReq) (*Message, error) {
	// ensure chat exists
	if _, err := s.chats.GetByID(in.ChatID); err != nil {
		return nil, err
	}
	m := &Message{
		UserID: userID, ChatID: in.ChatID,
		Text: in.Text, MediaURL: in.MediaURL,
		SendTime: time.Now(),
	}
	res, err := s.repo.Create(m)
	if err != nil {
		return nil, err
	}

	// side effects: popularity + kafka event
	s.chats.IncPopular(ctx, in.ChatID)
	_ = s.emit(res)

	return res, nil
}

func (s *service) SendWithUpload(ctx context.Context, userID string, chatID int64, fileName string, data []byte, text string, bearer string) (*Message, error) {
	url, err := s.media.Upload("file", fileName, bytesReader(data), bearer)
	if err != nil {
		return nil, err
	}
	return s.Send(ctx, userID, SendReq{ChatID: chatID, Text: text, MediaURL: url})
}

func (s *service) MarkSeen(messageID int64, userID string) error {
	return s.repo.MarkSeen(messageID, userID)
}

func (s *service) ListByChat(chatID int64, limit, offset int) ([]Message, error) {
	return s.repo.ListByChat(chatID, limit, offset)
}

func (s *service) emit(m *Message) error {
	b, _ := json.Marshal(map[string]any{
		"message_id": m.ID, "chat_id": m.ChatID, "user_id": m.UserID,
		"text": m.Text, "media_url": m.MediaURL, "send_time": m.SendTime,
	})
	return s.kafka.Publish(context.Background(), "chat:"+strconv.FormatInt(m.ChatID, 10), b)
}
