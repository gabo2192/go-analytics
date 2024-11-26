// repositories/beehiiv_metrics_repository.go
package repository

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"thedefiant.io/analytics/models"
	"thedefiant.io/analytics/services/beehiiv"
)

type BeehiivMetricsRepository struct {
	DB     *gorm.DB
	Client *beehiiv.Client
}

type BeehiivWeekMetrics struct {
	Date     time.Time `json:"date"`
	
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


func NewBeehiivMetricsRepository(db *gorm.DB, client *beehiiv.Client) *BeehiivMetricsRepository {
	return &BeehiivMetricsRepository{
		DB:     db,
		Client: client,
	}
}

func (r *BeehiivMetricsRepository) UpdatePostMetrics() error {
	var page = 1
	for {
		posts, err := r.Client.GetPosts(page)
		if err != nil {
			return fmt.Errorf("failed to get posts: %w", err)
		}
		
		if len(posts.Data) == 0 {
			break
		}

		for _, post := range posts.Data {
			// Calculate rates
			emailOpenRate := float64(0)
			if post.Stats.Email.Delivered > 0 {
				emailOpenRate = float64(post.Stats.Email.UniqueOpens) / float64(post.Stats.Email.Delivered) * 100
			}
			
			emailClickRate := float64(0)
			if post.Stats.Email.UniqueOpens > 0 {
				emailClickRate = float64(post.Stats.Email.UniqueClicks) / float64(post.Stats.Email.UniqueOpens) * 100
			}
			
			webClickRate := float64(0)
			if post.Stats.Web.Views > 0 {
				webClickRate = float64(post.Stats.Web.Clicks) / float64(post.Stats.Web.Views) * 100
			}

			metrics := models.BeehiivPostMetrics{
				PostID:            post.ID,
				Title:             post.Title,
				Slug:              post.Slug,
				PublishDate:       time.Unix(post.PublishDate, 0),
				
				EmailRecipients:   post.Stats.Email.Recipients,
				EmailDelivered:    post.Stats.Email.Delivered,
				EmailOpens:        post.Stats.Email.Opens,
				EmailUniqueOpens:  post.Stats.Email.UniqueOpens,
				EmailClicks:       post.Stats.Email.Clicks,
				EmailUniqueClicks: post.Stats.Email.UniqueClicks,
				EmailOpenRate:     emailOpenRate,
				EmailClickRate:    emailClickRate,
				
				WebViews:         post.Stats.Web.Views,
				WebClicks:        post.Stats.Web.Clicks,
				WebClickRate:     webClickRate,
				
				TotalEngagements: post.Stats.Email.UniqueOpens + post.Stats.Email.UniqueClicks + post.Stats.Web.Views + post.Stats.Web.Clicks,
				
				CreatedAt:        time.Now(),
			}

			if err := r.DB.Create(&metrics).Error; err != nil {
				log.Printf("Error saving metrics for post %s: %v", post.ID, err)
				continue
			}
		}
		
		page++
	}

	return nil
}

func (r *BeehiivMetricsRepository) GetPostMetrics(days int) ([]models.BeehiivPostMetrics, error) {
	var metrics []models.BeehiivPostMetrics
	err := r.DB.Where("created_at >= ?", time.Now().AddDate(0, 0, -days)).
		Order("publish_date desc").
		Find(&metrics).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post metrics: %w", err)
	}
	return metrics, nil
}

