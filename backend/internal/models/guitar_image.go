package models

import (
	"time"

	"github.com/google/uuid"
)

type GuitarImage struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	GuitarID  uuid.UUID `gorm:"type:uuid;not null" json:"guitar_id"`
	ImageURL  string    `gorm:"size:500;not null" json:"image_url"`
	Source    string    `gorm:"size:50;not null" json:"source"`
	Width     int       `json:"width,omitempty"`
	Height    int       `json:"height,omitempty"`
	IsPrimary bool      `gorm:"default:false" json:"is_primary"`
	ScrapedAt time.Time `gorm:"autoCreateTime" json:"scraped_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (g *GuitarImage) BeforeCreate(tx interface{}) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}
