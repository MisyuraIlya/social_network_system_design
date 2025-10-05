package user

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"

	"users-service/pkg/shard"
)

type IUserService interface {
	Register(email, password, name string) (*User, error)
	Login(email, password string) (*User, error)
	ListAll(shardID int) ([]User, error)
	ListShard(shardID, limit, offset int) ([]User, error)
	GetByUserID(uid string) (*User, error)
}

type UserService struct {
	repo      IUserRepository
	numShards int
}

func NewUserService(repo IUserRepository) IUserService {
	ns := 1
	if v := os.Getenv("NUM_SHARDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ns = n
		}
	}
	return &UserService{repo: repo, numShards: ns}
}

func (s *UserService) Register(email, password, name string) (*User, error) {
	sh := shard.PickShard(email, s.numShards)

	// ensure uniqueness on the owning shard
	if existing, _ := s.repo.FindByEmail(email, sh); existing != nil {
		return nil, errors.New("user already exists")
	}

	// user_id format: "<shard>-<random64hex>"
	var b [8]byte
	_, _ = rand.Read(b[:])
	uid := fmt.Sprintf("%d-%x", sh, binary.BigEndian.Uint64(b[:]))

	u := &User{
		UserID:   uid,
		ShardID:  sh,
		Email:    email,
		Password: password, // TODO: bcrypt
		Name:     name,
	}
	return s.repo.Create(u)
}

func (s *UserService) Login(email, password string) (*User, error) {
	sh := shard.PickShard(email, s.numShards)
	usr, err := s.repo.FindByEmail(email, sh)
	if err != nil || usr.Password != password {
		return nil, errors.New("wrong credentials")
	}
	return usr, nil
}

func (s *UserService) ListAll(shardID int) ([]User, error) {
	return s.repo.FindAllByShard(shardID)
}

func (s *UserService) GetByUserID(uid string) (*User, error) {
	return s.repo.FindByUserID(uid)
}

func (s *UserService) ListShard(shardID, limit, offset int) ([]User, error) {
	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.FindAllByShardPaged(shardID, limit, offset)
}
