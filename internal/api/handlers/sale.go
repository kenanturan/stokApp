package handlers

import (
	"bytes"
	"io"
	"math"
	"net/http"
	"stock-api/internal/models"
	"time"

	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SaleHandler struct {
	db *gorm.DB
}

type RecipeSaleInput struct {
	RecipeID  uint      `json:"recipeId" binding:"required"`
	Quantity  float64   `json:"quantity" binding:"required,gt=0"`
	SaleDate  time.Time `json:"saleDate" binding:"required"`
	SalePrice float64   `json:"salePrice" binding:"required,gt=0"`
	UnitCost  float64   `json:"unitCost" binding:"required,gte=0"`
	Note      string    `json:"note"`
	Discount  float64   `json:"discount"`
	VAT       float64   `json:"vat"`
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

	// Validasyonlar
	if sale.ProductID == 0 || sale.Quantity <= 0 || sale.SalePrice < 0 ||
		sale.CustomerName == "" || sale.CustomerPhone == "" ||
		sale.UnitCost < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz değerler"})
		return
	}

	// Transaction başlat
	tx := h.db.Begin()

	// Önce ürünü kontrol et
	var product models.Product
	if err := tx.First(&product, sale.ProductID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ürün bulunamadı"})
		return
	}

	// FIFO için stok hareketlerini al
	var movements []models.StockMovement
	query := tx.Debug().
		Raw(`
			SELECT sm.* 
			FROM stock_movements sm
			JOIN products p1 ON sm.product_id = p1.id
			JOIN products p2 ON p1.product_name = p2.product_name
			WHERE p2.id = ?
			AND sm.remaining_quantity > 0
			ORDER BY sm.movement_date ASC
		`, sale.ProductID)

	log.Printf("SQL Sorgusu: %v", query.Statement.SQL.String())
	log.Printf("Parametreler: %v", query.Statement.Vars)

	if err := query.Scan(&movements).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketleri alınamadı"})
		return
	}

	// Debug için stok hareketlerini logla
	log.Printf("Bulunan stok hareketleri: %+v", movements)
	log.Printf("Toplam hareket sayısı: %d", len(movements))

	// Toplam stok kontrolü
	var totalStock float64
	for _, m := range movements {
		totalStock += m.RemainingQuantity
		log.Printf("Hareket ID: %d, Product ID: %d, Kalan miktar: %f, Ürün: %s",
			m.ID, m.ProductID, m.RemainingQuantity, product.ProductName)
	}

	log.Printf("Toplam stok: %f, İstenen miktar: %f", totalStock, sale.Quantity)

	if totalStock < sale.Quantity {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Yetersiz stok"})
		return
	}

	// Satışı kaydet
	if err := tx.Create(&sale).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Satış kaydedilemedi"})
		return
	}

	// Satış ve ürün detaylarını yükle
	var completeSale models.Sale
	if err := tx.Debug().
		Model(&models.Sale{}).
		Where("id = ?", sale.ID).
		First(&completeSale).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Satış detayları alınamadı"})
		return
	}

	// Product bilgilerini set et
	completeSale.Product = product

	// Debug için satış detaylarını logla
	log.Printf("Ürün: %+v", product)
	log.Printf("Satış ID: %d", completeSale.ID)
	log.Printf("Ürün ID: %d", completeSale.ProductID)
	log.Printf("Product: %+v", completeSale.Product)
	log.Printf("Satış detayları: %+v", completeSale)

	// FIFO mantığına göre stok düşümü
	remaining := sale.Quantity
	var stockUsages []models.StockUsage

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

		// Stok kullanımını tam olarak yükle
		if err := tx.First(&usage, usage.ID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok kullanımı detayları alınamadı"})
			return
		}

		stockUsages = append(stockUsages, usage)

		// Stok hareketini güncelle
		if err := tx.Model(&models.StockMovement{}).
			Where("id = ?", m.ID).
			Update("remaining_quantity", gorm.Expr("remaining_quantity - ?", use)).Error; err != nil {
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

	// Fiyatları hesapla
	completeSale.CalculatePrices()

	// Response'u direkt olarak yaz
	c.Set("response", completeSale)
	c.Writer.WriteHeader(http.StatusCreated)
}

func (h *SaleHandler) GetSales(c *gin.Context) {
	var sales []models.Sale

	// Tüm satışları yükle
	if err := h.db.Preload("Product").
		Preload("Recipe").
		Find(&sales).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Satışlar listelenemedi"})
		return
	}

	// Her satış için düzenleme yap
	for i := range sales {
		// Eğer bu bir reçete satışı ise
		if sales[i].RecipeID != nil && sales[i].Recipe != nil {
			// Ürün yerine reçete adını göster
			sales[i].Product = models.Product{
				Category:    "Reçete",
				ProductName: "Reçete: " + sales[i].Recipe.Name,
			}
		}
		sales[i].CalculatePrices()
	}

	// Response'u middleware'e bırak
	c.Set("response", gin.H{
		"sales": sales,
	})
}

