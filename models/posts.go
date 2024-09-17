package models

import "gorm.io/gorm"


type Posts struct {
	ID *string `json:"id"`
	Title *string `json:"title"`
	Slug *string `json:"slug"`
	AuthorId *string `json:"authorId"`
}

func MigratePosts(db *gorm.DB) error {
	err := db.AutoMigrate(&Posts{})
	return err
}