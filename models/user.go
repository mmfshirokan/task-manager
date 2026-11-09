package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// Token is a signed bearer access token issued on login.
type Token struct {
	AccessToken string
	ExpiresIn   time.Duration
}
