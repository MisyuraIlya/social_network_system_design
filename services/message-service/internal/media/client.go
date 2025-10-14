package media

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
)

type Client struct{ base string }

func New(base string) *Client {
	if base == "" {
		base = "http://media-service:8088"
	}
	return &Client{base: base}
}

func (c *Client) Upload(fieldName, fileName string, r io.Reader) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile(fieldName, fileName)
	_, _ = io.Copy(fw, r)
	_ = w.Close()

	req, _ := http.NewRequest("POST", c.base+"/media/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", io.ErrUnexpectedEOF
	}
	var o struct {
		URL string `json:"url"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&o)
	return o.URL, nil
}
