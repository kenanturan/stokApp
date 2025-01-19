package handlers

import (
	"fmt"
	"math"
	"net/http"
	"stock-api/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SaleHandler struct {
	db *gorm.DB
}

type RecipeSaleInput struct {
	RecipeID      uint      `json:"recipeId" binding:"required"`
	Quantity      float64   `json:"quantity" binding:"required,gt=0"`
	SaleDate      time.Time `json:"saleDate" binding:"required"`
	SalePrice     float64   `json:"salePrice" binding:"required,gt=0"`
	CustomerName  string    `json:"customerName" binding:"required"`
	CustomerPhone string    `json:"customerPhone" binding:"required"`
	Note          string    `json:"note"`
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

func (h *SaleHandler) CreateRecipeSale(c *gin.Context) {
	var input RecipeSaleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz veri formatı"})
		return
	}

	// Transaction başlat
	tx := h.db.Begin()

	// Reçeteyi getir
	var recipe models.Recipe
	if err := tx.Preload("RecipeItems.Product").First(&recipe, input.RecipeID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Reçete bulunamadı"})
		return
	}

	// Ana ürünü belirle (ilk ürünü ana ürün olarak kabul ediyoruz)
	if len(recipe.RecipeItems) == 0 {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reçetede ürün bulunamadı"})
		return
	}
	mainProduct := recipe.RecipeItems[0].Product

	// Satış kaydı oluştur
	sale := models.Sale{
		ProductID:     mainProduct.ID, // Ana ürünün ID'si
		RecipeID:      &input.RecipeID,
		Quantity:      input.Quantity,
		SaleDate:      input.SaleDate,
		SalePrice:     input.SalePrice,
		CustomerName:  input.CustomerName,
		CustomerPhone: input.CustomerPhone,
		Note:          input.Note,
	}

	if err := tx.Create(&sale).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Satış kaydedilemedi"})
		return
	}

	// Her malzeme için stok kontrolü yap
	for _, item := range recipe.RecipeItems {
		var totalStock float64
		var movements []models.StockMovement

		// Aynı ürün adına sahip tüm stokları getir
		if err := tx.Joins("JOIN products ON products.id = stock_movements.product_id").
			Where("products.product_name = ? AND remaining_quantity > 0", item.Product.ProductName).
			Order("movement_date asc").
			Find(&movements).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketleri alınamadı"})
			return
		}

		// Toplam stok kontrolü
		for _, m := range movements {
			totalStock += m.RemainingQuantity
		}

		// Gerekli miktar = Reçetedeki miktar * Satış miktarı
		requiredStock := item.Quantity * input.Quantity
		if totalStock < requiredStock {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s için yetersiz stok", item.Product.ProductName)})
			return
		}
	}

	// Her malzeme için stok kullan
	for _, item := range recipe.RecipeItems {
		remaining := item.Quantity * input.Quantity
		var movements []models.StockMovement

		// FIFO mantığına göre stok kullan
		if err := tx.Joins("JOIN products ON products.id = stock_movements.product_id").
			Where("products.product_name = ? AND remaining_quantity > 0", item.Product.ProductName).
			Order("movement_date asc").
			Find(&movements).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketleri alınamadı"})
			return
		}

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
	}

	tx.Commit()

	// İlişkili verileri yükle
	h.db.Preload("Product").First(&sale, sale.ID)

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"sale":    sale,
		"recipe":  recipe,
		"message": "Reçete satışı başarılı",
	}})
}
