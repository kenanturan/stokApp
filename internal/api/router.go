package api

import (
	"stock-api/internal/api/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(v1 *gin.RouterGroup, db *gorm.DB) {
	// Product handler
	productHandler := handlers.NewProductHandler(db)
	v1.GET("/products", productHandler.GetProducts)
	v1.GET("/products/:id", productHandler.GetProduct)
	v1.POST("/products", productHandler.CreateProduct)
	v1.DELETE("/products/:id", productHandler.DeleteProduct)

	// Sale handler
	saleHandler := handlers.NewSaleHandler(db)
	v1.GET("/sales", saleHandler.GetSales)
	v1.POST("/sales", saleHandler.CreateSale)
	v1.DELETE("/sales/:id", saleHandler.DeleteSale)
	v1.POST("/recipe-sales", saleHandler.CreateRecipeSale)

	// Stock movement handler
	stockMovementHandler := handlers.NewStockMovementHandler(db)
	v1.GET("/stock-movements", stockMovementHandler.GetStockMovements)

	// Reçete endpoint'leri
	recipeHandler := handlers.NewRecipeHandler(db)
	v1.GET("/recipes", recipeHandler.GetRecipes)
	v1.POST("/recipes", recipeHandler.CreateRecipe)
	v1.DELETE("/recipes/:id", recipeHandler.DeleteRecipe)
	v1.POST("/recipes/:id/produce", recipeHandler.ProduceFromRecipe)
}
