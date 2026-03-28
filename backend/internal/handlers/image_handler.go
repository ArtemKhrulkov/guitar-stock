package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"guitar-stock/internal/scraper"
)

type ImageHandler struct {
	imageService *scraper.ImageService
}

func NewImageHandler(imageService *scraper.ImageService) *ImageHandler {
	return &ImageHandler{
		imageService: imageService,
	}
}

func (h *ImageHandler) ScrapeGuitar(c *gin.Context) {
	guitarIDStr := c.Param("guitar_id")
	guitarID, err := uuid.Parse(guitarIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid guitar ID"})
		return
	}

	result, err := h.imageService.ScrapeGuitar(c.Request.Context(), guitarID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "No image found for this guitar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Image scraped successfully",
		"image_url": result.URL,
		"source":    result.Source,
		"width":     result.Width,
		"height":    result.Height,
	})
}

func (h *ImageHandler) ScrapeAll(c *gin.Context) {
	result, err := h.imageService.ScrapeAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ImageHandler) GetGuitarsWithoutImages(c *gin.Context) {
	ids, err := h.imageService.GetGuitarsWithoutImages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(ids),
		"ids":   ids,
	})
}
