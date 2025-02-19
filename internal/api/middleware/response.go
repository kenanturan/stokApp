package middleware

import (
	"net/http"

	"log"

	"github.com/gin-gonic/gin"
)

// ResponseMiddleware - response'ları wrap eden middleware
func ResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		log.Printf("Middleware - Status: %d", c.Writer.Status())
		log.Printf("Middleware - Written: %v", c.Writer.Written())

		// Eğer response zaten yazılmışsa, işlem yapma
		if c.Writer.Written() {
			log.Printf("Middleware - Response zaten yazılmış")
			return
		}

		// Response'u al
		if data, exists := c.Get("response"); exists {
			log.Printf("Middleware - Response data: %+v", data)

			status := http.StatusOK
			if code, exists := c.Get("status"); exists {
				if statusCode, ok := code.(int); ok {
					status = statusCode
				}
			}
			log.Printf("Middleware - Status code: %d", status)

			c.JSON(status, data)
		} else {
			log.Printf("Middleware - Response data bulunamadı")
		}
	}
}
