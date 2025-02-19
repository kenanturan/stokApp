package main

import (
	"log"
	"stock-api/internal/api/handlers"
	"stock-api/internal/api/middleware"
	"stock-api/internal/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	log.Println("API başlatılıyor...")
	// Veritabanı bağlantısını başlat
	db, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Router ayarlanıyor...")
	// Özel router yapılandırması
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.ResponseMiddleware())

	// API versiyonu için group oluştur
	v1 := r.Group("/api/v1")

	// Router'ı ayarla
	setupRoutes(v1, db)

	log.Println("Sunucu 8080 portunda başlatılıyor...")
	// Sunucuyu başlat
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func setupRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	// Handlers'ları oluştur
	productHandler := handlers.NewProductHandler(db)
	saleHandler := handlers.NewSaleHandler(db)
	recipeHandler := handlers.NewRecipeHandler(db)

	// Products endpoints
	v1.POST("/products", productHandler.CreateProduct)
	v1.GET("/products", productHandler.GetProducts)
	v1.GET("/products/average-price", productHandler.GetAveragePrice)
	v1.GET("/products/:id", productHandler.GetProduct)
	v1.DELETE("/products/:id", productHandler.DeleteProduct)

	// Sales endpoints
	v1.POST("/sales", saleHandler.CreateSale)
	v1.GET("/sales", saleHandler.GetSales)
	v1.DELETE("/sales/:id", saleHandler.DeleteSale)

	// Recipe Sales endpoint
	v1.POST("/sales/recipe", saleHandler.CreateRecipeSale)

	// Recipe endpoints
	v1.GET("/recipes", recipeHandler.GetRecipes)
	v1.POST("/recipes", recipeHandler.CreateRecipe)
	v1.GET("/recipes/:id", recipeHandler.GetRecipe)
	v1.DELETE("/recipes/:id", recipeHandler.DeleteRecipe)
}
