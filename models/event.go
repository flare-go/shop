package models

import (
	"time"

	"github.com/stripe/stripe-go/v79"
)

type Event struct {
	ID        string           `json:"id"`
	Type      stripe.EventType `json:"type"`
	Processed bool             `json:"processed"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}
