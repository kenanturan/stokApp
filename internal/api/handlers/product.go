package handlers

import (
	"log"
	"net/http"
	"stock-api/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductHandler struct {
	db *gorm.DB
}

func NewProductHandler(db *gorm.DB) *ProductHandler {
	return &ProductHandler{db: db}
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz veri formatı: " + err.Error()})
		return
	}

	// Ürün oluşturma validasyonu
	if product.CompanyName == "" || product.Category == "" || product.ProductName == "" ||
		product.Unit == "" || product.InvoiceNo == "" || product.InvoiceDate.IsZero() ||
		product.InitialStock < 0 || product.CurrentStock < 0 || product.UnitPrice < 0 ||
		product.VAT < 0 || product.TotalCost < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tüm alanlar gereklidir ve sayısal değerler 0'dan büyük olmalıdır"})
		return
	}

	log.Printf("Ürün oluşturuluyor: %+v", product)
	// Transaction başlat
	tx := h.db.Begin()

	if err := tx.Create(&product).Error; err != nil {
		log.Printf("Ürün oluşturma hatası: %v", err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ürün kaydedilemedi"})
		return
	}
	log.Printf("Ürün oluşturuldu, ID: %d", product.ID)

	// İlk stok hareketini kaydet
	stockMovement := models.StockMovement{
		ProductID:         product.ID,
		InitialQuantity:   product.InitialStock,
		RemainingQuantity: product.InitialStock,
		UnitCost:          product.UnitPrice,
		MovementDate:      product.InvoiceDate,
	}

	log.Printf("Stok hareketi oluşturuluyor: %+v", stockMovement)

	if err := tx.Create(&stockMovement).Error; err != nil {
		log.Printf("Stok hareketi oluşturma hatası: %v", err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketi kaydedilemedi"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusCreated, gin.H{"data": product})
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")

	result := h.db.Delete(&models.Product{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ürün silinemedi"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ürün bulunamadı"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ürün başarıyla silindi"})
}

func (h *ProductHandler) GetProducts(c *gin.Context) {
	var products []models.Product

	if err := h.db.Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ürünler listelenemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": products})
}
