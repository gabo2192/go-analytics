package repository

import (
	"fmt"
	"log"
	"strings"
	"time"

	"database/sql"

	"thedefiant.io/analytics/models"
	"thedefiant.io/analytics/services/analytics"
	"thedefiant.io/analytics/services/sanity"
	"thedefiant.io/analytics/utils"
)

type AuthorRepository struct {
	DB     *sql.DB
	Sanity *sanity.Client
	Analytics *analytics.Client
}

type AuthorWithViews struct {
	models.Author
	Views int64
	CreatedAt time.Time
}

type AuthorsResponse struct {
	models.Author
	Views []AuthorView `json:"views"`
}

// AuthorView represents a single view entry for an author
type AuthorView struct {
	View      int64     `json:"view"`
	CreatedAt time.Time `json:"createdAt"`
}


func NewAuthorRepository(db *sql.DB, sanityClient *sanity.Client, analyticsClient *analytics.Client) *AuthorRepository {
	return &AuthorRepository{
		DB:     db,
		Sanity: sanityClient,
		Analytics: analyticsClient,
	}
}

func (r *AuthorRepository) CreateAuthor() ([]models.Author, error) {
	thirtyDaysAgo := utils.GetDateNDaysAgo(30)
	query := `*[_type == 'author' && _createdAt > $date]{'id': _id, name}`
	params := map[string]interface{}{
		"date": thirtyDaysAgo,
	}
	result, err := r.Sanity.Query(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query Sanity: %w", err)
	}

	var authors []models.Author
	if err := result.Unmarshal(&authors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Sanity result: %w", err)
	}

	if len(authors) == 0 {
		log.Println("No new authors found")
		return authors, nil
	}

	// Build SQL INSERT
	var values []interface{}
	var placeholders []string
	i := 1

	now := time.Now()
	for _, author := range authors {
		if author.ID == nil || *author.Name == "" {
			continue
		}
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d)", i, i+1, i+2))
		values = append(values, *author.ID, author.Name, now)
		i += 3
	}

	querySQL := `
		INSERT INTO authors (id, name, created_at)
		VALUES ` + strings.Join(placeholders, ",") + `
		ON CONFLICT (id) DO NOTHING
	`

	_, err = r.DB.Exec(querySQL, values...)
	if err != nil {
		return nil, fmt.Errorf("failed to insert authors: %w", err)
	}

	log.Printf("✅ Inserted %d authors (duplicates skipped)", len(values)/3)
	return authors, nil
}

func (r *AuthorRepository) GetAuthorsFromDatabase() ([]AuthorsResponse, error) {
	query := `
	SELECT a.id, a.name, a.created_at, av.views, av.created_at
	FROM authors a
	LEFT JOIN author_views av ON a.id = av.author_id
	ORDER BY a.id, av.created_at DESC
	`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch authors with views: %w", err)
	}
	defer rows.Close()

	var authors []AuthorsResponse
	var currentID *string
	var currentIndex = -1

	for rows.Next() {
		var id, name string
		var authorCreated, viewCreated time.Time
		var views sql.NullInt64

		err := rows.Scan(&id, &name, &authorCreated, &views, &viewCreated)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if currentID == nil || *currentID != id {
			currentID = &id
			currentIndex++
			authors = append(authors, AuthorsResponse{
				Author: models.Author{
					ID:        &id,
					Name:      &name,
					CreatedAt: authorCreated,
				},
				Views: []AuthorView{},
			})
		}

		if views.Valid {
			authors[currentIndex].Views = append(authors[currentIndex].Views, AuthorView{
				View:      views.Int64,
				CreatedAt: viewCreated,
			})
		}
	}

	return authors, nil
}


func (r *AuthorRepository) GetAuthorByID(id string) (*models.Author, error) {
	query := `SELECT id, name, created_at FROM authors WHERE id = $1`

	var author models.Author
	err := r.DB.QueryRow(query, id).Scan(&author.ID, &author.Name, &author.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch author by ID: %w", err)
	}
	return &author, nil
}

