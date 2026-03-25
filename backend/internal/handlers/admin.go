package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"guitar-stock/internal/models"
	"guitar-stock/internal/repository"
)

type AdminHandler struct {
	purchaseLinkRepo *repository.PurchaseLinkRepository
	guitarRepo       *repository.GuitarRepository
}

func NewAdminHandler(purchaseLinkRepo *repository.PurchaseLinkRepository, guitarRepo *repository.GuitarRepository) *AdminHandler {
	return &AdminHandler{
		purchaseLinkRepo: purchaseLinkRepo,
		guitarRepo:       guitarRepo,
	}
}

type AddLinkRequest struct {
	GuitarID string          `json:"guitar_id" binding:"required"`
	Platform models.Platform `json:"platform" binding:"required"`
	URL      string          `json:"url" binding:"required,url"`
	PriceRUB *float64        `json:"price_rub"`
	PriceUSD *float64        `json:"price_usd"`
	InStock  *bool           `json:"in_stock"`
}

func (h *AdminHandler) GetLinks(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))

	links, total, err := h.purchaseLinkRepo.FindAll(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"links": links,
		"total": total,
	})
}

func (h *AdminHandler) AddLink(c *gin.Context) {
	var req AddLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	guitarID, err := uuid.Parse(req.GuitarID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid guitar_id format"})
		return
	}

	guitar, err := h.guitarRepo.FindByID(guitarID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guitar not found"})
		return
	}

	if guitar == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guitar not found"})
		return
	}

	validPlatforms := map[models.Platform]bool{
		models.PlatformOzon:         true,
		models.PlatformWildberries:  true,
		models.PlatformSweetwater:   true,
		models.PlatformGuitarCenter: true,
	}
	if !validPlatforms[req.Platform] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid platform. Must be one of: ozon, wildberries, sweetwater, guitarcenter"})
		return
	}

	inStock := true
	if req.InStock != nil {
		inStock = *req.InStock
	}

	link := &models.PurchaseLink{
		GuitarID: guitarID,
		Platform: req.Platform,
		URL:      req.URL,
		PriceRUB: req.PriceRUB,
		PriceUSD: req.PriceUSD,
		InStock:  inStock,
		IsManual: true,
	}

	if err := h.purchaseLinkRepo.Create(link); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create link: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"link": link})
}

type DeleteLinkRequest struct {
	LinkID string `json:"link_id" binding:"required"`
}

func (h *AdminHandler) DeleteLink(c *gin.Context) {
	var req DeleteLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	linkID, err := uuid.Parse(req.LinkID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid link_id format"})
		return
	}

	if err := h.purchaseLinkRepo.DeleteByID(linkID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete link: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Link deleted successfully"})
}
