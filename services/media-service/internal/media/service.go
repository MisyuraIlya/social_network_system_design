package media

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type IMediaService interface {
	Upload(file multipart.File, header *multipart.FileHeader) (*Media, error)
	Get(id uint) (*Media, error)
}

type MediaService struct {
	repo        IMediaRepository
	storagePath string
}

func NewMediaService(r IMediaRepository, storagePath string) IMediaService {
	return &MediaService{
		repo:        r,
		storagePath: storagePath,
	}
}

func (s *MediaService) Upload(file multipart.File, header *multipart.FileHeader) (*Media, error) {
	filename := header.Filename
	outPath := filepath.Join(s.storagePath, filename)

	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	size, err := io.Copy(outFile, file)
	if err != nil {
		return nil, err
	}

	media := &Media{
		FileName:    filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        size,
	}
	return s.repo.Create(media)
}

func (s *MediaService) Get(id uint) (*Media, error) {
	return s.repo.FindByID(id)
}
