package message

import (
	"bytes"
	"io"
)

// bytesReader returns an io.Reader for []byte
func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
