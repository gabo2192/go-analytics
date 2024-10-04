package repository

import (
	"fmt"
	"log"

	"gorm.io/gorm"
	"thedefiant.io/analytics/models"
	"thedefiant.io/analytics/services/analytics"
	"thedefiant.io/analytics/services/sanity"
	"thedefiant.io/analytics/utils"
)

type PostRepository struct {
	DB        *gorm.DB 
	Sanity    *sanity.Client
	Analytics *analytics.Client
}

func NewPostRepository(db *gorm.DB, sanityClient *sanity.Client, analyticsClient *analytics.Client) *PostRepository {
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

	for _, post := range posts {
		err := r.DB.Create(&post).Error
		if err != nil {
			if err.Error() == "duplicate key value violates unique constraint" {
				log.Printf("Skipping duplicate post: %s", *post.ID)
				continue
			}
			log.Printf("Error creating post: %v", err)
		}
	}
	return posts, nil
}

func (r *PostRepository) GetPostsFromDatabase() ([]models.Post, error) {
	var posts []models.Post
	err := r.DB.Find(&posts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posts from database: %w", err)
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

	fieldName := utils.GetDBFieldName(dateRange)
	for i, post := range posts {
		slug := "/" + *post.MainCategory + "/" + *post.SubCategory + "/" + *post.Slug
		if views, ok := pageViews[slug]; ok {
			err := r.DB.Model(&posts[i]).Update(fieldName, views).Error
			if err != nil {
				log.Printf("Error updating post %s: %v", *post.ID, err)
			}
		}
	}

	return posts, nil
}

func (r *PostRepository) UpdateYesterdayViews() ([]models.Post, error) {
	var dbPosts []models.Post
	yesterday := utils.GetDateNDaysAgo(2)
	err := r.DB.Where("DATE(published_at) >= ?", yesterday).Find(&dbPosts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posts for yesterday: %w", err)
	}
	if len(dbPosts) == 0 {
		log.Println("No posts found for yesterday")
		return nil, nil
	}
	
	return r.GetAnalyticsData("yesterday", dbPosts)
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
	var dbPosts []models.Post
	nDaysAgo := utils.GetDateNDaysAgo(utils.GetDaysFromRangeType(rangeType))
	
	err := r.DB.Where("DATE(published_at) = ?", nDaysAgo).Find(&dbPosts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posts for %s: %w", rangeType, err)
	}
	if len(dbPosts) == 0 {
		log.Printf("No posts found for %s", rangeType)
		return nil, nil
	}
	
	return r.GetAnalyticsData(rangeType, dbPosts)
}