package models

import (
	"gofalre.io/shop/sqlc"
	"time"
)

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

func (c *Category) ConvertSqlcCategory(sqlcCategory any) *Category {

	var id uint64
	var name, description string
	var parentID *uint64
	var createdAt, updatedAt time.Time

	switch sp := sqlcCategory.(type) {
	case *sqlc.Category:
		id = uint64(sp.ID)
		name = sp.Name
		if sp.Description != nil {
			description = *sp.Description
		}
		if sp.ParentID != nil {
			categoryParentID := uint64(*sp.ParentID)
			parentID = &categoryParentID
		}
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	c.ID = id
	c.Name = name
	c.Description = description
	c.ParentID = parentID
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt

	return c
}
