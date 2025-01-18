package handlers

import (
	"net/http"
	"stock-api/internal/models"

	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StockMovementHandler struct {
	db *gorm.DB
}

func NewStockMovementHandler(db *gorm.DB) *StockMovementHandler {
	return &StockMovementHandler{db: db}
}

func (h *StockMovementHandler) GetStockMovements(c *gin.Context) {
	var stockMovements []models.StockMovement
	query := h.db.Debug().Order("movement_date asc")

	// Ürün ID'sine göre filtrele
	if productID := c.Query("productId"); productID != "" {
		log.Printf("ProductID ile filtreleniyor: %s", productID)
		query = query.Where("product_id = ?", productID)
	}

	// Ürün bilgilerini de getir
	query = query.Preload("Product")

	if err := query.Find(&stockMovements).Error; err != nil {
		log.Printf("Stok hareketleri listeleme hatası: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketleri listelenemedi"})
		return
	}

	log.Printf("Bulunan stok hareketi sayısı: %d", len(stockMovements))
	c.JSON(http.StatusOK, gin.H{"data": stockMovements})
}
