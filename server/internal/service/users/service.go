package users

import (
	"context"

	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/store"
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
