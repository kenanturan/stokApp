package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"stock-api/internal/api"
	"stock-api/internal/database"
	"stock-api/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateProduct(t *testing.T) {
	db, err := database.InitDB()
	assert.NoError(t, err)

	router := api.SetupRouter(db)

	product := models.Product{
		CompanyName:  "Test Company",
		Category:     "Test Category",
		ProductName:  "Test Product",
		Unit:         "Adet",
		InvoiceNo:    "INV001",
		InitialStock: 100,
		CurrentStock: 100,
		UnitPrice:    10.5,
		VAT:          18,
		TotalCost:    1239.0,
	}

	jsonValue, _ := json.Marshal(product)
	req, _ := http.NewRequest("POST", "/api/v1/products", bytes.NewBuffer(jsonValue))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestDeleteProduct(t *testing.T) {
	db, err := database.InitDB()
	assert.NoError(t, err)

	router := api.SetupRouter(db)

	// Önce test için bir ürün oluştur
	product := models.Product{
		CompanyName:  "Test Company",
		Category:     "Test Category",
		ProductName:  "Test Product",
		Unit:         "Adet",
		InvoiceNo:    "INV001",
		InitialStock: 100,
		CurrentStock: 100,
		UnitPrice:    10.5,
		VAT:          18,
		TotalCost:    1239.0,
	}

	db.Create(&product)

	req, _ := http.NewRequest("DELETE", "/api/v1/products/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
