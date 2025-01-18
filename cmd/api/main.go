package main

import (
	"log"
	"stock-api/internal/api"
	"stock-api/internal/database"
)

func main() {
	log.Println("API başlatılıyor...")
	// Veritabanı bağlantısını başlat
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Veritabanı başlatılamadı: %v", err)
	}

	log.Println("Router ayarlanıyor...")
	// Router'ı oluştur
	router := api.SetupRouter(db)

	// Sunucuyu başlat
	log.Println("Sunucu 8080 portunda başlatılıyor...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Sunucu başlatılamadı: %v", err)
	}
}
