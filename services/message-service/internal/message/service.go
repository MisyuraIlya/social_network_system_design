package message

import (
	"encoding/json" // For marshaling messages to Kafka
	"log"           // For logging publishing errors
	"net/http"      // For Media Service client
	"time"          // For HTTP client timeout
)

// Service defines business logic for messages.
type Service interface {
	CreateMessage(userID uint, chatID uint, content string) (*Message, error)
	ListMessages() ([]Message, error)
	UpdateMessage(messageID uint, content string) error
	DeleteMessage(messageID uint) error
	ListMessagesByChat(chatID uint) ([]Message, error)
}

type service struct {
	repo        Repository
	cache       Cache
	publisher   Publisher    // Added Publisher dependency
	mediaSvcURL string       // Added Media Service URL
	httpClient  *http.Client // Added HTTP client
}

// NewService creates a new message Service.
func NewService(repo Repository, cache Cache, publisher Publisher, mediaSvcURL string) Service { // Added mediaSvcURL parameter
	return &service{
		repo:        repo,
		cache:       cache,
		publisher:   publisher,
		mediaSvcURL: mediaSvcURL,                             // Store Media Service URL
		httpClient:  &http.Client{Timeout: 10 * time.Second}, // Initialize HTTP client
	}
}

// CreateMessage handles message creation, including potential media upload.
// TODO: Update signature to accept media data (e.g., io.Reader, filename, content type)
func (s *service) CreateMessage(userID uint, chatID uint, content string) (*Message, error) {

	// TODO: Handle media upload if media data is provided
	// 1. Call Media Service API to upload the file
	//    mediaURL, uploadErr := s.uploadMedia(mediaData, filename, contentType)
	//    if uploadErr != nil {
	//       return nil, fmt.Errorf("media upload failed: %w", uploadErr)
	//    }
	// 2. Store the returned mediaURL with the message

	msg := &Message{
		UserID:  userID,
		ChatID:  chatID,
		Content: content,
		// MediaURL: mediaURL, // Store the URL from Media Service
	}
	err := s.repo.Save(msg)
	if err == nil {
		// Publish event to Kafka after successful save (consider including MediaURL)
		payload, marshalErr := json.Marshal(msg)
		if marshalErr != nil {
			log.Printf("Error marshaling message for Kafka: %v", marshalErr)
			// Decide if this should return an error or just log
		} else {
			publishErr := s.publisher.PublishNewMessage(payload)
			if publishErr != nil {
				log.Printf("Error publishing message to Kafka: %v", publishErr)
				// Decide if this should return an error or just log
			}
		}
		// Optionally update cache or perform additional business logic here.
	}
	return msg, err
}

func (s *service) ListMessages() ([]Message, error) {
	return s.repo.FindAll()
}

func (s *service) UpdateMessage(messageID uint, content string) error {
	return s.repo.UpdateMessage(messageID, content)
}

func (s *service) DeleteMessage(messageID uint) error {
	return s.repo.DeleteMessage(messageID)
}

func (s *service) ListMessagesByChat(chatID uint) ([]Message, error) {
	return s.repo.FindByChat(chatID)
}
