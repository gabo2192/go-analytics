package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"thedefiant.io/analytics/models"
	"thedefiant.io/analytics/services/beehiiv"
)

type BeehiivMetricsRepository struct {
	DB     *sql.DB
	Client *beehiiv.Client
}

type BeehiivWeekMetrics struct {
	Date               time.Time `json:"date"`
	EmailRecipients    int       `json:"email_recipients"`
	EmailDelivered     int       `json:"email_delivered"`
	EmailOpens         int       `json:"email_opens"`
	EmailUniqueOpens   int       `json:"email_unique_opens"`
	EmailClicks        int       `json:"email_clicks"`
	EmailUniqueClicks  int       `json:"email_unique_clicks"`
	EmailOpenRate      float64   `json:"email_open_rate"`
	EmailClickRate     float64   `json:"email_click_rate"`
	WebViews           int       `json:"web_views"`
	WebClicks          int       `json:"web_clicks"`
	WebClickRate       float64   `json:"web_click_rate"`
	TotalEngagements   int       `json:"total_engagements"`
	CreatedAt          time.Time `json:"created_at"`
}

func NewBeehiivMetricsRepository(db *sql.DB, client *beehiiv.Client) *BeehiivMetricsRepository {
	return &BeehiivMetricsRepository{DB: db, Client: client}
}

func (r *BeehiivMetricsRepository) UpdatePostMetrics(until *time.Time) error {
	now := time.Now()

	// Get first page for total_pages
	res, err := r.Client.GetPosts(1)
	if err != nil {
		return fmt.Errorf("failed to fetch page 1: %w", err)
	}
	totalPages := res.TotalPages
	log.Printf("Total pages: %d", totalPages)
	totalSynced := 0
	for page := 1; page <= totalPages; page++ {
		res, err := r.Client.GetPosts(page)
		log.Printf("Fetching page %d", page)
		log.Printf("Page %d: %d posts", page, len(res.Data))
		if err != nil {
			log.Printf("❌ Failed to fetch page %d: %v", page, err)
			continue
		}
		if len(res.Data) == 0 {
			break
		}

		// Check if all posts are older than "until"
		allOlder := true
		for _, post := range res.Data {
			publishedAt := time.Unix(post.PublishDate, 0)
		
			// Skip if no publish date
			if post.PublishDate == 0 {
				allOlder = false
				continue
			}
		
			if until == nil || publishedAt.After(*until) {
				allOlder = false
				break
			}
		}
		
		if allOlder {
			log.Printf("🛑 All posts on page %d are before cutoff %s — stopping", page, until.Format("2006-01-02"))
			break
		}

		var values []interface{}
		var placeholders []string
		i := 1
		skipped := 0

		for _, post := range res.Data {
			publishedAt := time.Unix(post.PublishDate, 0)
			log.Printf("Post %s: %s (%s)", post.ID, post.Title, publishedAt.Format("2006-01-02"))
			// Skip if older than "until"
			if until != nil && publishedAt.Before(*until) {
				skipped++
				continue
			}

			// Compute metrics
			emailOpenRate := 0.0
			if post.Stats.Email.Delivered > 0 {
				emailOpenRate = float64(post.Stats.Email.UniqueOpens) / float64(post.Stats.Email.Delivered) * 100
			}

			emailClickRate := 0.0
			if post.Stats.Email.UniqueOpens > 0 {
				emailClickRate = float64(post.Stats.Email.UniqueClicks) / float64(post.Stats.Email.UniqueOpens) * 100
			}

			webClickRate := 0.0
			if post.Stats.Web.Views > 0 {
				webClickRate = float64(post.Stats.Web.Clicks) / float64(post.Stats.Web.Views) * 100
			}

			total := post.Stats.Email.UniqueOpens + post.Stats.Email.UniqueClicks + post.Stats.Web.Views + post.Stats.Web.Clicks

			placeholders = append(placeholders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				i, i+1, i+2, i+3, i+4, i+5, i+6, i+7, i+8, i+9, i+10, i+11, i+12, i+13, i+14, i+15, i+16, i+17, i+18))
			values = append(values,
				post.ID, post.Title, post.Slug, publishedAt,
				post.Stats.Email.Recipients, post.Stats.Email.Delivered, post.Stats.Email.Opens, post.Stats.Email.UniqueOpens,
				post.Stats.Email.Clicks, post.Stats.Email.UniqueClicks, emailOpenRate, emailClickRate,
				post.Stats.Web.Views, post.Stats.Web.Clicks, webClickRate,
				total,
				now, now, now,
			)
			i += 19
			totalSynced++
		}

		if len(values) == 0 {
			log.Printf("ℹ️ No valid posts to sync on page %d (skipped %d)", page, skipped)
			continue
		}

		query := `
		INSERT INTO beehiiv_post_metrics (
			post_id, title, slug, publish_date,
			email_recipients, email_delivered, email_opens, email_unique_opens,
			email_clicks, email_unique_clicks, email_open_rate, email_click_rate,
			web_views, web_clicks, web_click_rate,
			total_engagements,
			created_at, inserted_at, updated_at
		) VALUES ` + strings.Join(placeholders, ",") + `
		ON CONFLICT (post_id) DO UPDATE SET
			title = EXCLUDED.title,
			slug = EXCLUDED.slug,
			publish_date = EXCLUDED.publish_date,
			email_recipients = EXCLUDED.email_recipients,
			email_delivered = EXCLUDED.email_delivered,
			email_opens = EXCLUDED.email_opens,
			email_unique_opens = EXCLUDED.email_unique_opens,
			email_clicks = EXCLUDED.email_clicks,
			email_unique_clicks = EXCLUDED.email_unique_clicks,
			email_open_rate = EXCLUDED.email_open_rate,
			email_click_rate = EXCLUDED.email_click_rate,
			web_views = EXCLUDED.web_views,
			web_clicks = EXCLUDED.web_clicks,
			web_click_rate = EXCLUDED.web_click_rate,
			total_engagements = EXCLUDED.total_engagements,
			updated_at = EXCLUDED.updated_at
		`

		if _, err := r.DB.Exec(query, values...); err != nil {
			log.Printf("❌ Error inserting page %d: %v", page, err)
		} else {
			log.Printf("✅ Page %d: Synced %d posts (skipped %d)", page, len(values)/19, skipped)
		}
	}

	log.Printf("✅ Finished syncing %d Beehiiv posts", totalSynced)
	return nil
}

