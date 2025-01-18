package database

import (
	"stock-api/internal/models"

	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	log.Println("Veritabanı başlatılıyor...")
	db, err := gorm.Open(sqlite.Open("stock.db"), &gorm.Config{})
	if err != nil {
		log.Printf("Veritabanı bağlantı hatası: %v", err)
		return nil, err
	}

	// Debug modu aktif et
	db = db.Debug()

	log.Println("Tablolar oluşturuluyor...")
	// Tabloları oluştur
	err = db.AutoMigrate(
		&models.Product{},
		&models.Sale{},
		&models.StockMovement{},
		&models.StockUsage{},
	)
	if err != nil {
		log.Printf("AutoMigrate hatası: %v", err)
		return nil, err
	}
	log.Println("Tablolar başarıyla oluşturuldu")

	return db, nil
}
