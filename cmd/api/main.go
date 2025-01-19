package main

import (
	"log"
	"stock-api/internal/api"
	"stock-api/internal/database"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("API başlatılıyor...")
	// Veritabanı bağlantısını başlat
	db, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Router ayarlanıyor...")
	// Router'ı ayarla
	r := gin.Default()

	// API versiyonu için group oluştur
	v1 := r.Group("/api/v1")

	// Router'ı ayarla
	api.SetupRouter(v1, db)

	log.Println("Sunucu 8080 portunda başlatılıyor...")
	// Sunucuyu başlat
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
