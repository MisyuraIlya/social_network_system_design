package user

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"

	"users-service/internal/shared/shard"

	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(email, password, name string) (*User, error)
	Login(email, password string) (*User, error)
	GetByUserID(uid string) (*User, error)
	ListMine(shardID, limit, offset int) ([]User, error)
}
type service struct {
	repo      Repository
	numShards int
}

func NewService(r Repository) Service {
	n := 1
	if s := os.Getenv("NUM_SHARDS"); s != "" {
		if v, e := strconv.Atoi(s); e == nil && v > 0 {
			n = v
		}
	}
	return &service{repo: r, numShards: n}
}

func (s *service) Register(email, password, name string) (*User, error) {
	sh := shard.Pick(email, s.numShards)
	if exist, _ := s.repo.GetByEmail(email, sh); exist != nil {
		return nil, errors.New("user exists")
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	uid := fmt.Sprintf("%d-%x", sh, binary.BigEndian.Uint64(b[:]))
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("hash fail")
	}
	return s.repo.Create(&User{
		UserID: uid, ShardID: sh, Email: email, PassHash: string(hash), Name: name,
	})
}
func (s *service) Login(email, password string) (*User, error) {
	sh := shard.Pick(email, s.numShards)
	u, err := s.repo.GetByEmail(email, sh)
	if err != nil {
		return nil, errors.New("wrong credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PassHash), []byte(password)) != nil {
		return nil, errors.New("wrong credentials")
	}
	return u, nil
}
func (s *service) GetByUserID(uid string) (*User, error) { return s.repo.GetByUserID(uid) }
func (s *service) ListMine(shardID, limit, offset int) ([]User, error) {
	return s.repo.ListByShard(shardID, limit, offset)
}