func (r *AuthorRepository) GetSanityPostsByAuthorCurrentMonth() ([]models.OutputAuthor, error) {
	thirtyDaysAgo := utils.GetDateNDaysAgo(30)
	today := time.Now().Format("2006-01-02")
	query := `*[_type == 'author' && count(
		*[_type in ['blog', 'sponsor'] 
		&& references(^._id) 
		&& publishedAt > $from
		&& publishedAt < $to 
		&& defined(mainCategory) 
		&& defined(subCategory)
		]) > 0
	]{
		_id,
		name,
		'postsSlugs': *[_type in ['blog', 'sponsor'] 
		&& references(^._id) 
		&& publishedAt > $from
		&& publishedAt < $to 
		&& defined(mainCategory) 
		&& defined(subCategory)
		]{
			'slug': slug.current,
			'mainCategory': mainCategory->slug.current,
			'subCategory': subCategory->slug.current
		}
	}`
	
	params := map[string]interface{}{
		"from": thirtyDaysAgo,
		"to":   today,
	}

	result, err := r.Sanity.Query(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query Sanity: %w", err)
	}

	var inputAuthors []models.InputAuthor
	if err := result.Unmarshal(&inputAuthors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Sanity result: %w", err)
	}

	outputAuthors := make([]models.OutputAuthor, len(inputAuthors))
	for i, author := range inputAuthors {
		slugs := make([]string, len(author.PostsSlugs))
		for j, post := range author.PostsSlugs {
			slugs[j] = "/" + post.MainCategory + "/" + post.SubCategory + "/" + post.Slug 
		}
		outputAuthors[i] = models.OutputAuthor{
			ID:   author.ID,
			Slugs: slugs,
		}
	}

	return outputAuthors, nil
}

func (r *AuthorRepository) UpdateAnalyticsViews() ([]models.AuthorViews, error) {
	authorsAndSlugs, err := r.GetSanityPostsByAuthorCurrentMonth()
	if err != nil {
		return nil, fmt.Errorf("failed to get authors and slugs: %w", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Load known authors
	rows, err := r.DB.Query(`SELECT id FROM authors`)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch author ids: %w", err)
	}
	defer rows.Close()

	known := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		known[id] = struct{}{}
	}

	var values []interface{}
	var placeholders []string
	i := 1
	for _, a := range authorsAndSlugs {
		if _, ok := known[a.ID]; !ok {
			log.Printf("⚠️ Skipping author view insert for missing author_id: %s", a.ID)
			continue
		}

		views := int64(0)
		pageViews, err := r.Analytics.GetPageViews("last30days", a.Slugs)
		if err != nil {
			return nil, err
		}
		for _, slug := range a.Slugs {
			views += pageViews[slug]
		}

		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d)", i, i+1, i+2))
		values = append(values, a.ID, views, today)
		i += 3
	}

	if len(values) == 0 {
		log.Println("ℹ️ No valid AuthorViews to insert")
		return nil, nil
	}

	query := `
		INSERT INTO author_views (author_id, views, created_at)
		VALUES ` + strings.Join(placeholders, ",") + `
		ON CONFLICT (author_id, created_at) DO NOTHING`

	if _, err := r.DB.Exec(query, values...); err != nil {
		return nil, fmt.Errorf("insert author views failed: %w", err)
	}

	log.Printf("✅ Inserted %d AuthorViews", len(values)/3)
	return nil, nil
}

func (r *AuthorRepository) UpdateAuthor(author *models.Author) error {
	query := `UPDATE authors SET name = $1 WHERE id = $2`
	_, err := r.DB.Exec(query, author.Name, author.ID)
	if err != nil {
		return fmt.Errorf("failed to update author: %w", err)
	}
	return nil
}

func (r *AuthorRepository) DeleteAuthor(id string) error {
	_, err := r.DB.Exec(`DELETE FROM authors WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete author: %w", err)
	}
	return nil
}