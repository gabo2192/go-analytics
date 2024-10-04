package models

import (
	"time"

	"gorm.io/gorm"
)
type InputAuthor struct {
	ID         string `json:"_id"`
	Name       string `json:"name"`
	PostsSlugs []struct {
		Slug         string `json:"slug"`
		MainCategory string `json:"mainCategory"`
		SubCategory  string `json:"subCategory"`
	} `json:"postsSlugs"`
}

type OutputAuthor struct {
	ID   string   `json:"id"`
	Slugs []string `json:"slug"`
}

type AnalyticsReponse struct{
	Views int64 `json:"views"`
	ID string `json:"id"`
}

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