package models

import (
	"time"
)

type Product struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CompanyName  string    `json:"companyName"`
	Category     string    `json:"category"`
	ProductName  string    `json:"productName"`
	Unit         string    `json:"unit"`
	InvoiceNo    string    `json:"invoiceNo"`
	InvoiceDate  time.Time `json:"invoiceDate"`
	InitialStock float64   `json:"initialStock"`
	CurrentStock float64   `json:"currentStock"`
	UnitPrice    float64   `json:"unitPrice"`
	VAT          float64   `json:"vat"`
	TotalCost    float64   `json:"totalCost"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
