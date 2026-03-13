package relational

import (
	"context"

	"github.com/srad/mediasink/internal/db"
)

type UserStore struct{}

func NewUserStore() *UserStore {
	return &UserStore{}
}

func (s *UserStore) Create(_ context.Context, user *db.User) error {
	return db.CreateUser(user)
}

func (s *UserStore) ExistsUsername(_ context.Context, username string) (bool, error) {
	var count int64
	if err := db.DB.Model(&db.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *UserStore) FindByUsername(_ context.Context, username string) (*db.User, error) {
	return db.FindUserByUsername(username)
}

func (s *UserStore) FindByID(_ context.Context, id uint) (*db.User, error) {
	return db.FindUserByID(id)
}
