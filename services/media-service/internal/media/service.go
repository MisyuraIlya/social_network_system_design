package media

import (
	"fmt"
	"path"
	"strings"
	"time"

	"media-service/internal/storage/s3"
)

type Service struct {
	s3 *s3.Storage
}

func NewService(s *s3.Storage) *Service { return &Service{s3: s} }

func (s *Service) BuildKey(prefix, filename string, userID string) string {
	fn := path.Base(filename)
	now := time.Now().UTC().Format("20060102T150405")
	p := strings.Trim(prefix, "/")
	if p != "" {
		return fmt.Sprintf("%s/%s_%s_%s", p, userID, now, fn)
	}
	return fmt.Sprintf("%s_%s_%s", userID, now, fn)
}
