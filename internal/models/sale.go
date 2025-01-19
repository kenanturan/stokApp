package models

import (
	"time"
)

type Sale struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProductID     uint      `json:"productId" binding:"required"`
	RecipeID      *uint     `json:"recipeId,omitempty"`
	Product       Product   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Recipe        *Recipe   `gorm:"foreignKey:RecipeID" json:"recipe,omitempty"`
	Quantity      float64   `json:"quantity" binding:"required,gt=0"`
	SaleDate      time.Time `json:"saleDate" binding:"required"`
	SalePrice     float64   `json:"salePrice" binding:"required,gt=0"`
	CustomerName  string    `json:"customerName" binding:"required"`
	CustomerPhone string    `json:"customerPhone" binding:"required"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
