package handlers

import (
	"log"
	"math"
	"net/http"
	"stock-api/internal/models"
	"strconv"

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

	if err := h.db.Where("company_name != ?", "").Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ürünler listelenemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": products})
}

func (h *ProductHandler) GetAveragePrice(c *gin.Context) {
	// Ürün adını al
	productName := c.Query("name")
	if productName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ürün adı gerekli"})
		return
	}

	// Satış miktarını al (opsiyonel)
	quantity := 0.0
	if q := c.Query("quantity"); q != "" {
		var err error
		quantity, err = strconv.ParseFloat(q, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz miktar"})
			return
		}
	}

	// FIFO mantığına göre stok hareketlerini al
	var movements []models.StockMovement
	if err := h.db.Joins("JOIN products ON products.id = stock_movements.product_id").
		Where("products.product_name = ? AND remaining_quantity > 0", productName).
		Order("movement_date asc").
		Find(&movements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketleri alınamadı"})
		return
	}

	// Hesaplamaları yap
	var totalStock, totalValue, fifoValue float64
	var nextFIFOCost float64
	remainingQty := quantity

	for _, m := range movements {
		totalStock += m.RemainingQuantity
		totalValue += m.RemainingQuantity * m.UnitCost

		// Sonraki FIFO maliyeti
		if nextFIFOCost == 0 && m.RemainingQuantity > 0 {
			nextFIFOCost = m.UnitCost
		}

		// FIFO değeri hesapla
		if remainingQty > 0 {
			use := math.Min(remainingQty, m.RemainingQuantity)
			fifoValue += use * m.UnitCost
			remainingQty -= use
		}
	}

	// Sonuçları hazırla
	result := gin.H{
		"productName":  productName,
		"totalStock":   totalStock,
		"averagePrice": 0.0,
		"nextFIFOCost": nextFIFOCost,
	}

	if totalStock > 0 {
		result["averagePrice"] = totalValue / totalStock
	}

	if quantity > 0 {
		if quantity > totalStock {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Yetersiz stok"})
			return
		}
		result["fifoValue"] = fifoValue
		result["fifoCost"] = fifoValue / quantity
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetProduct - ID ile ürün getirme
func (h *ProductHandler) GetProduct(c *gin.Context) {
	var product models.Product
	if err := h.db.First(&product, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ürün bulunamadı"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}
