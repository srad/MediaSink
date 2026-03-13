package users

import (
	"context"

	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/store"
)

type Service struct {
	users store.UserStore
}

func NewService(users store.UserStore) *Service {
	return &Service{users: users}
}

func (s *Service) GetByID(ctx context.Context, id uint) (*db.User, error) {
	return s.users.FindByID(ctx, id)
}
