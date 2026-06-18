package user

import (
	"context"

	"lingw/internal/domain"
)

type RegisterService struct {
	users domain.UserStore
}

func NewRegisterService(users domain.UserStore) *RegisterService {
	return &RegisterService{users: users}
}

func (s *RegisterService) Ensure(ctx context.Context, telegramID int64, username string) (domain.User, error) {
	return s.users.UpsertUser(ctx, telegramID, username)
}
