package models

import (
	"time"

	"gorm.io/gorm"
)

type Sale struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	ProductID     uint      `json:"productId" binding:"required"`
	RecipeID      *uint     `json:"recipeId,omitempty"`
	Product       Product   `json:"product" gorm:"foreignKey:ProductID;references:ID"`
	ProductData   Product   `json:"-" gorm:"-"`
	Recipe        *Recipe   `json:"recipe,omitempty" gorm:"foreignKey:RecipeID"`
	Quantity      float64   `json:"quantity" binding:"required,gt=0"`
	SaleDate      time.Time `json:"saleDate" binding:"required"`
	SalePrice     float64   `json:"salePrice" binding:"required,gt=0"`
	Discount      float64   `json:"discount" binding:"omitempty,gte=0"`
	VAT           float64   `json:"vat" binding:"omitempty,gte=0,lte=100"`
	NetPrice      float64   `json:"netPrice" gorm:"-"`
	VatAmount     float64   `json:"vatAmount" gorm:"-"`
	TotalPrice    float64   `json:"totalPrice" gorm:"-"`
	CustomerName  string    `json:"customerName" binding:"required"`
	CustomerPhone string    `json:"customerPhone" binding:"required"`
	Note          string    `json:"note"`
	UnitCost      float64   `json:"unitCost" binding:"required,gte=0"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// CalculatePrices fiyatları hesaplar
func (s *Sale) CalculatePrices() {
	// Net fiyat = (Birim fiyat × Miktar) - İskonto
	s.NetPrice = s.SalePrice * s.Quantity
	if s.Discount > 0 {
		s.NetPrice -= s.Discount
	}

	// KDV tutarı = Net fiyat × (KDV oranı / 100)
	if s.VAT > 0 {
		s.VatAmount = s.NetPrice * (s.VAT / 100.0)
	}

	// Toplam fiyat = Net fiyat + KDV tutarı
	s.TotalPrice = s.NetPrice + s.VatAmount
}

// AfterFind gorm hook'u ile fiyatları hesapla
func (s *Sale) AfterFind(*gorm.DB) error {
	s.CalculatePrices()
	return nil
}

// BeforeCreate gorm hook'u ile fiyatları hesapla
func (s *Sale) BeforeCreate(*gorm.DB) error {
	s.CalculatePrices()
	return nil
}

// RecipeSale struct'ı ekleyelim
type RecipeSale struct {
	RecipeID  uint      `json:"recipeId" binding:"required"`
	Quantity  float64   `json:"quantity" binding:"required,gt=0"`
	SaleDate  time.Time `json:"saleDate" binding:"required"`
	SalePrice float64   `json:"salePrice" binding:"required,gt=0"`
	Note      string    `json:"note"`
	Discount  float64   `json:"discount"`
	VAT       float64   `json:"vat"`
}
