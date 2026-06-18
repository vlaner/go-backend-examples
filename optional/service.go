package main

import (
	"context"

	optional "github.com/vlaner/go-backend-examples/optional/optional"
)

type UpdateUserBioCommand struct {
	Bio optional.Optional[string]
}

type UserService struct {
	repository *UserRepository
}

func NewUserService(repository *UserRepository) *UserService {
	return &UserService{repository: repository}
}

func (s *UserService) Find(ctx context.Context, id int64) (User, error) {
	return s.repository.Find(ctx, id)
}

func (s *UserService) UpdateBio(ctx context.Context, id int64, command UpdateUserBioCommand) (User, error) {
	return s.repository.UpdateBio(ctx, id, command.Bio.Ptr())
}
