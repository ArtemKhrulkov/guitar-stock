package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"guitar-stock/internal/config"
	"guitar-stock/internal/handlers"
	"guitar-stock/internal/middleware"
	"guitar-stock/internal/repository"
	"guitar-stock/internal/scraper"
)

func Setup(db *gorm.DB, cfg *config.Config, scraperService *scraper.Service) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.AllowedOrigins},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	brandRepo := repository.NewBrandRepository(db)
	guitarRepo := repository.NewGuitarRepository(db)
	playerRepo := repository.NewPlayerRepository(db)
	purchaseLinkRepo := repository.NewPurchaseLinkRepository(db)

	brandHandler := handlers.NewBrandHandler(brandRepo)
	guitarHandler := handlers.NewGuitarHandler(guitarRepo)
	playerHandler := handlers.NewPlayerHandler(playerRepo)
	authHandler := handlers.NewAuthHandler(cfg)
	adminHandler := handlers.NewAdminHandler(purchaseLinkRepo, guitarRepo)
	scraperHandler := handlers.NewScraperHandler(scraperService)
	imageService := scraper.NewImageService(db)
	imageHandler := handlers.NewImageHandler(imageService)

	api := r.Group("/api")
	{
		api.GET("/brands", brandHandler.GetAll)
		api.GET("/brands/:id", brandHandler.GetByID)

		api.GET("/guitars", guitarHandler.GetAll)
		api.GET("/guitars/:id", guitarHandler.GetByID)

		api.GET("/players", playerHandler.GetAll)
		api.GET("/players/:id", playerHandler.GetByID)

		api.GET("/auth/check", authHandler.Check)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/logout", authHandler.Logout)

		admin := api.Group("/admin")
		admin.Use(middleware.BasicAuth(cfg))
		{
			admin.POST("/scrape/:guitar_id", scraperHandler.ScrapeGuitar)
			admin.POST("/scrape/all", scraperHandler.ScrapeAll)
			admin.GET("/links", adminHandler.GetLinks)
			admin.POST("/links", adminHandler.AddLink)
			admin.DELETE("/links", adminHandler.DeleteLink)
			admin.POST("/guitars", guitarHandler.Create)
			admin.PATCH("/guitars/:id", guitarHandler.Update)
			admin.POST("/images/:guitar_id", imageHandler.ScrapeGuitar)
			admin.POST("/images/all", imageHandler.ScrapeAll)
			admin.GET("/images/missing", imageHandler.GetGuitarsWithoutImages)
		}
	}

	return r
}
