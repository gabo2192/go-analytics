package models

import (
	"time"

	"gorm.io/gorm"
)


type Post struct {
	ID *string `json:"id"`
	Title *string `json:"title"`
	Slug *string `json:"slug"`
	AuthorId *string `json:"authorId"`
	MainCategory *string `json:"mainCategory"`
	SubCategory *string `json:"subCategory"`
	PublishedAt time.Time `json:"publishedAt"`
	YesterdayViews     int64     `json:"yesterdayViews" gorm:"column:yesterday_views"`
    LastSevenDaysViews int64     `json:"lastSevenDaysViews" gorm:"column:last_seven_days_views"`
    Last14DaysViews    int64     `json:"last14DaysViews" gorm:"column:last_14_days_views"`
    Last30DaysViews    int64     `json:"last30DaysViews" gorm:"column:last_30_days_views"`
    Last90DaysViews    int64     `json:"last90DaysViews" gorm:"column:last_90_days_views"`
    Last180DaysViews   int64     `json:"last180DaysViews" gorm:"column:last_180_days_views"`
    Last365DaysViews   int64     `json:"last365DaysViews" gorm:"column:last_365_days_views"`
	CreatedAt time.Time `json:"createdAt"`
}

func MigratePosts(db *gorm.DB) error {
	err := db.AutoMigrate(&Post{})
	return err
}