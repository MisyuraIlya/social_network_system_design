package media

import "time"

type UploadMediaResponse struct {
	ID          uint      `json:"id"`
	FileName    string    `json:"fileName"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"createdAt"`
}

type GetMediaResponse struct {
	ID          uint      `json:"id"`
	FileName    string    `json:"fileName"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"createdAt"`
}
