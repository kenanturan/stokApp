package handlers

import (
	"log"
	"net/http"
	"stock-api/internal/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RecipeHandler struct {
	db *gorm.DB
}

func NewRecipeHandler(db *gorm.DB) *RecipeHandler {
	return &RecipeHandler{db: db}
}

func (h *RecipeHandler) CreateRecipe(c *gin.Context) {
	var recipe models.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz veri formatı"})
		return
	}

	// Transaction başlat
	tx := h.db.Begin()

	// Reçete kalemlerini geçici olarak sakla
	recipeItems := recipe.RecipeItems
	recipe.RecipeItems = nil

	// Reçeteyi kaydet
	if err := tx.Create(&recipe).Error; err != nil {
		log.Printf("DEBUG - Reçete kaydedilemedi: %+v", err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reçete kaydedilemedi"})
		return
	}

	log.Printf("DEBUG - Reçete kaydedildi: %+v", recipe)

	// Reçete kalemlerini toplu kaydet
	for i := range recipeItems {
		recipeItems[i].RecipeID = recipe.ID
	}
	if err := tx.Create(&recipeItems).Error; err != nil {
		log.Printf("DEBUG - Reçete kalemleri kaydedilemedi: %+v", err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()

	// İlişkili verileri yükle
	h.db.Preload("RecipeItems.Product").First(&recipe, recipe.ID)

	c.JSON(http.StatusCreated, gin.H{"data": recipe})
}

func (h *RecipeHandler) GetRecipes(c *gin.Context) {
	var recipes []models.Recipe

	if err := h.db.Preload("RecipeItems.Product").Find(&recipes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reçeteler listelenemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": recipes})
}

func (h *RecipeHandler) ProduceFromRecipe(c *gin.Context) {
	recipeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz reçete ID"})
		return
	}

	// Reçeteyi getir
	var recipe models.Recipe
	if err := h.db.Preload("RecipeItems.Product").First(&recipe, recipeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reçete bulunamadı"})
		return
	}

	var input struct {
		Quantity float64   `json:"quantity" binding:"required,gt=0"`
		Date     time.Time `json:"date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz veri formatı"})
		return
	}

	// Geçici yanıt
	c.JSON(http.StatusOK, gin.H{
		"message":  "Üretim başarılı",
		"recipe":   recipe,
		"quantity": input.Quantity,
		"date":     input.Date,
	})
}

func (h *RecipeHandler) DeleteRecipe(c *gin.Context) {
	id := c.Param("id")

	// Transaction başlat
	tx := h.db.Begin()

	// Reçeteyi bul
	var recipe models.Recipe
	if err := tx.First(&recipe, id).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Reçete bulunamadı"})
		return
	}

	// Reçeteyi sil (RecipeItems cascade ile otomatik silinecek)
	if err := tx.Delete(&recipe).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reçete silinemedi"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Reçete başarıyla silindi"})
}
