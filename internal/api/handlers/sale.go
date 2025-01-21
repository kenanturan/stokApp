package handlers

import (
	"fmt"
	"log"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Geçersiz veri formatı: %v", err)})
		// Debug için request body'yi logla
		var body map[string]interface{}
		c.ShouldBindJSON(&body)
		log.Printf("Frontend'den gelen hatalı istek: %+v", body)
		log.Printf("Binding hatası: %v", err)
		return
	}

	// Başarılı binding sonrası gelen veriyi logla
	log.Printf("Frontend'den gelen geçerli istek: %+v", sale)

	// Transaction başlat
	tx := h.db.Begin()

	// 1. Ürünü al
	var product models.Product
	if err := tx.Model(&models.Product{}).
		Where("id = ?", sale.ProductID).
		First(&product).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Ürün bulunamadı"})
		return
	}

	// Fiyatları hesapla
	sale.CalculatePrices()

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

func (h *SaleHandler) DeleteSale(c *gin.Context) {
	id := c.Param("id")

	// Transaction başlat
	tx := h.db.Begin()

	// Satışı bul
	var sale models.Sale
	if err := tx.First(&sale, id).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Satış bulunamadı"})
		return
	}

	// Stok kullanımlarını bul ve tarihe göre sırala
	var usages []struct {
		models.StockUsage
		MovementDate time.Time
	}
	if err := tx.Table("stock_usages").
		Select("stock_usages.*, stock_movements.movement_date").
		Joins("JOIN stock_movements ON stock_movements.id = stock_usages.stock_movement_id").
		Where("stock_usages.sale_id = ?", id).
		Order("stock_movements.movement_date DESC"). // En yeni tarihten başla
		Find(&usages).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok kullanımları alınamadı"})
		return
	}

	// Her stok kullanımı için stokları geri al (yeni tarihten eskiye doğru)
	for _, usage := range usages {
		// Stok hareketini ve ürün ID'sini al
		var stockMovement models.StockMovement
		if err := tx.First(&stockMovement, usage.StockMovementID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketi bulunamadı"})
			return
		}

		// Stok hareketini güncelle
		if err := tx.Model(&models.StockMovement{}).
			Where("id = ?", usage.StockMovementID).
			Update("remaining_quantity", gorm.Expr("remaining_quantity + ?", usage.UsedQuantity)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketi güncellenemedi"})
			return
		}

		// Ürün stoğunu güncelle
		if err := tx.Model(&models.Product{}).
			Where("id = ?", stockMovement.ProductID). // Stok hareketinin ait olduğu ürüne iade et
			Update("current_stock", gorm.Expr("current_stock + ?", usage.UsedQuantity)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ürün stoğu güncellenemedi"})
			return
		}
	}

	// Stok kullanımlarını sil
	if err := tx.Where("sale_id = ?", id).Delete(&models.StockUsage{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok kullanımları silinemedi"})
		return
	}

	// Satışı sil
	if err := tx.Delete(&sale).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Satış silinemedi"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Satış başarıyla silindi"})
}
