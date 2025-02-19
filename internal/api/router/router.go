// Router dosyasını görebilir miyim?
// Response'u wrap eden bir middleware var mı kontrol etmemiz gerekiyor

package router

import (
	"net/http"
	"stock-api/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRouter(productHandler *handlers.ProductHandler,
	saleHandler *handlers.SaleHandler,
	recipeHandler *handlers.RecipeHandler) *gin.Engine {

	router := gin.Default()

	// Middleware'ler burada tanımlanır
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(ResponseMiddleware()) // <-- Bu tür bir middleware response'u wrap ediyor olabilir

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Product routes
		v1.GET("/products", productHandler.GetProducts)
		v1.POST("/products", productHandler.CreateProduct)
		// ... diğer product routes

		// Sale routes
		v1.POST("/sales", saleHandler.CreateSale)
		v1.GET("/sales", saleHandler.GetSales)
		// ... diğer sale routes

		// Recipe routes
		v1.POST("/recipes", recipeHandler.CreateRecipe)
		// ... diğer recipe routes
	}

	return router
}

// Bu tür bir middleware response'u wrap ediyor olabilir
func ResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Response'u wrap ediyor olabilir
		if data, exists := c.Get("response"); exists {
			c.JSON(http.StatusOK, gin.H{"data": data})
		}
	}
}
