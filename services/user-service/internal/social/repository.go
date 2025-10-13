package social

import (
	"errors"

	"users-service/internal/shared/db"
	"users-service/internal/shared/shard"
	"users-service/internal/user"
)

type Repository interface {
	Follow(uid, target string) error
	Unfollow(uid, target string) error
	ListFollowing(uid string, limit, offset int) ([]string, error)

	Befriend(a, b string) error
	Unfriend(a, b string) error
	ListFriends(uid string, limit, offset int) ([]string, error)

	CreateRelationship(uid, related string, typ int) error
	DeleteRelationship(uid, related string, typ int) error
	ListRelationships(uid string, typ, limit, offset int) ([]string, error)
}

type repo struct {
	store *db.Store
	users user.Repository
}

func NewRepository(s *db.Store, ur user.Repository) Repository { return &repo{store: s, users: ur} }

func (r *repo) ensureUser(uid string) error {
	_, err := r.users.GetByUserID(uid)
	return err
}

func (r *repo) Follow(uid, target string) error {
	if uid == target {
		return errors.New("cannot follow self")
	}
	if err := r.ensureUser(target); err != nil {
		return errors.New("target not found")
	}
	sh, _ := shard.Extract(uid)
	return r.store.Write(sh).FirstOrCreate(&Follow{UserID: uid, TargetID: target}).Error
}
func (r *repo) Unfollow(uid, target string) error {
	sh, _ := shard.Extract(uid)
	return r.store.Write(sh).Delete(&Follow{}, "user_id=? AND target_id=?", uid, target).Error
}
func (r *repo) ListFollowing(uid string, limit, offset int) ([]string, error) {
	sh, _ := shard.Extract(uid)
	type Row struct{ TargetID string }
	var rows []Row
	if err := r.store.Use(sh).Model(&Follow{}).
		Where("user_id = ?", uid).Order("created_at DESC").
		Limit(limit).Offset(offset).Select("target_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].TargetID
	}
	return out, nil
}

func (r *repo) Befriend(a, b string) error {
	if a == b {
		return errors.New("cannot friend self")
	}
	if err := r.ensureUser(b); err != nil {
		return errors.New("target not found")
	}
	sha, _ := shard.Extract(a)
	shb, _ := shard.Extract(b)
	if err := r.store.Write(sha).FirstOrCreate(&Friend{UserID: a, FriendID: b}).Error; err != nil {
		return err
	}
	return r.store.Write(shb).FirstOrCreate(&Friend{UserID: b, FriendID: a}).Error
}
func (r *repo) Unfriend(a, b string) error {
	sha, _ := shard.Extract(a)
	shb, _ := shard.Extract(b)
	if err := r.store.Write(sha).Delete(&Friend{}, "user_id=? AND friend_id=?", a, b).Error; err != nil {
		return err
	}
	return r.store.Write(shb).Delete(&Friend{}, "user_id=? AND friend_id=?", b, a).Error
}
func (r *repo) ListFriends(uid string, limit, offset int) ([]string, error) {
	sh, _ := shard.Extract(uid)
	type Row struct{ FriendID string }
	var rows []Row
	if err := r.store.Use(sh).Model(&Friend{}).
		Where("user_id = ?", uid).Order("created_at DESC").
		Limit(limit).Offset(offset).Select("friend_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].FriendID
	}
	return out, nil
}

func (r *repo) CreateRelationship(uid, related string, typ int) error {
	if uid == related {
		return errors.New("cannot relate to self")
	}
	if err := r.ensureUser(related); err != nil {
		return errors.New("target not found")
	}
	sh, _ := shard.Extract(uid)
	rel := &Relationship{UserID: uid, RelatedID: related, Type: typ}
	return r.store.Write(sh).FirstOrCreate(rel).Error
}

func (r *repo) DeleteRelationship(uid, related string, typ int) error {
	sh, _ := shard.Extract(uid)
	return r.store.Write(sh).Delete(&Relationship{}, "user_id=? AND related_id=? AND type=?", uid, related, typ).Error
}

func (r *repo) ListRelationships(uid string, typ, limit, offset int) ([]string, error) {
	sh, _ := shard.Extract(uid)
	type Row struct{ RelatedID string }
	var rows []Row
	dbq := r.store.Use(sh).Model(&Relationship{}).Where("user_id = ?", uid)
	if typ != 0 {
		dbq = dbq.Where("type = ?", typ)
	}
	if err := dbq.Order("created_at DESC").
		Limit(limit).Offset(offset).
		Select("related_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].RelatedID
	}
	return out, nil
}
