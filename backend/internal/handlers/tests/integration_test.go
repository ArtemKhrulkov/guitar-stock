package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"guitar-stock/internal/config"
	"guitar-stock/internal/handlers"
	"guitar-stock/internal/middleware"
	"guitar-stock/internal/models"
	"guitar-stock/internal/repository"
	"guitar-stock/internal/scraper"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(postgres.Open("postgres://postgres:postgres@localhost:5432/guitar_stock_test?sslmode=disable"), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	db.Exec("DROP TABLE IF EXISTS guitar_players CASCADE")
	db.Exec("DROP TABLE IF EXISTS purchase_links CASCADE")
	db.Exec("DROP TABLE IF EXISTS guitars CASCADE")
	db.Exec("DROP TABLE IF EXISTS brands CASCADE")
	db.Exec("DROP TABLE IF EXISTS wishlists CASCADE")
	db.Exec("DROP TABLE IF EXISTS users CASCADE")
	db.Exec("DROP TABLE IF EXISTS user_verifications CASCADE")
	db.Exec("DROP TABLE IF EXISTS price_history CASCADE")

	err = db.AutoMigrate(
		&models.Brand{},
		&models.Guitar{},
		&models.Player{},
		&models.GuitarPlayer{},
		&models.PurchaseLink{},
		&models.User{},
		&models.UserVerification{},
		&models.Wishlist{},
	)
	require.NoError(t, err)

	return db
}

func setupRouter(db *gorm.DB) *gin.Engine {
	cfg := &config.Config{
		AllowedOrigins: "http://localhost:3000",
		AdminUser:      "admin",
		AdminPass:      "changeme",
	}

	scraperService := scraper.NewService(db)

	r := gin.New()

	brandRepo := repository.NewBrandRepository(db)
	guitarRepo := repository.NewGuitarRepository(db)
	playerRepo := repository.NewPlayerRepository(db)
	purchaseLinkRepo := repository.NewPurchaseLinkRepository(db)
	userRepo := repository.NewUserRepository(db)
	wishlistRepo := repository.NewWishlistRepository(db)

	brandHandler := handlers.NewBrandHandler(brandRepo)
	guitarHandler := handlers.NewGuitarHandler(guitarRepo)
	playerHandler := handlers.NewPlayerHandler(playerRepo)
	authHandler := handlers.NewAuthHandler(cfg, userRepo, wishlistRepo)
	adminHandler := handlers.NewAdminHandler(purchaseLinkRepo, guitarRepo)
	scraperHandler := handlers.NewScraperHandler(scraperService)
	authMiddleware := middleware.NewAuthMiddleware(userRepo)

	api := r.Group("/api")
	{
		api.GET("/brands", brandHandler.GetAll)
		api.GET("/brands/:id", brandHandler.GetByID)
		api.GET("/guitars", guitarHandler.GetAll)
		api.GET("/guitars/:id", guitarHandler.GetByID)
		api.GET("/players", playerHandler.GetAll)
		api.GET("/players/:id", playerHandler.GetByID)
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/logout", authHandler.Logout)
		api.GET("/auth/check", authHandler.Check)

		user := api.Group("")
		user.Use(authMiddleware.RequireAuth())
		{
			user.GET("/auth/me", authHandler.GetProfile)
			user.GET("/wishlist", authHandler.GetWishlist)
			user.POST("/wishlist", authHandler.AddToWishlist)
			user.DELETE("/wishlist/:guitar_id", authHandler.RemoveFromWishlist)
		}

		admin := api.Group("/admin")
		admin.Use(authMiddleware.RequireAdmin())
		{
			admin.POST("/guitars", guitarHandler.Create)
			admin.PATCH("/guitars/:id", guitarHandler.Update)
			admin.DELETE("/guitars/:id", guitarHandler.Delete)
			admin.GET("/links", adminHandler.GetLinks)
			admin.POST("/links", adminHandler.AddLink)
		}
	}

	_ = scraperHandler

	return r
}

func TestGetBrands(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{
		Name:    "Fender",
		Country: "USA",
	}
	db.Create(brand)

	r := setupRouter(db)

	req, _ := http.NewRequest("GET", "/api/brands", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string][]models.Brand
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["brands"])
	assert.Equal(t, "Fender", response["brands"][0].Name)
}