func (r *BeehiivMetricsRepository) GetPostMetrics(days int) ([]models.BeehiivPostMetrics, error) {
	rows, err := r.DB.Query(`
		SELECT * FROM beehiiv_post_metrics
		WHERE created_at >= $1
		ORDER BY publish_date DESC
	`, time.Now().AddDate(0, 0, -days))
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var metrics []models.BeehiivPostMetrics
	for rows.Next() {
		var m models.BeehiivPostMetrics
		err := rows.Scan(
			&m.PostID, &m.Title, &m.Slug, &m.PublishDate,
			&m.EmailRecipients, &m.EmailDelivered, &m.EmailOpens, &m.EmailUniqueOpens,
			&m.EmailClicks, &m.EmailUniqueClicks, &m.EmailOpenRate, &m.EmailClickRate,
			&m.WebViews, &m.WebClicks, &m.WebClickRate,
			&m.TotalEngagements, &m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func (r *BeehiivMetricsRepository) GetWeekPostMetrics() (BeehiivWeekMetrics, error) {
	rows, err := r.DB.Query(`
		SELECT email_recipients, email_delivered, email_opens, email_unique_opens,
			email_clicks, email_unique_clicks, email_open_rate, email_click_rate,
			web_views, web_clicks, web_click_rate, total_engagements
		FROM beehiiv_post_metrics
		WHERE publish_date >= $1`, time.Now().AddDate(0, 0, -7))
	if err != nil {
		return BeehiivWeekMetrics{}, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var avg BeehiivWeekMetrics
	count := 0

	for rows.Next() {
		var m BeehiivWeekMetrics
		err := rows.Scan(
			&m.EmailRecipients, &m.EmailDelivered, &m.EmailOpens, &m.EmailUniqueOpens,
			&m.EmailClicks, &m.EmailUniqueClicks, &m.EmailOpenRate, &m.EmailClickRate,
			&m.WebViews, &m.WebClicks, &m.WebClickRate, &m.TotalEngagements,
		)
		if err != nil {
			return avg, fmt.Errorf("scan failed: %w", err)
		}
		avg.EmailRecipients += m.EmailRecipients
		avg.EmailDelivered += m.EmailDelivered
		avg.EmailOpens += m.EmailOpens
		avg.EmailUniqueOpens += m.EmailUniqueOpens
		avg.EmailClicks += m.EmailClicks
		avg.EmailUniqueClicks += m.EmailUniqueClicks
		avg.EmailOpenRate += m.EmailOpenRate
		avg.EmailClickRate += m.EmailClickRate
		avg.WebViews += m.WebViews
		avg.WebClicks += m.WebClicks
		avg.WebClickRate += m.WebClickRate
		avg.TotalEngagements += m.TotalEngagements
		count++
	}

	if count == 0 {
		return avg, nil
	}

	avg.Date = time.Now()
	avg.EmailRecipients /= count
	avg.EmailDelivered /= count
	avg.EmailOpens /= count
	avg.EmailUniqueOpens /= count
	avg.EmailClicks /= count
	avg.EmailUniqueClicks /= count
	avg.EmailOpenRate /= float64(count)
	avg.EmailClickRate /= float64(count)
	avg.WebViews /= count
	avg.WebClicks /= count
	avg.WebClickRate /= float64(count)
	avg.TotalEngagements /= count
	return avg, nil
}

func (r *BeehiivMetricsRepository) GetWeekAlphaMetrics() (BeehiivWeekMetrics, error) {
	rows, err := r.DB.Query(`
		SELECT email_recipients, email_delivered, email_opens, email_unique_opens,
			email_clicks, email_unique_clicks, email_open_rate, email_click_rate,
			web_views, web_clicks, web_click_rate, total_engagements
		FROM beehiiv_post_metrics
		WHERE title LIKE 'DeFi Alpha:%'
		ORDER BY publish_date DESC LIMIT 1`)
	if err != nil {
		return BeehiivWeekMetrics{}, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var m BeehiivWeekMetrics
	if rows.Next() {
		err := rows.Scan(
			&m.EmailRecipients, &m.EmailDelivered, &m.EmailOpens, &m.EmailUniqueOpens,
			&m.EmailClicks, &m.EmailUniqueClicks, &m.EmailOpenRate, &m.EmailClickRate,
			&m.WebViews, &m.WebClicks, &m.WebClickRate, &m.TotalEngagements,
		)
		if err != nil {
			return m, fmt.Errorf("scan failed: %w", err)
		}
		m.Date = time.Now()
		return m, nil
	}
	return m, fmt.Errorf("no DeFi Alpha metrics found")
}

func (r *BeehiivMetricsRepository) GetMetricsByPostID(postID string) ([]models.BeehiivPostMetrics, error) {
	rows, err := r.DB.Query(`
		SELECT * FROM beehiiv_post_metrics
		WHERE post_id = $1
		ORDER BY created_at DESC`, postID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var metrics []models.BeehiivPostMetrics
	for rows.Next() {
		var m models.BeehiivPostMetrics
		err := rows.Scan(
			&m.PostID, &m.Title, &m.Slug, &m.PublishDate,
			&m.EmailRecipients, &m.EmailDelivered, &m.EmailOpens, &m.EmailUniqueOpens,
			&m.EmailClicks, &m.EmailUniqueClicks, &m.EmailOpenRate, &m.EmailClickRate,
			&m.WebViews, &m.WebClicks, &m.WebClickRate,
			&m.TotalEngagements, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func (r *BeehiivMetricsRepository) GetTopPerformingPosts(limit int) ([]models.BeehiivPostMetrics, error) {
	rows, err := r.DB.Query(`
		SELECT * FROM beehiiv_post_metrics
		WHERE email_recipients > 100
		ORDER BY email_open_rate DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.BeehiivPostMetrics
	for rows.Next() {
		var m models.BeehiivPostMetrics
		err := rows.Scan(
			&m.PostID, &m.Title, &m.Slug, &m.PublishDate,
			&m.EmailRecipients, &m.EmailDelivered, &m.EmailOpens, &m.EmailUniqueOpens,
			&m.EmailClicks, &m.EmailUniqueClicks, &m.EmailOpenRate, &m.EmailClickRate,
			&m.WebViews, &m.WebClicks, &m.WebClickRate,
			&m.TotalEngagements, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func (r *BeehiivMetricsRepository) GetFreePostsMetrics() ([]models.BeehiivPostMetrics, error) {
	rows, err := r.DB.Query(`
		SELECT * FROM beehiiv_post_metrics
		WHERE title NOT LIKE 'DeFi Alpha:%'
		ORDER BY publish_date DESC
		LIMIT 6`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.BeehiivPostMetrics
	for rows.Next() {
		var m models.BeehiivPostMetrics
		err := rows.Scan(
			&m.PostID, &m.Title, &m.Slug, &m.PublishDate,
			&m.EmailRecipients, &m.EmailDelivered, &m.EmailOpens, &m.EmailUniqueOpens,
			&m.EmailClicks, &m.EmailUniqueClicks, &m.EmailOpenRate, &m.EmailClickRate,
			&m.WebViews, &m.WebClicks, &m.WebClickRate,
			&m.TotalEngagements, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func (r *BeehiivMetricsRepository) GetFreePostsMonthlyMetrics() ([]models.BeehiivPostMetrics, error) {
	lastMonth := time.Now().AddDate(0, -1, 0)
	rows, err := r.DB.Query(`
		SELECT * FROM beehiiv_post_metrics
		WHERE title NOT LIKE 'DeFi Alpha:%' AND publish_date >= $1
		ORDER BY publish_date DESC`, lastMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.BeehiivPostMetrics
	for rows.Next() {
		var m models.BeehiivPostMetrics
		err := rows.Scan(
			&m.PostID, &m.Title, &m.Slug, &m.PublishDate,
			&m.EmailRecipients, &m.EmailDelivered, &m.EmailOpens, &m.EmailUniqueOpens,
			&m.EmailClicks, &m.EmailUniqueClicks, &m.EmailOpenRate, &m.EmailClickRate,
			&m.WebViews, &m.WebClicks, &m.WebClickRate,
			&m.TotalEngagements, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func (r *BeehiivMetricsRepository) GetAlphaPostsMonthlyMetrics() ([]models.BeehiivPostMetrics, error) {
	lastMonth := time.Now().AddDate(0, -1, 0)
	rows, err := r.DB.Query(`
		SELECT * FROM beehiiv_post_metrics
		WHERE title LIKE 'DeFi Alpha:%' AND publish_date >= $1
		ORDER BY publish_date DESC`, lastMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.BeehiivPostMetrics
	for rows.Next() {
		var m models.BeehiivPostMetrics
		err := rows.Scan(
			&m.PostID, &m.Title, &m.Slug, &m.PublishDate,
			&m.EmailRecipients, &m.EmailDelivered, &m.EmailOpens, &m.EmailUniqueOpens,
			&m.EmailClicks, &m.EmailUniqueClicks, &m.EmailOpenRate, &m.EmailClickRate,
			&m.WebViews, &m.WebClicks, &m.WebClickRate,
			&m.TotalEngagements, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}
