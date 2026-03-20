package domain

import "time"

type Product struct {
	ID          string
	Name        string
	Description string
	Price       float64
	Stock       int32
	Version     int32
	CategoryID  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Category struct {
	ID   string
	Name string
}
