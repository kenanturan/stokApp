package models

import (
	"time"
)

type Recipe struct {
	ID             uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string       `json:"name" binding:"required"`
	Description    string       `json:"description"`
	OutputQuantity float64      `json:"outputQuantity" binding:"required,gt=0"`
	SuggestedPrice float64      `json:"suggestedPrice" binding:"omitempty,gte=0"`
	RecipeItems    []RecipeItem `gorm:"constraint:OnDelete:CASCADE;" json:"recipeItems"`
	CreatedAt      time.Time    `json:"createdAt"`
	UpdatedAt      time.Time    `json:"updatedAt"`
}

type RecipeItem struct {
	ID          uint     `gorm:"primaryKey;autoIncrement" json:"id"`
	RecipeID    uint     `json:"recipeId"`
	ProductID   uint     `json:"productId" binding:"required"`
	Recipe      *Recipe  `gorm:"foreignKey:RecipeID" json:"-"`
	Product     *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Quantity    float64  `json:"quantity" binding:"required,gt=0"`
	Description string   `json:"description"`
}
