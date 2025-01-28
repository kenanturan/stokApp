package handlers

import (
	"log"
	"math"
	"net/http"
	"stock-api/internal/models"
	"strconv"
	"time"

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

func (h *ProductHandler) GetAveragePrice(c *gin.Context) {
	productName := c.Query("name")
	quantity := c.DefaultQuery("quantity", "0") // Opsiyonel satış miktarı

	if productName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ürün adı gerekli"})
		return
	}

	// Satış miktarını parse et
	saleQuantity, err := strconv.ParseFloat(quantity, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz miktar"})
		return
	}

	// Stok hareketlerini FIFO sırasına göre al
	var movements []struct {
		RemainingQuantity float64
		UnitCost          float64
		MovementDate      time.Time
		ProductName       string
	}

	err = h.db.Raw(`
		SELECT sm.remaining_quantity, sm.unit_cost, sm.movement_date, p.product_name
		FROM stock_movements sm
		JOIN products p ON p.id = sm.product_id
		WHERE p.product_name = ?
		AND sm.remaining_quantity > 0
		ORDER BY sm.created_at ASC
	`, productName).Scan(&movements).Error

	// Debug için hareketleri logla
	log.Printf("SQL Parametreleri: %s", productName)
	log.Printf("Toplam hareket sayısı: %d", len(movements))
	for i, m := range movements {
		log.Printf("Hareket %d: Ürün=%s, Miktar=%.2f, Maliyet=%.2f, Tarih=%v",
			i+1, m.ProductName, m.RemainingQuantity, m.UnitCost, m.MovementDate)
	}

	// Debug için toplam stok kontrolü
	var dbStock float64
	err = h.db.Raw(`
		SELECT COALESCE(SUM(remaining_quantity), 0)
		FROM stock_movements sm
		JOIN products p ON p.id = sm.product_id
		WHERE p.product_name = ?
		AND sm.remaining_quantity > 0
	`, productName).Scan(&dbStock).Error
	log.Printf("Veritabanındaki toplam stok: %.2f", dbStock)

	// FIFO mantığına göre hesaplama
	var totalStock float64
	var totalValue float64
	var nextFIFOCost float64
	var fifoValue float64 // Satış miktarına göre FIFO değeri
	var remainingQty float64 = saleQuantity

	for _, m := range movements {
		log.Printf("FIFO Hesaplama - Kalan Miktar: %.2f, Kullanılacak: %.2f, Birim Maliyet: %.2f",
			remainingQty, math.Min(remainingQty, m.RemainingQuantity), m.UnitCost)

		totalStock += m.RemainingQuantity
		totalValue += m.RemainingQuantity * m.UnitCost

		// Bir sonraki birim FIFO maliyeti
		if nextFIFOCost == 0 && m.RemainingQuantity > 0 {
			nextFIFOCost = m.UnitCost
		}

		// Satış miktarına göre FIFO hesaplama
		if remainingQty > 0 {
			useQty := math.Min(remainingQty, m.RemainingQuantity)
			fifoValue += useQty * m.UnitCost
			remainingQty -= useQty
			log.Printf("FIFO Değer Eklendi: %.2f × %.2f = %.2f, Toplam: %.2f",
				useQty, m.UnitCost, useQty*m.UnitCost, fifoValue)
		}
	}

	var result struct {
		AveragePrice    float64
		TotalStock      float64
		TotalStockValue float64
		NextFIFOCost    float64
		FIFOCost        float64 // Satış miktarına göre ortalama FIFO maliyeti
	}

	if totalStock > 0 {
		result.AveragePrice = totalValue / totalStock
	}

	if saleQuantity > totalStock {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Yetersiz stok"})
		return
	}

	if saleQuantity > 0 && saleQuantity <= totalStock {
		result.FIFOCost = fifoValue / saleQuantity
	}

	result.TotalStock = totalStock
	result.TotalStockValue = totalValue
	result.NextFIFOCost = nextFIFOCost

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"productName":     productName,
			"averagePrice":    result.AveragePrice,
			"totalStock":      result.TotalStock,
			"totalStockValue": result.TotalStockValue,
			"nextFIFOCost":    result.NextFIFOCost,
			"fifoCost":        result.FIFOCost,
			"saleQuantity":    saleQuantity,
		},
	})
}
