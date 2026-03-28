package repository

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"guitar-stock/internal/models"
)

type GuitarRepository struct {
	db *gorm.DB
}

func NewGuitarRepository(db *gorm.DB) *GuitarRepository {
	return &GuitarRepository{db: db}
}

type GuitarFilter struct {
	BrandID *uuid.UUID
	Type    *models.GuitarType
	Search  *string
	Page    int
	Limit   int
}

func (r *GuitarRepository) FindAll(filter GuitarFilter) ([]models.Guitar, int64, error) {
	var guitars []models.Guitar
	var total int64

	query := r.db.Model(&models.Guitar{})

	if filter.BrandID != nil {
		query = query.Where("brand_id = ?", *filter.BrandID)
	}

	if filter.Type != nil {
		query = query.Where("guitar_type = ?", *filter.Type)
	}

	if filter.Search != nil && *filter.Search != "" {
		searchTerm := "%" + strings.ToLower(*filter.Search) + "%"
		query = query.Where("LOWER(model) LIKE ? OR LOWER(history) LIKE ?", searchTerm, searchTerm)
	}

	query.Count(&total)

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 12
	}

	offset := (filter.Page - 1) * filter.Limit
	err := query.Preload("Brand").
		Preload("PurchaseLinks").
		Order("created_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&guitars).Error

	if err != nil {
		return guitars, total, err
	}

	for i := range guitars {
		calculatePricesFromLinks(&guitars[i])
	}

	return guitars, total, nil
}

func calculatePricesFromLinks(guitar *models.Guitar) {
	if len(guitar.PurchaseLinks) == 0 {
		return
	}

	var lowest, highest float64 = 0, 0
	inStock := false

	for _, link := range guitar.PurchaseLinks {
		if link.InStock {
			inStock = true
		}
		if link.PriceRUB != nil && *link.PriceRUB > 0 {
			if lowest == 0 || *link.PriceRUB < lowest {
				lowest = *link.PriceRUB
			}
			if *link.PriceRUB > highest {
				highest = *link.PriceRUB
			}
		}
	}

	if lowest > 0 {
		guitar.LowestPriceRUB = &lowest
	}
	if highest > 0 {
		guitar.HighestPriceRUB = &highest
	}
	guitar.InStock = inStock
}

func (r *GuitarRepository) FindByID(id uuid.UUID) (*models.Guitar, error) {
	var guitar models.Guitar
	err := r.db.Preload("Brand").
		Preload("Players").
		Preload("PurchaseLinks").
		Preload("PurchaseLinks.PriceHistory", func(db *gorm.DB) *gorm.DB {
			return db.Order("price_history.recorded_at DESC").Limit(10)
		}).
		First(&guitar, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	calculatePricesFromLinks(&guitar)
	return &guitar, nil
}

func (r *GuitarRepository) FindByBrandID(brandID uuid.UUID) ([]models.Guitar, error) {
	var guitars []models.Guitar
	err := r.db.Where("brand_id = ?", brandID).Find(&guitars).Error
	return guitars, err
}

func (r *GuitarRepository) Create(guitar *models.Guitar) error {
	return r.db.Create(guitar).Error
}

func (r *GuitarRepository) Update(guitar *models.Guitar) error {
	return r.db.Save(guitar).Error
}

func (r *GuitarRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Guitar{}, "id = ?", id).Error
}

func (r *GuitarRepository) FindAllIDs() ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.Model(&models.Guitar{}).Pluck("id", &ids).Error
	return ids, err
}

func (r *GuitarRepository) GetBrand(guitarID uuid.UUID) (*models.Brand, error) {
	var guitar models.Guitar
	err := r.db.Preload("Brand").First(&guitar, "id = ?", guitarID).Error
	if err != nil {
		return nil, err
	}
	return guitar.Brand, nil
}

func (r *GuitarRepository) UpdateImage(guitarID uuid.UUID, imageURL, imageSource string, scrapedAt *time.Time) error {
	updates := map[string]interface{}{
		"image_url":    imageURL,
		"image_source": imageSource,
	}
	if scrapedAt != nil {
		updates["image_scraped_at"] = scrapedAt
	}
	return r.db.Model(&models.Guitar{}).Where("id = ?", guitarID).Updates(updates).Error
}

func (r *GuitarRepository) FindIDsWithoutImages() ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.Model(&models.Guitar{}).
		Where("image_url IS NULL OR image_url = '' OR image_url LIKE '%placeholder%'").
		Pluck("id", &ids).Error
	return ids, err
}

func (r *GuitarRepository) FindAllForImageScrape() ([]models.Guitar, error) {
	var guitars []models.Guitar
	err := r.db.
		Where("image_url IS NULL OR image_url = '' OR image_url LIKE '%placeholder%'").
		Preload("Brand").
		Find(&guitars).Error
	return guitars, err
}
