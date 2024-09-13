package models

import "time"

type Category struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ParentID    *uint64   `json:"parent_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CategoryTree struct {
	*Category
	Children []*CategoryTree `json:"children,omitempty"`
}
