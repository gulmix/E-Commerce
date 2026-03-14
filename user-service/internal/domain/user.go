package domain

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
}
