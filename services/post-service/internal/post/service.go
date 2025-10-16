package post

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"post-service/internal/kafka"
	"post-service/internal/shared/validate"
	"post-service/internal/tag"
)

type Service interface {
	Create(uid string, in CreateReq) (*Post, error)
	GetByID(id uint64) (*Post, error)
	ListByUser(userID string, limit, offset int) ([]Post, error)
	AddView(postID uint64) error
	UploadAndCreate(uid string, filename string, file io.Reader, description string, tags []string, bearer string) (*Post, error)
}

type service struct {
	repo  Repository
	tags  tag.Service
	kafka kafka.Writer
}

func NewService(r Repository, t tag.Service, kw kafka.Writer) Service {
	return &service{repo: r, tags: t, kafka: kw}
}

func (s *service) Create(uid string, in CreateReq) (*Post, error) {
	if err := validate.Struct(in); err != nil {
		return nil, err
	}
	p := &Post{
		UserID: uid, Description: in.Description, MediaURL: in.MediaURL,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	out, err := s.repo.Create(p)
	if err != nil {
		return nil, err
	}
	tgs, err := s.tags.Ensure(in.Tags)
	if err != nil {
		return nil, err
	}
	if len(tgs) > 0 {
		ids := make([]uint64, 0, len(tgs))
		for _, t := range tgs {
			ids = append(ids, t.ID)
		}
		if err := s.repo.AttachTags(out.ID, ids); err != nil {
			return nil, err
		}
	}
	_ = s.kafka.WriteJSON(context.Background(), map[string]any{
		"id":          out.ID,
		"user_id":     out.UserID,
		"description": out.Description,
		"media_url":   out.MediaURL,
		"tags":        in.Tags,
		"created_at":  out.CreatedAt,
	})
	return out, nil
}

func (s *service) GetByID(id uint64) (*Post, error) { return s.repo.GetByID(id) }

func (s *service) ListByUser(userID string, limit, offset int) ([]Post, error) {
	return s.repo.ListByUser(userID, limit, offset)
}

func (s *service) AddView(postID uint64) error { return s.repo.IncView(postID) }

func (s *service) UploadAndCreate(uid, filename string, file io.Reader, description string, tags []string, bearer string) (*Post, error) {
	mediaURL, err := uploadToMediaService(filename, file, bearer)
	if err != nil {
		return nil, err
	}
	return s.Create(uid, CreateReq{Description: description, MediaURL: mediaURL, Tags: tags})
}

func uploadToMediaService(filename string, r io.Reader, bearer string) (string, error) {
	base := os.Getenv("MEDIA_SERVICE_URL")
	if base == "" {
		base = "http://media-service:8088"
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", filename)
	if _, err := io.Copy(fw, r); err != nil {
		return "", err
	}
	_ = w.Close()

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/media/upload", base), &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if strings.TrimSpace(bearer) != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media-service: %s", string(b))
	}
	type out struct {
		URL string `json:"url"`
	}
	var o out
	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return "", err
	}
	return o.URL, nil
}
