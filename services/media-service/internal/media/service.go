package media

import (
	"context"
	"fmt"
	"mime/multipart"
)

type MediaService interface {
	Upload(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (*MediaFile, error)
}

type mediaService struct {
	repo       MediaRepository
	bucketName string
}

func NewMediaService(repo MediaRepository, bucketName string) MediaService {
	return &mediaService{
		repo:       repo,
		bucketName: bucketName,
	}
}

func (m *mediaService) Upload(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (*MediaFile, error) {
	defer file.Close()

	fileName := fileHeader.Filename
	contentType := fileHeader.Header.Get("Content-Type")
	fileSize := fileHeader.Size

	objectName, err := m.repo.UploadFile(ctx, m.bucketName, fileName, file, fileSize, contentType)
	if err != nil {
		return nil, err
	}

	// Construct a URL or location to your S3 object.
	// This depends on how you'll serve it, e.g. presigned or from a CDN, etc.
	fileURL := fmt.Sprintf("%s/%s", m.bucketName, objectName)

	return &MediaFile{
		FileName: fileName,
		FileType: contentType,
		FileSize: fileSize,
		URL:      fileURL,
	}, nil
}
