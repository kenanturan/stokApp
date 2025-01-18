package handlers

import (
	"math"
	"net/http"
	"stock-api/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SaleHandler struct {
	db *gorm.DB
}

func NewSaleHandler(db *gorm.DB) *SaleHandler {
	return &SaleHandler{db: db}
}

func (h *SaleHandler) CreateSale(c *gin.Context) {
	var sale models.Sale
	if err := c.ShouldBindJSON(&sale); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz veri formatı"})
		return
	}

	// Transaction başlat
	tx := h.db.Begin()

	// 1. Ürün adını al
	var productName string
	if err := tx.Model(&models.Product{}).
		Select("product_name").
		Where("id = ?", sale.ProductID).
		First(&productName).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Ürün bulunamadı"})
		return
	}

	// 2. Bu ürün adına sahip tüm stok hareketlerini getir
	var movements []models.StockMovement
	if err := tx.Table("stock_movements").
		Joins("JOIN products ON products.id = stock_movements.product_id").
		Where("products.product_name = (SELECT product_name FROM products WHERE id = ?) AND remaining_quantity > 0", sale.ProductID).
		Order("movement_date asc").
		Find(&movements).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketleri alınamadı"})
		return
	}

	// 3. Toplam stok kontrolü
	var totalStock float64
	for _, m := range movements {
		totalStock += m.RemainingQuantity
	}
	if totalStock < sale.Quantity {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Yetersiz stok"})
		return
	}

	// 4. Satışı kaydet
	if err := tx.Create(&sale).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Satış kaydedilemedi"})
		return
	}

	// 5. FIFO mantığına göre stok kullan
	remaining := sale.Quantity
	var usages []models.StockUsage
	for _, m := range movements {
		if remaining <= 0 {
			break
		}

		use := math.Min(remaining, m.RemainingQuantity)

		// Stok kullanımını kaydet
		usage := models.StockUsage{
			SaleID:          sale.ID,
			StockMovementID: m.ID,
			UsedQuantity:    use,
		}
		if err := tx.Create(&usage).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok kullanımı kaydedilemedi"})
			return
		}
		usages = append(usages, usage)

		// Stok hareketini güncelle
		if err := tx.Model(&m).Update("remaining_quantity", m.RemainingQuantity-use).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketi güncellenemedi"})
			return
		}

		// Ürün stoğunu güncelle
		if err := tx.Model(&models.Product{}).
			Where("id = ?", m.ProductID).
			Update("current_stock", gorm.Expr("current_stock - ?", use)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ürün stoğu güncellenemedi"})
			return
		}

		remaining -= use
	}

	tx.Commit()

	// İlişkili verileri yükle
	h.db.Preload("Product").First(&sale, sale.ID)
	for i := range usages {
		h.db.Preload("StockMovement").First(&usages[i], usages[i].ID)
	}

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"sale":        sale,
		"stockUsages": usages,
	}})
}

func (h *SaleHandler) GetSales(c *gin.Context) {
	var sales []models.Sale

	if err := h.db.Preload("Product").Find(&sales).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Satışlar listelenemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sales})
}
