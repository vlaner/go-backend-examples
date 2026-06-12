package profilerepo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/vlaner/go-backend-examples/pgx-transactions/domain"
	"github.com/vlaner/go-backend-examples/pgx-transactions/postgres"
)

type dbProfile struct {
	UserID           uuid.UUID `db:"user_id"`
	Description      string    `db:"description"`
	Contact          string    `db:"contact"`
	SocialMediaLinks []string  `db:"social_media_links"`
}

func (p dbProfile) ToDomain() domain.Profile {
	return domain.Profile{
		UserID:           p.UserID,
		Description:      p.Description,
		Contact:          p.Contact,
		SocialMediaLinks: p.SocialMediaLinks,
	}
}

type repository struct {
	db postgres.DBTX
}

func New(db postgres.DBTX) domain.ProfileRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, profile domain.Profile) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO profiles (user_id, description, contact, social_media_links)
		VALUES ($1, $2, $3, $4)
	`, profile.UserID, profile.Description, profile.Contact, profile.SocialMediaLinks)
	if err != nil {
		return fmt.Errorf("create profile: %w", err)
	}

	return nil
}

func (r *repository) GetByUserID(ctx context.Context, userID uuid.UUID) (domain.Profile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, description, contact, social_media_links
		FROM profiles
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("query profile by user id: %w", err)
	}

	profile, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dbProfile])
	if err != nil {
		return domain.Profile{}, fmt.Errorf("collect profile by user id: %w", err)
	}

	return profile.ToDomain(), nil
}
