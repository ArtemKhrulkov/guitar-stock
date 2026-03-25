package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Platform string

const (
	PlatformOzon         Platform = "ozon"
	PlatformWildberries  Platform = "wildberries"
	PlatformSweetwater   Platform = "sweetwater"
	PlatformGuitarCenter Platform = "guitarcenter"
)

type PurchaseLink struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	GuitarID     uuid.UUID      `gorm:"type:uuid;not null" json:"guitar_id"`
	Platform     Platform       `gorm:"type:varchar(50);not null" json:"platform"`
	URL          string         `gorm:"size:500;not null" json:"url"`
	PriceRUB     *float64       `gorm:"type:decimal(10,2)" json:"price_rub,omitempty"`
	PriceUSD     *float64       `gorm:"type:decimal(10,2)" json:"price_usd,omitempty"`
	InStock      bool           `gorm:"default:true" json:"in_stock"`
	IsManual     bool           `gorm:"default:false" json:"is_manual"`
	LastScraped  *time.Time     `json:"last_scraped,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	PriceHistory []PriceHistory `gorm:"foreignKey:PurchaseLinkID" json:"price_history,omitempty"`
}

func (pl *PurchaseLink) BeforeCreate(tx *gorm.DB) error {
	if pl.ID == uuid.Nil {
		pl.ID = uuid.New()
	}
	return nil
}
