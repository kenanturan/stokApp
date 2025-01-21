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
	Discount      float64   `json:"discount" binding:"omitempty,gte=0,lte=100"`
	VAT           float64   `json:"vat" binding:"omitempty,gte=0,lte=100"`
	NetPrice      float64   `json:"netPrice" gorm:"-"`
	VATAmount     float64   `json:"vatAmount" gorm:"-"`
	TotalPrice    float64   `json:"totalPrice" gorm:"-"`
	CustomerName  string    `json:"customerName" binding:"required"`
	CustomerPhone string    `json:"customerPhone" binding:"required"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (s *Sale) CalculatePrices() {
	discountRate := s.Discount / 100
	s.NetPrice = s.SalePrice * s.Quantity * (1 - discountRate)

	vatRate := s.VAT / 100
	s.VATAmount = s.NetPrice * vatRate

	s.TotalPrice = s.NetPrice + s.VATAmount
}
