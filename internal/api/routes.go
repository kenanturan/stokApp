package api

import (
	"stock-api/internal/api/handlers"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB) *gin.Engine {
	router := gin.Default()

	// CORS ayarları
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "X-CSRF-Token", "Accept", "X-Requested-With"}
	config.AllowCredentials = true
	config.ExposeHeaders = []string{"Content-Length"}
	config.MaxAge = 12 * time.Hour
	router.Use(cors.New(config))

	productHandler := handlers.NewProductHandler(db)
	saleHandler := handlers.NewSaleHandler(db)
	stockMovementHandler := handlers.NewStockMovementHandler(db)

	v1 := router.Group("/api/v1")
	{
		products := v1.Group("/products")
		{
			products.GET("", productHandler.GetProducts)
			products.POST("", productHandler.CreateProduct)
			products.DELETE("/:id", productHandler.DeleteProduct)
		}

		sales := v1.Group("/sales")
		{
			sales.GET("", saleHandler.GetSales)
			sales.POST("", saleHandler.CreateSale)
		}

		stockMovements := v1.Group("/stock-movements")
		{
			stockMovements.GET("", stockMovementHandler.GetStockMovements)
		}
	}

	return router
}
