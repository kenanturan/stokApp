package database

import (
	"stock-api/internal/models"

	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	log.Println("Veritabanı başlatılıyor...")

	// SQLite bağlantısı
	db, err := gorm.Open(sqlite.Open("stock.db"), &gorm.Config{})
	if err != nil {
		log.Printf("Veritabanı bağlantı hatası: %v", err)
		return nil, err
	}

	// Auto Migration
	err = db.AutoMigrate(
		&models.Product{},
		&models.Sale{},
		&models.StockMovement{},
		&models.StockUsage{},
		&models.Recipe{},
		&models.RecipeItem{},
	)
	if err != nil {
		log.Printf("Migration hatası: %v", err)
		return nil, err
	}

	return db, nil
}
