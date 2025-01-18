package models

import (
	"time"
)

type Sale struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProductID     uint      `json:"productId" binding:"required"`
	Product       Product   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Quantity      float64   `json:"quantity" binding:"required,min=0"`
	SaleDate      time.Time `json:"saleDate" binding:"required"`
	SalePrice     float64   `json:"salePrice" binding:"required,min=0"`
	CustomerName  string    `json:"customerName" binding:"required"`
	CustomerPhone string    `json:"customerPhone" binding:"required"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
