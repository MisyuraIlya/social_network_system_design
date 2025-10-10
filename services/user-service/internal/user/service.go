package user

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"users-service/pkg/shard"

	"golang.org/x/crypto/bcrypt"
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
	log.Printf("[user-service] Register route -> picked shard=%d for email=%s", sh, email)

	// ensure uniqueness on the owning shard
	if existing, _ := s.repo.FindByEmail(email, sh); existing != nil {
		return nil, errors.New("user already exists")
	}

	// user_id format: "<shard>-<random64hex>"
	var b [8]byte
	_, _ = rand.Read(b[:])
	uid := fmt.Sprintf("%d-%x", sh, binary.BigEndian.Uint64(b[:]))

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	u := &User{
		UserID:   uid,
		ShardID:  sh,
		Email:    email,
		Password: string(hash),
		Name:     name,
	}
	return s.repo.Create(u)
}

func (s *UserService) Login(email, password string) (*User, error) {
	sh := shard.PickShard(email, s.numShards)
	log.Printf("[user-service] Login route -> picked shard=%d for email=%s", sh, email)

	usr, err := s.repo.FindByEmail(email, sh)
	if err != nil {
		return nil, errors.New("wrong credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(password)) != nil {
		return nil, errors.New("wrong credentials")
	}
	return usr, nil
}

func (s *UserService) ListAll(shardID int) ([]User, error) {
	return s.repo.FindAllByShard(shardID)
}

func (s *UserService) GetByUserID(uid string) (*User, error) {
	sh, ok := shard.ExtractShard(uid)
	if ok {
		log.Printf("[user-service] GetByUserID -> extracted shard=%d from user_id=%s", sh, uid)
	}
	return s.repo.FindByUserID(uid)
}

func (s *UserService) ListShard(shardID, limit, offset int) ([]User, error) {
	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	log.Printf("[user-service] ListShard -> shard=%d limit=%d offset=%d", shardID, limit, offset)
	return s.repo.FindAllByShardPaged(shardID, limit, offset)
}
