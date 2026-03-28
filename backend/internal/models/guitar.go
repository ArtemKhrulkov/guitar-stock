package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuitarType string

const (
	GuitarTypeElectric GuitarType = "electric"
	GuitarTypeAcoustic GuitarType = "acoustic"
	GuitarTypeBass     GuitarType = "bass"
)

type Specifications struct {
	BodyWood     string `json:"body_wood,omitempty"`
	NeckWood     string `json:"neck_wood,omitempty"`
	Fretboard    string `json:"fretboard,omitempty"`
	PickupConfig string `json:"pickup_config,omitempty"`
	Freets       int    `json:"frets,omitempty"`
	ScaleLength  string `json:"scale_length,omitempty"`
	Hardware     string `json:"hardware,omitempty"`
	Bridge       string `json:"bridge,omitempty"`
	Tuners       string `json:"tuners,omitempty"`
}

func (s Specifications) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *Specifications) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

type Guitar struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	BrandID         uuid.UUID       `gorm:"type:uuid;not null" json:"brand_id"`
	Brand           *Brand          `gorm:"foreignKey:BrandID" json:"brand,omitempty"`
	Model           string          `gorm:"size:255;not null" json:"model"`
	GuitarType      GuitarType      `gorm:"type:varchar(50);not null" json:"guitar_type"`
	PriceRange      string          `gorm:"size:100" json:"price_range"`
	Specifications  *Specifications `gorm:"type:jsonb" json:"specifications,omitempty"`
	History         *string         `gorm:"type:text" json:"history,omitempty"`
	ImageURL        *string         `gorm:"size:500" json:"image_url,omitempty"`
	ImageSource     *string         `gorm:"size:50" json:"image_source,omitempty"`
	ImageScrapedAt  *time.Time      `json:"image_scraped_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Players         []Player        `gorm:"many2many:guitar_players;" json:"players,omitempty"`
	PurchaseLinks   []PurchaseLink  `gorm:"foreignKey:GuitarID" json:"purchase_links,omitempty"`
	LowestPriceRUB  *float64        `json:"lowest_price_rub,omitempty"`
	HighestPriceRUB *float64        `json:"highest_price_rub,omitempty"`
	InStock         bool            `json:"in_stock"`
}

func (g *Guitar) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}
