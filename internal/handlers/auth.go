package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
	"github.com/marketplace-ops/backend/internal/services"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	adminRepo  *repositories.AdminRepository
	jwtService *services.JWTService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(adminRepo *repositories.AdminRepository, jwtService *services.JWTService) *AuthHandler {
	return &AuthHandler{
		adminRepo:  adminRepo,
		jwtService: jwtService,
	}
}

// LoginRequest is the body for the login endpoint.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// ChangePasswordRequest is the body for the change password endpoint.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// Login authenticates an admin and returns a JWT.
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Invalid email or password format"))
		return
	}

	admin, err := h.adminRepo.FindByEmail(req.Email)
	if err != nil {
		// Generic message to prevent email enumeration
		c.JSON(http.StatusUnauthorized, models.ErrorResponse("UNAUTHORIZED", "Invalid credentials"))
		return
	}

	if !admin.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse("UNAUTHORIZED", "Invalid credentials"))
		return
	}

	if !admin.IsActive() {
		c.JSON(http.StatusForbidden, models.ErrorResponse("FORBIDDEN", "Account is inactive"))
		return
	}

	// Update last login
	now := time.Now()
	admin.LastLoginAt = &now
	_ = h.adminRepo.Update(admin)

	// Generate token with role "admin" (MVP)
	token, err := h.jwtService.GenerateToken(admin.ID, admin.Email, "admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to generate token"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"admin": admin.ToResponse(),
		"token": token,
	}, "Login successful"))
}

// Me returns the profile of the currently authenticated admin.
// GET /api/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	adminIDRaw, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse("UNAUTHORIZED", "Not authenticated"))
		return
	}

	adminID := adminIDRaw.(uuid.UUID)
	admin, err := h.adminRepo.FindByID(adminID.String())
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Admin not found"))
		return
	}

	if !admin.IsActive() {
		c.JSON(http.StatusForbidden, models.ErrorResponse("FORBIDDEN", "Account is inactive"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"admin": admin.ToResponse(),
	}, ""))
}

// Logout invalidates the client session.
// POST /api/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Stateless JWT - client must drop the token.
	// We just return success.
	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Logged out successfully"))
}

// ChangePassword updates the password for the current admin.
// PATCH /api/auth/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	adminIDRaw, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse("UNAUTHORIZED", "Not authenticated"))
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Invalid password format. Must be at least 8 characters."))
		return
	}

	adminID := adminIDRaw.(uuid.UUID)
	admin, err := h.adminRepo.FindByID(adminID.String())
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Admin not found"))
		return
	}

	if !admin.CheckPassword(req.CurrentPassword) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse("UNAUTHORIZED", "Current password is incorrect"))
		return
	}

	if err := admin.SetPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to update password"))
		return
	}

	if err := h.adminRepo.Update(admin); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to save new password"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Password updated successfully"))
}
