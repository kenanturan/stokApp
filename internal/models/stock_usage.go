package models

import (
	"time"
)

type StockUsage struct {
	ID              uint      `json:"id" gorm:"primarykey"`
	SaleID          uint      `json:"saleId"`
	StockMovementID uint      `json:"stockMovementId"`
	UsedQuantity    float64   `json:"usedQuantity"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