func (r *BeehiivMetricsRepository) GetWeekPostMetrics() (BeehiivWeekMetrics, error) {
	var metrics []models.BeehiivPostMetrics
		metricsAverage := BeehiivWeekMetrics{
			Date: time.Now(),
		}

	err := r.DB.Where("publish_date >= ?", time.Now().AddDate(0, 0, -7)).
		Order("publish_date desc").
		Find(&metrics).Error
	// get metrics and average them
	
	if err != nil {
		return metricsAverage, fmt.Errorf("failed to fetch post metrics: %w", err)
	}
	for _, metric := range metrics {
		metricsAverage.EmailRecipients += metric.EmailRecipients
		metricsAverage.EmailDelivered += metric.EmailDelivered
		metricsAverage.EmailOpens += metric.EmailOpens
		metricsAverage.EmailUniqueOpens += metric.EmailUniqueOpens
		metricsAverage.EmailClicks += metric.EmailClicks
		metricsAverage.EmailUniqueClicks += metric.EmailUniqueClicks
		metricsAverage.EmailOpenRate += metric.EmailOpenRate
		metricsAverage.EmailClickRate += metric.EmailClickRate
		metricsAverage.WebViews += metric.WebViews
		metricsAverage.WebClicks += metric.WebClicks
		metricsAverage.WebClickRate += metric.WebClickRate
		metricsAverage.TotalEngagements += metric.TotalEngagements
	}
	metricsAverage.EmailRecipients = metricsAverage.EmailRecipients / len(metrics)
	metricsAverage.EmailDelivered = metricsAverage.EmailDelivered / len(metrics)
	metricsAverage.EmailOpens = metricsAverage.EmailOpens / len(metrics)
	metricsAverage.EmailUniqueOpens = metricsAverage.EmailUniqueOpens / len(metrics)
	metricsAverage.EmailClicks = metricsAverage.EmailClicks / len(metrics)
	metricsAverage.EmailUniqueClicks = metricsAverage.EmailUniqueClicks / len(metrics)
	metricsAverage.EmailOpenRate = metricsAverage.EmailOpenRate / float64(len(metrics))
	metricsAverage.EmailClickRate = metricsAverage.EmailClickRate / float64(len(metrics))
	metricsAverage.WebViews = metricsAverage.WebViews / len(metrics)
	metricsAverage.WebClicks = metricsAverage.WebClicks / len(metrics)
	metricsAverage.WebClickRate = metricsAverage.WebClickRate / float64(len(metrics))
	metricsAverage.TotalEngagements = metricsAverage.TotalEngagements / len(metrics)

	return metricsAverage, nil
}
func (r *BeehiivMetricsRepository) GetWeekAlphaMetrics() (BeehiivWeekMetrics, error) {
	var metrics []models.BeehiivPostMetrics
		metricsAverage := BeehiivWeekMetrics{
			Date: time.Now(),
		}

	err := r.DB.Where("publish_date >= ?", time.Now().AddDate(0, 0, -8)).Where("title LIKE ?", "DeFi Alpha:%").
		Order("publish_date desc").
		Find(&metrics).Error
	// get metrics and average them
	
	if err != nil {
		return metricsAverage, fmt.Errorf("failed to fetch post metrics: %w", err)
	}
	if (len(metrics) == 0) {
		return metricsAverage, fmt.Errorf("no posts found: %w",err)
	}
	for _, metric := range metrics {
		
		metricsAverage.EmailRecipients += metric.EmailRecipients
		metricsAverage.EmailDelivered += metric.EmailDelivered
		metricsAverage.EmailOpens += metric.EmailOpens
		metricsAverage.EmailUniqueOpens += metric.EmailUniqueOpens
		metricsAverage.EmailClicks += metric.EmailClicks
		metricsAverage.EmailUniqueClicks += metric.EmailUniqueClicks
		metricsAverage.EmailOpenRate += metric.EmailOpenRate
		metricsAverage.EmailClickRate += metric.EmailClickRate
		metricsAverage.WebViews += metric.WebViews
		metricsAverage.WebClicks += metric.WebClicks
		metricsAverage.WebClickRate += metric.WebClickRate
		metricsAverage.TotalEngagements += metric.TotalEngagements
	}
	metricsAverage.EmailRecipients = metricsAverage.EmailRecipients / len(metrics)
	metricsAverage.EmailDelivered = metricsAverage.EmailDelivered / len(metrics)
	metricsAverage.EmailOpens = metricsAverage.EmailOpens / len(metrics)
	metricsAverage.EmailUniqueOpens = metricsAverage.EmailUniqueOpens / len(metrics)
	metricsAverage.EmailClicks = metricsAverage.EmailClicks / len(metrics)
	metricsAverage.EmailUniqueClicks = metricsAverage.EmailUniqueClicks / len(metrics)
	metricsAverage.EmailOpenRate = metricsAverage.EmailOpenRate / float64(len(metrics))
	metricsAverage.EmailClickRate = metricsAverage.EmailClickRate / float64(len(metrics))
	metricsAverage.WebViews = metricsAverage.WebViews / len(metrics)
	metricsAverage.WebClicks = metricsAverage.WebClicks / len(metrics)
	metricsAverage.WebClickRate = metricsAverage.WebClickRate / float64(len(metrics))
	metricsAverage.TotalEngagements = metricsAverage.TotalEngagements / len(metrics)

	return metricsAverage, nil
}

func (r *BeehiivMetricsRepository) GetMetricsByPostID(postID string) ([]models.BeehiivPostMetrics, error) {
	var metrics []models.BeehiivPostMetrics
	err := r.DB.Where("post_id = ?", postID).
		Order("created_at desc").
		Find(&metrics).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metrics for post: %w", err)
	}
	return metrics, nil
}

// Get top performing posts by email open rate
func (r *BeehiivMetricsRepository) GetTopPerformingPosts(limit int) ([]models.BeehiivPostMetrics, error) {
	var metrics []models.BeehiivPostMetrics
	err := r.DB.Where("email_recipients > ?", 100). // Minimum sample size
		Order("email_open_rate desc").
		Limit(limit).
		Find(&metrics).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top performing posts: %w", err)
	}
	return metrics, nil
}