func TestGetBrandByID(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{
		Name:    "Gibson",
		Country: "USA",
	}
	db.Create(brand)

	r := setupRouter(db)

	req, _ := http.NewRequest("GET", "/api/brands/"+brand.ID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "Gibson", response["brand"].(map[string]interface{})["name"])
}

func TestGetBrandNotFound(t *testing.T) {
	db := setupTestDB(t)

	r := setupRouter(db)

	req, _ := http.NewRequest("GET", "/api/brands/00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetGuitars(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{Name: "Gibson", Country: "USA"}
	db.Create(brand)

	guitar := &models.Guitar{
		BrandID:    brand.ID,
		Model:      "Les Paul",
		GuitarType: models.GuitarTypeElectric,
	}
	db.Create(guitar)

	r := setupRouter(db)

	req, _ := http.NewRequest("GET", "/api/guitars", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	guitars := response["guitars"].([]interface{})
	assert.NotEmpty(t, guitars)
	assert.Equal(t, float64(1), response["total"])
}

func TestGetGuitarsWithTypeFilter(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{Name: "Ibanez", Country: "Japan"}
	db.Create(brand)

	db.Create(&models.Guitar{BrandID: brand.ID, Model: "RG550", GuitarType: models.GuitarTypeElectric})
	db.Create(&models.Guitar{BrandID: brand.ID, Model: "Artcore", GuitarType: models.GuitarTypeAcoustic})

	r := setupRouter(db)

	req, _ := http.NewRequest("GET", "/api/guitars?type=electric", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, float64(1), response["total"])
}

func TestGetGuitarByID(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{Name: "Fender", Country: "USA"}
	db.Create(brand)

	guitar := &models.Guitar{
		BrandID:    brand.ID,
		Model:      "Stratocaster",
		GuitarType: models.GuitarTypeElectric,
	}
	db.Create(guitar)

	r := setupRouter(db)

	req, _ := http.NewRequest("GET", "/api/guitars/"+guitar.ID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.NotNil(t, response["guitar"])
}

func TestGetGuitarByIDNotFound(t *testing.T) {
	db := setupTestDB(t)

	r := setupRouter(db)

	req, _ := http.NewRequest("GET", "/api/guitars/00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSearchGuitars(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{Name: "Gibson", Country: "USA"}
	db.Create(brand)

	db.Create(&models.Guitar{BrandID: brand.ID, Model: "Les Paul Standard", GuitarType: models.GuitarTypeElectric})
	db.Create(&models.Guitar{BrandID: brand.ID, Model: "SG Standard", GuitarType: models.GuitarTypeElectric})

	r := setupRouter(db)

	req, _ := http.NewRequest("GET", "/api/guitars?search=les", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, float64(1), response["total"])
}

func TestPagination(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{Name: "TestBrand", Country: "USA"}
	db.Create(brand)

	for i := 1; i <= 25; i++ {
		db.Create(&models.Guitar{
			BrandID:    brand.ID,
			Model:      fmt.Sprintf("Guitar %d", i),
			GuitarType: models.GuitarTypeElectric,
		})
	}

	r := setupRouter(db)

	req, _ := http.NewRequest("GET", "/api/guitars?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	guitars := response["guitars"].([]interface{})
	assert.Equal(t, 10, len(guitars))
	assert.Equal(t, float64(25), response["total"])
	assert.Equal(t, float64(1), response["page"])
}

func TestCreateGuitarAsAdmin(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{Name: "ESP", Country: "Japan"}
	db.Create(brand)

	name := "Admin User"
	adminUser := &models.User{
		Email:        "admin@test.com",
		PasswordHash: "$2a$10$test", // dummy hash
		Role:         "admin",
		Name:         &name,
	}
	db.Create(adminUser)

	r := setupRouter(db)
	token := loginAndGetToken(t, r, adminUser.Email, "adminpassword")

	guitarData := map[string]interface{}{
		"brand_id":    brand.ID.String(),
		"model":       "Horizon",
		"guitar_type": "electric",
	}
	body, _ := json.Marshal(guitarData)

	req, _ := http.NewRequest("POST", "/api/admin/guitars", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateGuitarUnauthorized(t *testing.T) {
	db := setupTestDB(t)

	r := setupRouter(db)

	guitarData := map[string]interface{}{
		"model":       "Test Guitar",
		"guitar_type": "electric",
	}
	body, _ := json.Marshal(guitarData)

	req, _ := http.NewRequest("POST", "/api/admin/guitars", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRegister(t *testing.T) {
	db := setupTestDB(t)

	r := setupRouter(db)

	registerData := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	}
	body, _ := json.Marshal(registerData)

	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestRegisterDuplicateEmail(t *testing.T) {
	db := setupTestDB(t)

	r := setupRouter(db)

	registerData := map[string]string{
		"email":    "duplicate@test.com",
		"password": "password123",
	}
	body, _ := json.Marshal(registerData)

	req1, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	body2, _ := json.Marshal(registerData)
	req2, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestLogin(t *testing.T) {
	db := setupTestDB(t)

	name := "Login User"
	user := &models.User{
		Email:        "login@test.com",
		PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
		Name:         &name,
	}
	db.Create(user)

	r := setupRouter(db)

	loginData := map[string]string{
		"email":    user.Email,
		"password": "password",
	}
	body, _ := json.Marshal(loginData)

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.NotEmpty(t, response["token"])
}

func TestLoginInvalidCredentials(t *testing.T) {
	db := setupTestDB(t)

	r := setupRouter(db)

	loginData := map[string]string{
		"email":    "nonexistent@test.com",
		"password": "wrongpassword",
	}
	body, _ := json.Marshal(loginData)

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetWishlist(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{Name: "Test", Country: "USA"}
	db.Create(brand)
	guitar := &models.Guitar{BrandID: brand.ID, Model: "Test Guitar", GuitarType: models.GuitarTypeElectric}
	db.Create(guitar)

	name := "Wishlist User"
	user := &models.User{
		Email:        "wishlist@test.com",
		PasswordHash: "$2a$10$test",
		Name:         &name,
	}
	db.Create(user)

	r := setupRouter(db)
	token := loginAndGetToken(t, r, user.Email, "userpassword")

	req, _ := http.NewRequest("GET", "/api/wishlist", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAddToWishlist(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{Name: "Test", Country: "USA"}
	db.Create(brand)
	guitar := &models.Guitar{BrandID: brand.ID, Model: "Test Guitar", GuitarType: models.GuitarTypeElectric}
	db.Create(guitar)

	name := "AddWish User"
	user := &models.User{
		Email:        "addwish@test.com",
		PasswordHash: "$2a$10$test",
		Name:         &name,
	}
	db.Create(user)

	r := setupRouter(db)
	token := loginAndGetToken(t, r, user.Email, "userpassword")

	body, _ := json.Marshal(map[string]string{"guitar_id": guitar.ID.String()})

	req, _ := http.NewRequest("POST", "/api/wishlist", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestRemoveFromWishlist(t *testing.T) {
	db := setupTestDB(t)

	brand := &models.Brand{Name: "Test", Country: "USA"}
	db.Create(brand)
	guitar := &models.Guitar{BrandID: brand.ID, Model: "Test Guitar", GuitarType: models.GuitarTypeElectric}
	db.Create(guitar)

	name := "RemoveWish User"
	user := &models.User{
		Email:        "removewish@test.com",
		PasswordHash: "$2a$10$test",
		Name:         &name,
	}
	db.Create(user)
	db.Create(&models.Wishlist{UserID: user.ID, GuitarID: guitar.ID})

	r := setupRouter(db)
	token := loginAndGetToken(t, r, user.Email, "userpassword")

	req, _ := http.NewRequest("DELETE", "/api/wishlist/"+guitar.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func loginAndGetToken(t *testing.T, r *gin.Engine, email, password string) string {
	loginData := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(loginData)

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var response map[string]interface{}
	if w.Code == http.StatusOK {
		json.Unmarshal(w.Body.Bytes(), &response)
		if token, ok := response["token"].(string); ok {
			return token
		}
	}

	return ""
}
