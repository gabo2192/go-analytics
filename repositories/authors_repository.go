package repository

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"thedefiant.io/analytics/models"
	"thedefiant.io/analytics/services/analytics"
	"thedefiant.io/analytics/services/sanity"
	"thedefiant.io/analytics/utils"
)

type AuthorRepository struct {
	DB     *gorm.DB
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


func NewAuthorRepository(db *gorm.DB, sanityClient *sanity.Client, analyticsClient *analytics.Client) *AuthorRepository {
	return &AuthorRepository{
		DB:     db,
		Sanity: sanityClient,
		Analytics: analyticsClient,
	}
}

func (r *AuthorRepository) CreateAuthor() ([]models.Author, error) {
	thirtyDaysAgo := utils.GetDateNDaysAgo(30)
	query := `*[_type == 'author'&& _createdAt > $date]{'id':_id,name}`
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

	for _, author := range authors {
		err := r.DB.Create(&author).Error
		if err != nil {
			if err.Error() == "duplicate key value violates unique constraint" {
				log.Printf("Skipping duplicate author: %s", *author.ID)
				continue
			}
			log.Printf("Error creating author: %v", err)
		}
	}
	return authors, nil
}

func (r *AuthorRepository) GetAuthorsFromDatabase() ([]AuthorsResponse, error) {
	var authorsWithViews []AuthorWithViews

	// Fetch all authors with their views
	err := r.DB.Model(&models.Author{}).
		Select("authors.*, author_views.views, author_views.created_at").
		Joins("LEFT JOIN author_views ON authors.id = author_views.author_id").
		Order("authors.id, author_views.created_at DESC").
		Where("author_views.views IS NOT NULL").
		Find(&authorsWithViews).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch authors with views: %w", err)
	}

	// map author views into author response struct appending views if author exists
	authors := make([]AuthorsResponse, 0)
	for _, author := range authorsWithViews {
		if len(authors) == 0 || *authors[len(authors)-1].ID != *author.ID {
			authors = append(authors, AuthorsResponse{
				Author: models.Author{
					ID: author.ID,
					Name: author.Name,
					CreatedAt: author.CreatedAt,
				},
				Views: []AuthorView{
					{
						View: 	author.Views,
						CreatedAt: author.CreatedAt,
					},
				},
			})
	} else {
		authors[len(authors)-1].Views = append(authors[len(authors)-1].Views, AuthorView{
			View: author.Views,
			CreatedAt: author.CreatedAt,
		})
	}
}

	

	return authors, nil
}


func (r *AuthorRepository) GetAuthorByID(id string) (*models.Author, error) {
	var author models.Author
	err := r.DB.Where("id = ?", id).First(&author).Error
	if err != nil {
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

func (r *AuthorRepository) UpdateAnalyticsViews()([]models.AuthorViews, error){
	authorsAndSlugs, err := r.GetSanityPostsByAuthorCurrentMonth()
	if err != nil {
		return nil, fmt.Errorf("failed to get authors and slugs: %w", err)
	}
	authorViews:= make([]models.AnalyticsReponse, len(authorsAndSlugs))
	for i, author := range authorsAndSlugs {
		
		authorViews[i].ID = author.ID
		
		pageViews, err := r.Analytics.GetPageViews("last30days", author.Slugs)
		
		if err != nil {
			return nil, fmt.Errorf("failed to get page views: %w", err)
		}
		for _, slug := range author.Slugs {
			authorViews[i].Views += pageViews[slug]
		}	
	}
	authorViewsDB := make([]models.AuthorViews, len(authorViews))
	for i, authorView := range authorViews {
		authorViewsDB[i] = models.AuthorViews{
			AuthorId: &authorView.ID,
			Views: authorView.Views,
			CreatedAt: time.Now(),
		}
		err := r.DB.Create(&authorViewsDB[i]).Error
		if err != nil {
			log.Printf("Error creating author views: %v", err)
		}
	}
	return authorViewsDB, nil
}

func (r *AuthorRepository) UpdateAuthor(author *models.Author) error {
	return r.DB.Save(author).Error
}

func (r *AuthorRepository) DeleteAuthor(id string) error {
	return r.DB.Delete(&models.Author{}, "id = ?", id).Error
}