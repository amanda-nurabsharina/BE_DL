package customers

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Customer struct {
	ID                  uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Code                string         `gorm:"uniqueIndex;not null" json:"code"`
	Name                string         `gorm:"not null" json:"name"` // Doctor name
	ClinicName          string         `json:"clinic_name"`
	Phone               string         `gorm:"not null" json:"phone"`
	Email               string         `json:"email"`
	Address             string         `json:"address"`
	Notes               string         `json:"notes"`
	FavoriteShade       string         `json:"favorite_shade"`
	FavoriteDentureType string         `json:"favorite_denture_type"`
	LoyaltyTier         string         `gorm:"default:'REGULAR'" json:"loyalty_tier"` // REGULAR, SILVER, GOLD, VIP
	DiscountPercent     float64        `gorm:"default:0" json:"discount_percent"`
	Preferences         string         `gorm:"type:jsonb" json:"preferences"`
	IsActive            bool           `json:"is_active"`
	TotalOrders         int            `gorm:"-" json:"total_orders"`
	TotalSpent          float64        `gorm:"-" json:"total_spent"`
	LastOrderDate       *time.Time     `gorm:"-" json:"last_order_date"`
	DaysSinceLastOrder  int            `gorm:"-" json:"days_since_last_order"`
	PatientsCount       int            `gorm:"-" json:"patients_count"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Customer) TableName() string {
	return "customers"
}

type Patient struct {
	ID               uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Code             string         `gorm:"uniqueIndex;not null" json:"code"`
	CustomerID       *uuid.UUID     `json:"customer_id"`
	Customer         *Customer      `gorm:"foreignKey:CustomerID" json:"customer"`
	Name             string         `gorm:"not null" json:"name"`
	Gender           string         `gorm:"default:'L'" json:"gender"` // L or P
	Age              int            `gorm:"default:0" json:"age"`
	Phone            string         `json:"phone"`
	DefaultShade     string         `gorm:"default:'VITA A2'" json:"default_shade"`
	LastTeethNumbers string         `json:"last_teeth_numbers"`
	AllergiesOrNotes string         `json:"allergies_or_notes"`
	IsActive         bool           `gorm:"default:true" json:"is_active"`
	TotalOrders      int            `gorm:"-" json:"total_orders"`
	LastOrderDate    *time.Time     `gorm:"-" json:"last_order_date"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Patient) TableName() string {
	return "patients"
}

type CreateCustomerRequest struct {
	Name                string  `json:"name"`
	ClinicName          string  `json:"clinic_name"`
	Phone               string  `json:"phone"`
	Email               string  `json:"email"`
	Address             string  `json:"address"`
	Notes               string  `json:"notes"`
	FavoriteShade       string  `json:"favorite_shade"`
	FavoriteDentureType string  `json:"favorite_denture_type"`
	LoyaltyTier         string  `json:"loyalty_tier"`
	DiscountPercent     float64 `json:"discount_percent"`
	Preferences         string  `json:"preferences"`
}

type UpdateCustomerRequest struct {
	Name                *string  `json:"name"`
	ClinicName          *string  `json:"clinic_name"`
	Phone               *string  `json:"phone"`
	Email               *string  `json:"email"`
	Address             *string  `json:"address"`
	Notes               *string  `json:"notes"`
	FavoriteShade       *string  `json:"favorite_shade"`
	FavoriteDentureType *string  `json:"favorite_denture_type"`
	LoyaltyTier         *string  `json:"loyalty_tier"`
	DiscountPercent     *float64 `json:"discount_percent"`
	Preferences         *string  `json:"preferences"`
	IsActive            *bool    `json:"is_active"`
}

type CreatePatientRequest struct {
	CustomerID       string `json:"customer_id"`
	Name             string `json:"name"`
	Gender           string `json:"gender"`
	Age              int    `json:"age"`
	Phone            string `json:"phone"`
	DefaultShade     string `json:"default_shade"`
	LastTeethNumbers string `json:"last_teeth_numbers"`
	AllergiesOrNotes string `json:"allergies_or_notes"`
}

type UpdatePatientRequest struct {
	CustomerID       *string `json:"customer_id"`
	Name             *string `json:"name"`
	Gender           *string `json:"gender"`
	Age              *int    `json:"age"`
	Phone            *string `json:"phone"`
	DefaultShade     *string `json:"default_shade"`
	LastTeethNumbers *string `json:"last_teeth_numbers"`
	AllergiesOrNotes *string `json:"allergies_or_notes"`
	IsActive         *bool   `json:"is_active"`
}
