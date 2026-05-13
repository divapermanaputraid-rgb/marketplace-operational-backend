package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Admin represents an admin user account.
// MVP supports single Admin role with 1-2 accounts.
type Admin struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name         string         `gorm:"type:varchar(120);not null" json:"name"`
	Email        string         `gorm:"type:varchar(160);uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"type:text;not null" json:"-"` // never expose in JSON
	Status       string         `gorm:"type:varchar(30);not null;default:'active'" json:"status"`
	LastLoginAt  *time.Time     `gorm:"type:timestamptz" json:"last_login_at"`
	CreatedAt    time.Time      `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"type:timestamptz;index" json:"-"` // soft delete
}

// TableName overrides the default table name.
func (Admin) TableName() string {
	return "admins"
}

// SetPassword hashes the given plain-text password using bcrypt and stores it.
func (a *Admin) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies a plain-text password against the stored hash.
func (a *Admin) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password))
	return err == nil
}

// IsActive returns true if the admin account status is "active".
func (a *Admin) IsActive() bool {
	return a.Status == "active"
}

// AdminResponse is the safe response DTO that excludes sensitive fields.
type AdminResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ToResponse converts an Admin model to a safe response DTO.
func (a *Admin) ToResponse() AdminResponse {
	return AdminResponse{
		ID:          a.ID,
		Name:        a.Name,
		Email:       a.Email,
		Role:        "admin",
		IsActive:    a.IsActive(),
		LastLoginAt: a.LastLoginAt,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}
