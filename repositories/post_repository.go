package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"thedefiant.io/analytics/models"
	"thedefiant.io/analytics/services/analytics"
	"thedefiant.io/analytics/services/sanity"
	"thedefiant.io/analytics/utils"
)

type PostRepository struct {
	DB        *sql.DB
	Sanity    *sanity.Client
	Analytics *analytics.Client
}

func NewPostRepository(db *sql.DB, sanityClient *sanity.Client, analyticsClient *analytics.Client) *PostRepository {
	return &PostRepository{
		DB:        db,
		Sanity:    sanityClient,
		Analytics: analyticsClient,
	}
}

func (r *PostRepository) CreatePost() ([]models.Post, error) {
	var posts []models.Post
	twoDaysAgo := utils.GetDateNDaysAgo(2)

	query := `*[_type in ['blog', 'sponsor'] && defined(mainCategory) && defined(subCategory) && !(_id in path('drafts.**')) && publishedAt > $date ]| order(dateTime(publishedAt) desc){'id':_id, title,'slug': slug.current,'authorId': author._ref, 'mainCategory': mainCategory->slug.current, 'subCategory': subCategory->slug.current,publishedAt, 'createdAt': _createdAt}`
	params := map[string]interface{}{
		"date": twoDaysAgo,
	}

	result, err := r.Sanity.Query(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query Sanity: %w", err)
	}

	if err := result.Unmarshal(&posts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Sanity result: %w", err)
	}

	if len(posts) == 0 {
		log.Println("No posts returned from Sanity")
		return posts, nil
	}

	// Step 1: Fetch all known author IDs
	existingAuthors := make(map[string]bool)
	rows, err := r.DB.Query("SELECT id FROM authors")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch authors: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			existingAuthors[id] = true
		}
	}

	// Step 2: Filter posts to only those with valid authors
	var values []interface{}
	var placeholders []string
	i := 1

	for _, p := range posts {
		if p.AuthorId == nil || !existingAuthors[*p.AuthorId] {
			log.Printf("⚠️ Skipping post %s due to missing author_id: %v", *p.ID, p.AuthorId)
			continue
		}
		placeholders = append(placeholders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d)", i, i+1, i+2, i+3, i+4, i+5, i+6))
		values = append(values,
			*p.ID, *p.Title, *p.Slug, *p.AuthorId,
			*p.MainCategory, *p.SubCategory, p.PublishedAt,
		)
		i += 7
	}


	querySQL := `
	INSERT INTO posts (id, title, slug, author_id, main_category, sub_category, published_at)
	VALUES ` + strings.Join(placeholders, ",") + `
	ON CONFLICT (id) DO NOTHING
	`

	if _, err := r.DB.Exec(querySQL, values...); err != nil {
		return nil, fmt.Errorf("failed to insert posts: %w", err)
	}

	log.Printf("✅ Inserted %d posts (duplicates skipped)", len(values)/7)
	return posts, nil
}

func (r *PostRepository) GetPostsFromDatabase() ([]models.Post, error) {
	query := `SELECT id, title, slug, author_id, main_category, sub_category, published_at, created_at FROM posts`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posts: %w", err)
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.AuthorId, &p.MainCategory, &p.SubCategory, &p.PublishedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func (r *PostRepository) GetAnalyticsData(dateRange string, posts []models.Post) ([]models.Post, error) {
	slugs := make([]string, len(posts))
	for i, post := range posts {
		slugs[i] = "/" + *post.MainCategory + "/" + *post.SubCategory + "/" + *post.Slug
	}

	pageViews, err := r.Analytics.GetPageViews(dateRange, slugs)
	if err != nil {
		return nil, fmt.Errorf("failed to get page views: %w", err)
	}

	field := utils.GetDBFieldName(dateRange)
	today := time.Now()

	for _, post := range posts {
		slug := "/" + *post.MainCategory + "/" + *post.SubCategory + "/" + *post.Slug
		views := pageViews[slug]

		query := fmt.Sprintf(`UPDATE posts SET %s = $1, updated_at = $2 WHERE id = $3`, field)
		if _, err := r.DB.Exec(query, views, today, *post.ID); err != nil {
			log.Printf("Error updating %s for post %s: %v", field, *post.ID, err)
		}
	}

	return posts, nil
}

func (r *PostRepository) UpdateYesterdayViews() ([]models.Post, error) {
	yesterday := utils.GetDateNDaysAgo(2)
	return r.fetchAndUpdate("yesterday", "DATE(published_at) >= $1", yesterday)
}

func (r *PostRepository) UpdateLastSevenDaysViews() ([]models.Post, error) {
	return r.updateViewsForDateRange("last7days")
}

func (r *PostRepository) UpdateLast14DaysViews() ([]models.Post, error) {
	return r.updateViewsForDateRange("last14days")
}

func (r *PostRepository) UpdateLast30DaysViews() ([]models.Post, error) {
	return r.updateViewsForDateRange("last30days")
}

func (r *PostRepository) UpdateLast90DaysViews() ([]models.Post, error) {
	return r.updateViewsForDateRange("last90days")
}

func (r *PostRepository) UpdateLast180DaysViews() ([]models.Post, error) {
	return r.updateViewsForDateRange("last180days")
}

func (r *PostRepository) UpdateLast365DaysViews() ([]models.Post, error) {
	return r.updateViewsForDateRange("last365days")
}

func (r *PostRepository) updateViewsForDateRange(rangeType string) ([]models.Post, error) {
	nDaysAgo := utils.GetDateNDaysAgo(utils.GetDaysFromRangeType(rangeType))
	return r.fetchAndUpdate(rangeType, "DATE(published_at) = $1", nDaysAgo)
}

func (r *PostRepository) fetchAndUpdate(rangeType, condition string, arg interface{}) ([]models.Post, error) {
	query := fmt.Sprintf(`
		SELECT id, title, slug, author_id, main_category, sub_category, published_at, created_at
		FROM posts
		WHERE %s AND slug IS NOT NULL`, condition)

	rows, err := r.DB.Query(query, arg)
	if err != nil {
		return nil, fmt.Errorf("query error for %s: %w", rangeType, err)
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.AuthorId, &p.MainCategory, &p.SubCategory, &p.PublishedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	if len(posts) == 0 {
		log.Printf("No posts found for %s", rangeType)
		return nil, nil
	}

	return r.GetAnalyticsData(rangeType, posts)
}
