package approachunitofwork

import (
	"github.com/vlaner/go-backend-examples/pgx-transactions/domain"
	"github.com/vlaner/go-backend-examples/pgx-transactions/postgres"
	"github.com/vlaner/go-backend-examples/pgx-transactions/profilerepo"
	"github.com/vlaner/go-backend-examples/pgx-transactions/userrepo"
)

type Repositories interface {
	Users() domain.UserRepository
	Profiles() domain.ProfileRepository
}

func NewRepositories(db postgres.DBTX) Repositories {
	return repositories{
		users:    userrepo.New(db),
		profiles: profilerepo.New(db),
	}
}

type repositories struct {
	users    domain.UserRepository
	profiles domain.ProfileRepository
}

func (r repositories) Users() domain.UserRepository {
	return r.users
}

func (r repositories) Profiles() domain.ProfileRepository {
	return r.profiles
}
