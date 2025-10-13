package shard

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"
)

func Pick(key string, n int) int {
	h := sha256.Sum256([]byte(key))
	v := binary.BigEndian.Uint32(h[:4]) ^ binary.BigEndian.Uint32(h[4:8])
	return int(uint32(v) % uint32(n))
}
func Extract(userID string) (int, bool) {
	i := strings.IndexByte(userID, '-')
	if i <= 0 {
		return 0, false
	}
	n, err := strconv.Atoi(userID[:i])
	if err != nil {
		return 0, false
	}
	return n, true
}
