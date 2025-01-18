package models

import (
	"time"
)

type StockMovement struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	ProductID         uint      `json:"productId"`
	Product           Product   `gorm:"foreignKey:ProductID" json:"product"`
	InitialQuantity   float64   `json:"initialQuantity"`
	RemainingQuantity float64   `json:"remainingQuantity"`
	UnitCost          float64   `json:"unitCost"`
	MovementDate      time.Time `json:"movementDate"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}
