package db

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type User struct {
	UserID   uint   `json:"userId" gorm:"autoIncrement;primaryKey;column:user_id" extensions:"!x-nullable"`
	Username string `json:"username" gorm:"unique"`
	// Password holds the bcrypt hash and must never be serialized to a client.
	Password  string `json:"-"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserStore is the user aggregate. It is a concrete type: consumers declare the
// interface they need over it, per ARCHITECTURE.md section 2.
type UserStore struct {
	gorm *gorm.DB
}

func NewUserStore(conn *gorm.DB) *UserStore {
	return &UserStore{gorm: conn}
}

// Exists reports whether the username is taken.
//
// It answers (bool, error) rather than returning an error to mean "taken", so a caller
// can tell a duplicate name from an unreachable database. The duplicate signup still
// answers 500; moving it to 409 is a wire change owned by the API redesign phase, and
// internal/api/testdata/public_auth.golden pins today's answer.
func (s *UserStore) Exists(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := s.gorm.WithContext(ctx).
		Model(&User{}).
		Where("username = ?", username).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("count users named %q: %w", username, err)
	}
	return count > 0, nil
}

func (s *UserStore) Create(ctx context.Context, user *User) error {
	if err := s.gorm.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("create user %q: %w", user.Username, err)
	}
	return nil
}

// ByUsername returns the user, or gorm.ErrRecordNotFound when there is none. The
// sentinel is passed through unwrapped-at-the-top so callers can errors.Is it.
func (s *UserStore) ByUsername(ctx context.Context, username string) (*User, error) {
	var user *User
	if err := s.gorm.WithContext(ctx).
		Where("username = ?", username).
		First(&user).Error; err != nil {
		return nil, fmt.Errorf("find user %q: %w", username, err)
	}
	return user, nil
}

func (s *UserStore) ByID(ctx context.Context, id uint) (*User, error) {
	var user *User
	if err := s.gorm.WithContext(ctx).
		Where("user_id = ?", id).
		First(&user).Error; err != nil {
		return nil, fmt.Errorf("find user %d: %w", id, err)
	}
	return user, nil
}
