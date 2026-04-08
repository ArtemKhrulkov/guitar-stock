package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"guitar-stock/internal/config"
	"guitar-stock/internal/models"
	"guitar-stock/internal/repository"
)

type AuthHandler struct {
	cfg          *config.Config
	userRepo     *repository.UserRepository
	wishlistRepo *repository.WishlistRepository
}

func NewAuthHandler(cfg *config.Config, userRepo *repository.UserRepository, wishlistRepo *repository.WishlistRepository) *AuthHandler {
	return &AuthHandler{
		cfg:          cfg,
		userRepo:     userRepo,
		wishlistRepo: wishlistRepo,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required,len=6"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	Name          *string   `json:"name"`
	Role          string    `json:"role"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

func generateOTP() string {
	r := time.Now().UnixNano() % 1000000
	return strconv.FormatInt(r, 10)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	existingUser, err := h.userRepo.FindByEmailIncludeUnverified(req.Email)
	if err == nil && existingUser != nil {
		if existingUser.EmailVerified {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
			return
		}
		// User exists but not verified - delete and recreate
		_ = h.userRepo.HardDelete(existingUser.ID)
	}

	// Hash password
	hashedPassword, err := h.userRepo.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user (unverified)
	user := &models.User{
		Email:         req.Email,
		PasswordHash:  hashedPassword,
		EmailVerified: false,
		Role:          "user",
	}
	if req.Name != "" {
		user.Name = &req.Name
	}

	if err := h.userRepo.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate OTP
	otp := generateOTP()
	verification := &models.UserVerification{
		UserID:    user.ID,
		Code:      otp,
		Type:      "email_verify",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := h.userRepo.CreateVerification(verification); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create verification"})
		return
	}

	// Log OTP to console (dev mode)
	if h.cfg.GinMode == "debug" {
		println("\n=== EMAIL VERIFICATION OTP ===")
		println("Email:", req.Email)
		println("OTP:", otp)
		println("Expires in 10 minutes")
		println("==============================\n")
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email for OTP code.",
		"email":   req.Email,
	})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Find valid verification
	verification, err := h.userRepo.FindValidVerification(user.ID, req.Code, "email_verify")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// Mark verification as used
	_ = h.userRepo.MarkVerificationUsed(verification.ID)

	// Mark user as verified
	user.EmailVerified = true
	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify email"})
		return
	}

	// Set auth cookie
	h.setAuthCookie(c, user.ID.String())

	c.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully",
		"user":    mapUserToResponse(user),
	})
}

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user (must be verified)
	user, err := h.userRepo.FindUserByEmailForReset(req.Email)
	if err != nil {
		// Don't reveal if user exists
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists and is verified, you will receive a reset code"})
		return
	}

	// Delete old verifications
	_ = h.userRepo.DeleteExpiredVerifications(user.ID)

	// Generate OTP
	otp := generateOTP()
	verification := &models.UserVerification{
		UserID:    user.ID,
		Code:      otp,
		Type:      "password_reset",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := h.userRepo.CreateVerification(verification); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create verification"})
		return
	}

	// Log OTP to console (dev mode)
	if h.cfg.GinMode == "debug" {
		println("\n=== PASSWORD RESET OTP ===")
		println("Email:", req.Email)
		println("OTP:", otp)
		println("Expires in 10 minutes")
		println("==========================\n")
	}

	c.JSON(http.StatusOK, gin.H{"message": "If the email exists and is verified, you will receive a reset code"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Find valid verification
	verification, err := h.userRepo.FindValidVerification(user.ID, req.Code, "password_reset")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// Hash new password
	hashedPassword, err := h.userRepo.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update password
	user.PasswordHash = hashedPassword
	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Mark verification as used
	_ = h.userRepo.MarkVerificationUsed(verification.ID)
	_ = h.userRepo.DeleteExpiredVerifications(user.ID)

	// Set auth cookie
	h.setAuthCookie(c, user.ID.String())

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
		"user":    mapUserToResponse(user),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check admin credentials first (backward compatibility)
	if req.Email == h.cfg.AdminUser && req.Password == h.cfg.AdminPass {
		// Admin login - check if admin user exists, create if not
		adminUser, err := h.userRepo.FindByEmail("admin@" + h.cfg.AdminUser + ".local")
		if err != nil {
			// Create admin user
			hashedPassword, _ := h.userRepo.HashPassword(req.Password)
			adminUser = &models.User{
				Email:         "admin@" + h.cfg.AdminUser + ".local",
				PasswordHash:  hashedPassword,
				EmailVerified: true,
				Role:          "admin",
			}
			_ = h.userRepo.Create(adminUser)
		}

		h.setAuthCookie(c, adminUser.ID.String())
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"user":    mapUserToResponse(adminUser),
		})
		return
	}

	// Regular user login
	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !h.userRepo.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.EmailVerified {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Please verify your email first"})
		return
	}

	h.setAuthCookie(c, user.ID.String())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    mapUserToResponse(user),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	switch h.cfg.CookieSameSite {
	case "strict":
		c.SetSameSite(http.SameSiteStrictMode)
	case "none":
		c.SetSameSite(http.SameSiteNoneMode)
	default:
		c.SetSameSite(http.SameSiteLaxMode)
	}

	c.SetCookie(
		"auth_token",
		"",
		-1,
		"/",
		"",
		h.cfg.CookieSecure,
		true,
	)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AuthHandler) Check(c *gin.Context) {
	userID, err := h.getUserIDFromCookie(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"authenticated": false})
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"authenticated": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"user":          mapUserToResponse(user),
		"expires":       time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, err := h.getUserIDFromCookie(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": mapUserToResponse(user)})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, err := h.getUserIDFromCookie(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	type UpdateRequest struct {
		Name *string `json:"name"`
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if req.Name != nil {
		user.Name = req.Name
	}

	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": mapUserToResponse(user)})
}

// Wishlist handlers
func (h *AuthHandler) GetWishlist(c *gin.Context) {
	userID, err := h.getUserIDFromCookie(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	wishlists, err := h.wishlistRepo.FindByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wishlist"})
		return
	}

	var guitarIDs []string
	for _, w := range wishlists {
		guitarIDs = append(guitarIDs, w.GuitarID.String())
	}

	c.JSON(http.StatusOK, gin.H{"guitar_ids": guitarIDs, "count": len(guitarIDs)})
}

func (h *AuthHandler) AddToWishlist(c *gin.Context) {
	userID, err := h.getUserIDFromCookie(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	type AddWishlistRequest struct {
		GuitarID string `json:"guitar_id" binding:"required"`
	}

	var req AddWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	guitarID, err := uuid.Parse(req.GuitarID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid guitar ID"})
		return
	}

	// Check if already exists
	existing, _ := h.wishlistRepo.FindByUserAndGuitar(userID, guitarID)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Already in wishlist"})
		return
	}

	wishlist := &models.Wishlist{
		UserID:   userID,
		GuitarID: guitarID,
	}

	if err := h.wishlistRepo.Create(wishlist); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to wishlist"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Added to wishlist"})
}

func (h *AuthHandler) RemoveFromWishlist(c *gin.Context) {
	userID, err := h.getUserIDFromCookie(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	guitarIDStr := c.Param("guitar_id")
	guitarID, err := uuid.Parse(guitarIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid guitar ID"})
		return
	}

	if err := h.wishlistRepo.DeleteByUserAndGuitar(userID, guitarID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove from wishlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Removed from wishlist"})
}

// Helper functions
func (h *AuthHandler) setAuthCookie(c *gin.Context, userID string) {
	cookieMaxAge := 7 * 24 * 60 * 60 // 7 days in seconds

	// Set SameSite based on config
	switch h.cfg.CookieSameSite {
	case "strict":
		c.SetSameSite(http.SameSiteStrictMode)
	case "none":
		c.SetSameSite(http.SameSiteNoneMode)
	default: // lax
		c.SetSameSite(http.SameSiteLaxMode)
	}

	c.SetCookie(
		"auth_token",
		userID,
		cookieMaxAge,
		"/",
		"",
		h.cfg.CookieSecure,
		true,
	)
}

func (h *AuthHandler) getUserIDFromCookie(c *gin.Context) (uuid.UUID, error) {
	cookie, err := c.Cookie("auth_token")
	if err != nil || cookie == "" {
		return uuid.Nil, errors.New("no cookie")
	}

	userID, err := uuid.Parse(cookie)
	if err != nil {
		return uuid.Nil, errors.New("invalid cookie")
	}

	return userID, nil
}

func mapUserToResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:            user.ID,
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt,
	}
}
