package models

import (
	"time"
)

type StockUsage struct {
	ID              uint          `gorm:"primaryKey" json:"id"`
	SaleID          uint          `json:"saleId"`
	Sale            Sale          `gorm:"foreignKey:SaleID" json:"sale"`
	StockMovementID uint          `json:"stockMovementId"`
	StockMovement   StockMovement `gorm:"foreignKey:StockMovementID" json:"stockMovement"`
	UsedQuantity    float64       `json:"usedQuantity"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}
