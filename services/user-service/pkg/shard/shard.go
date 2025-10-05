package shard

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"
)

func PickShard(key string, numShards int) int {
	h := sha256.Sum256([]byte(key))
	v := binary.BigEndian.Uint32(h[:4]) ^ binary.BigEndian.Uint32(h[4:8])
	return int(uint32(v) % uint32(numShards))
}

// "0-abcdef..." → 0, true
func ExtractShard(userID string) (int, bool) {
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
