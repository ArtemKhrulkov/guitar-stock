package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"guitar-stock/internal/scraper"
)

type ScraperHandler struct {
	service *scraper.Service
}

func NewScraperHandler(service *scraper.Service) *ScraperHandler {
	return &ScraperHandler{service: service}
}

func (h *ScraperHandler) ScrapeGuitar(c *gin.Context) {
	idStr := c.Param("guitar_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid guitar ID"})
		return
	}

	links, err := h.service.ScrapeGuitar(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to scrape guitar",
			"links": []interface{}{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"links": links,
	})
}

func (h *ScraperHandler) ScrapeAll(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)

	go func() {
		defer cancel()
		h.service.ScrapeAll(ctx)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Scraping started in background",
	})
}
