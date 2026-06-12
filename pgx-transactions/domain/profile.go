package domain

import "github.com/google/uuid"

type Profile struct {
	UserID           uuid.UUID
	Description      string
	Contact          string
	SocialMediaLinks []string
}
