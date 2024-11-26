package models

import (
	"time"

	"gorm.io/gorm"
)

type BeehiivPostMetrics struct {
	ID              uint      `gorm:"primaryKey"`
	PostID          string    `json:"post_id" gorm:"uniqueIndex"`
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	PublishDate     time.Time `json:"publish_date"`
	
	// Email metrics
	EmailRecipients    int     `json:"email_recipients"`
	EmailDelivered     int     `json:"email_delivered"`
	EmailOpens         int     `json:"email_opens"`
	EmailUniqueOpens   int     `json:"email_unique_opens"`
	EmailClicks        int     `json:"email_clicks"`
	EmailUniqueClicks  int     `json:"email_unique_clicks"`
	EmailOpenRate      float64 `json:"email_open_rate"`
	EmailClickRate     float64 `json:"email_click_rate"`
	
	// Web metrics
	WebViews          int     `json:"web_views"`
	WebClicks         int     `json:"web_clicks"`
	WebClickRate      float64 `json:"web_click_rate"`
	
	// Combined metrics
	TotalEngagements  int     `json:"total_engagements"`
	
	CreatedAt        time.Time `json:"created_at"`
}

func MigrateBeehiivMetrics(db *gorm.DB) error {
	return db.AutoMigrate(&BeehiivPostMetrics{})
}