func (h *SaleHandler) CreateRecipeSale(c *gin.Context) {
	var recipeSale models.RecipeSale

	// Request body'yi logla
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Body okuma hatası: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Request body okunamadı"})
		return
	}
	log.Printf("Gelen request body: %s", string(body))
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	if err := c.ShouldBindJSON(&recipeSale); err != nil {
		log.Printf("Binding hatası: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz veri formatı: " + err.Error()})
		return
	}

	log.Printf("Reçete satışı verisi: %+v", recipeSale)

	// Transaction başlat
	tx := h.db.Begin()

	// Reçeteyi getir
	var recipe models.Recipe
	if err := tx.Preload("RecipeItems").
		Preload("RecipeItems.Product").
		First(&recipe, recipeSale.RecipeID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Reçete bulunamadı"})
		return
	}

	log.Printf("Bulunan reçete: %+v", recipe)

	// Stok kontrolü yap
	for _, item := range recipe.RecipeItems {
		itemQuantity := item.Quantity * recipeSale.Quantity

		// FIFO için stok hareketlerini al
		var movements []models.StockMovement
		query := tx.Debug().
			Raw(`
				SELECT sm.* 
				FROM stock_movements sm
				JOIN products p1 ON sm.product_id = p1.id
				JOIN products p2 ON p1.product_name = p2.product_name
				WHERE p2.id = ?
				AND sm.remaining_quantity > 0
				ORDER BY sm.movement_date ASC
			`, item.ProductID)

		log.Printf("SQL Sorgusu: %v", query.Statement.SQL.String())
		log.Printf("Parametreler: %v", query.Statement.Vars)

		if err := query.Scan(&movements).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketleri alınamadı"})
			return
		}

		// Debug için stok hareketlerini logla
		log.Printf("Bulunan stok hareketleri: %+v", movements)
		log.Printf("Toplam hareket sayısı: %d", len(movements))

		// Toplam stok kontrolü
		var totalStock float64
		for _, m := range movements {
			totalStock += m.RemainingQuantity
			log.Printf("Hareket ID: %d, Kalan miktar: %f", m.ID, m.RemainingQuantity)
		}

		log.Printf("Toplam stok: %f, İstenen miktar: %f", totalStock, itemQuantity)

		if totalStock < itemQuantity {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Yetersiz stok: " + item.Product.ProductName})
			return
		}
	}

	// Tek bir satış kaydı oluştur
	sale := models.Sale{
		RecipeID:  &recipe.ID,
		Quantity:  recipeSale.Quantity,
		SaleDate:  recipeSale.SaleDate,
		SalePrice: recipeSale.SalePrice,
		UnitCost:  recipeSale.UnitCost,
		Note:      recipeSale.Note,
		Discount:  recipeSale.Discount,
		VAT:       recipeSale.VAT,
		Product: models.Product{ // Reçete bilgilerini Product'a ekle
			Category:    "Reçete",
			ProductName: "Reçete: " + recipe.Name,
		},
	}

	// Satışı kaydet
	if err := tx.Create(&sale).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Satış kaydedilemedi"})
		return
	}

	// Stok düşümlerini yap
	var allStockUsages []models.StockUsage
	for _, item := range recipe.RecipeItems {
		itemQuantity := item.Quantity * recipeSale.Quantity

		// FIFO için stok hareketlerini al
		var movements []models.StockMovement
		query := tx.Debug().
			Raw(`
				SELECT sm.* 
				FROM stock_movements sm
				JOIN products p1 ON sm.product_id = p1.id
				JOIN products p2 ON p1.product_name = p2.product_name
				WHERE p2.id = ?
				AND sm.remaining_quantity > 0
				ORDER BY sm.movement_date ASC
			`, item.ProductID)

		if err := query.Scan(&movements).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok hareketleri alınamadı"})
			return
		}

		// FIFO mantığına göre stok düşümü
		remaining := itemQuantity
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

			allStockUsages = append(allStockUsages, usage)

			// Stok hareketini güncelle
			if err := tx.Model(&models.StockMovement{}).
				Where("id = ?", m.ID).
				Update("remaining_quantity", gorm.Expr("remaining_quantity - ?", use)).Error; err != nil {
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

	// Response'u hazırla
	c.Set("response", gin.H{
		"sale":        sale,
		"recipe":      recipe,
		"stockUsages": allStockUsages,
	})
	c.Set("status", http.StatusCreated)
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
