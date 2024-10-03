package models

import (
	"time"

	"gorm.io/gorm"
)


type Author struct {
	ID *string `json:"id"`
    Name   *string     `json:"name" gorm:"column:name"`
	CreatedAt time.Time `json:"createdAt"`
}
type AuthorViews struct {
	ID uint `gorm:"primaryKey"`
	AuthorId *string `json:"authorId"`
	Author Author `gorm:"foreignKey:AuthorId"`
	Views int64 `json:"views"`
	CreatedAt time.Time `json:"createdAt"`
}

func MigrateAuthors(db *gorm.DB) error {
	err := db.AutoMigrate(&Author{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&AuthorViews{})
	return err
}