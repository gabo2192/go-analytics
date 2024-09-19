package main

import (
	ctx "context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	sanity "github.com/sanity-io/client-go"
	"golang.org/x/oauth2/google"
	analyticsdata "google.golang.org/api/analyticsdata/v1beta"
	"google.golang.org/api/option"
	"gorm.io/gorm"
	"thedefiant.io/analytics/models"
	"thedefiant.io/analytics/storage"
)




type Repository struct {
	DB *gorm.DB
}

func (r *Repository) CreatePost() ([]models.Post, error) {
	client, err := sanity.New("6oftkxoa", sanity.WithCallbacks(
		sanity.Callbacks{
			OnQueryResult: func(result *sanity.QueryResult) {
				log.Printf("Sanity queried in %d ms!", result.Time.Milliseconds())
			},
		},
	));
	if err != nil {
		return nil, err
	}

	var posts []models.Post
	yesterday := time.Now().AddDate(0, 0, -2).Format("2006-01-02")

	result, err := client.Query("*[_type in ['blog', 'sponsor'] && defined(mainCategory) && defined(subCategory) && !(_id in path('drafts.**')) && publishedAt > $date ]| order(dateTime(publishedAt) desc){'id':_id, title,'slug': slug.current,'authorId': author._ref, 'mainCategory': mainCategory->slug.current, 'subCategory': subCategory->slug.current,publishedAt, 'createdAt': _createdAt}").Param("date", yesterday).Do(ctx.Background())

	if err != nil {
		return nil, err
	}


	if err := result.Unmarshal(&posts); err != nil {
		return nil, err
	}

	log.Printf("Sanity result: %+v", posts)
	for _, post := range posts {
		err := r.DB.Create(&post).Error
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
                log.Printf("Skipping duplicate post: %s", *post.ID)
                continue
            }
			log.Printf("Error creating post: %v", err)
		}
	}
	return posts, nil
}

func (r *Repository) GetPostsFromDatabase() ([]models.Post, error) {
	var posts []models.Post
	err := r.DB.Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}
func getDateRange(rangeType string) (string, string) {
    endDate := "yesterday"
    switch rangeType {
    case "yesterday":
        return "yesterday", endDate
    case "last7days":
        return "7daysAgo", endDate
    case "last14days":
        return "14daysAgo", endDate
    case "last30days":
        return "30daysAgo", endDate
	case "last90days":
		return "90daysAgo", endDate
    case "last180days":
        return "180daysAgo", endDate
    case "last365days":
        return "365daysAgo", endDate
    default:
        return "", ""
    }
}
func getDBFieldName(rangeType string) string {
    switch rangeType {
    case "yesterday":
        return "yesterday_views"
    case "last7days":
        return "last_seven_days_views"
    case "last14days":
        return "last_14_days_views"
    case "last30days":
        return "last_30_days_views"
    case "last90days":
        return "last_90_days_views"
    case "last180days":
        return "last_180_days_views"
    case "last365days":
        return "last_365_days_views"
    default:
        return ""
    }
}
func (r *Repository) GetAnalyticsData(dateRange string, posts []models.Post) ([]models.Post, error) {
	credJSON := []byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON"))
    creds, err := google.CredentialsFromJSON(ctx.Background(), credJSON, analyticsdata.AnalyticsReadonlyScope)
	if err != nil {
        return nil, fmt.Errorf("failed to create credentials: %v", err)
    }

	analyticsService, err := analyticsdata.NewService(ctx.Background(), option.WithCredentials(creds))

	if err != nil {
		log.Printf("Error creating analytics service: %v", err)
		return nil, err
	}
	
	startDate, endDate := getDateRange(dateRange)

	postsSlugs := make([]string, len(posts))
	for _, post := range posts {
		postsSlugs = append(postsSlugs, "/" +*post.MainCategory + "/" +*post.SubCategory + "/" + *post.Slug)
	}
	log.Printf("Posts fetched: %v", postsSlugs)

	// Define the request to fetch data
	req := &analyticsdata.RunReportRequest{
        Property: "properties/" + os.Getenv("GA4_PROPERTY_ID"),
        DateRanges: []*analyticsdata.DateRange{
            {StartDate: startDate, EndDate: endDate},
        },
        Metrics: []*analyticsdata.Metric{
            {Name: "screenPageViews"},
        },
        Dimensions: []*analyticsdata.Dimension{
            {Name: "pagePath"},
        },
		DimensionFilter: &analyticsdata.FilterExpression{
			Filter: &analyticsdata.Filter{
				FieldName: "pagePath",
				InListFilter: &analyticsdata.InListFilter{
						Values: postsSlugs,
					},
				},
		},
    }
	resp, err := analyticsService.Properties.RunReport(req.Property, req).Do()
    if err != nil {
        log.Printf("Error fetching analytics data: %v", err)
        return nil, err
    }
	var analyticsData []struct {
		PagePath       string
		ScreenPageViews int64
	}

	for _, row := range resp.Rows {
		pagePath := row.DimensionValues[0].Value
		screenPageViews, _ := strconv.ParseInt(row.MetricValues[0].Value, 10, 64)

		analyticsData = append(analyticsData, struct {
			PagePath       string
			ScreenPageViews int64
		}{
			PagePath:       pagePath,
			ScreenPageViews: screenPageViews,
		})
	}

	// Log the processed data
	
	// Create a map to store view counts
    viewCounts := make(map[string]int64)
    for _, data := range analyticsData {
        viewCounts[data.PagePath] = data.ScreenPageViews
    }
	
    fieldName := getDBFieldName(dateRange)

    // Update posts with view counts
    for i, post := range posts {
        slug := "/" + *post.MainCategory + "/" + *post.SubCategory + "/" +*post.Slug
        if views, ok := viewCounts[slug]; ok {
            posts[i].LastSevenDaysViews = views
            // Update the database
            err := r.DB.Model(&post).Update(fieldName, views).Error
            if err != nil {
                log.Printf("Error updating post %s: %v", *post.ID, err)
            }

        }
    }
	
	return posts, nil
}

func (r *Repository) UpdateYesterdayViews() ([]models.Post, error) {
	var dbPosts []models.Post
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	err := r.DB.Where("DATE(published_at) = ?", yesterday).Find(&dbPosts).Error
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		return nil, err
	}
	if len(dbPosts) == 0 {
		log.Println("No posts found for yesterday")
		return nil, nil
	}
	
	analyticsPosts, err := r.GetAnalyticsData("yesterday", dbPosts)
	if err != nil {
		log.Printf("Error fetching analytics data: %v", err)
		return nil, err
	}
	return analyticsPosts, nil
}
func (r *Repository) UpdateLastSevenDaysViews() ([]models.Post, error) {
	var dbPosts []models.Post
	sevenDaysAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	err := r.DB.Where("DATE(published_at) = ?", sevenDaysAgo).Find(&dbPosts).Error
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		return nil, err
	}
	if len(dbPosts) == 0 {
		log.Println("No posts found for seven days ago")
		return nil, nil
	}
	
	analyticsPosts, err := r.GetAnalyticsData("last7days", dbPosts)
	if err != nil {
		log.Printf("Error fetching analytics data: %v", err)
		return nil, err
	}
	return analyticsPosts, nil
}

func (r *Repository) UpdateLast14DaysViews() ([]models.Post, error) {
	var dbPosts []models.Post
	fourteenDaysAgo := time.Now().AddDate(0, 0, -14).Format("2006-01-02")
	err := r.DB.Where("DATE(published_at) = ?", fourteenDaysAgo).Find(&dbPosts).Error
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		return nil, err
	}
	if len(dbPosts) == 0 {
		log.Println("No posts found for fourteen days ago")
		return nil, nil
	}
	
	analyticsPosts, err := r.GetAnalyticsData("last14days", dbPosts)
	if err != nil {
		log.Printf("Error fetching analytics data: %v", err)
		return nil, err
	}
	return analyticsPosts, nil
}

func (r *Repository) UpdateLast30DaysViews() ([]models.Post, error) {
	var dbPosts []models.Post
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	err := r.DB.Where("DATE(published_at) = ?", thirtyDaysAgo).Find(&dbPosts).Error
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		return nil, err
	}
	if len(dbPosts) == 0 {
		log.Println("No posts found for thirty days ago")
		return nil, nil
	}
	
	analyticsPosts, err := r.GetAnalyticsData("last30days", dbPosts)
	if err != nil {
		log.Printf("Error fetching analytics data: %v", err)
		return nil, err
	}
	return analyticsPosts, nil
}

func (r *Repository) UpdateLast90DaysViews() ([]models.Post, error) {
	var dbPosts []models.Post
	ninetyDaysAgo := time.Now().AddDate(0, 0, -90).Format("2006-01-02")
	err := r.DB.Where("DATE(published_at) = ?", ninetyDaysAgo).Find(&dbPosts).Error
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		return nil, err
	}	
	if len(dbPosts) == 0 {
		log.Println("No posts found for ninety days ago")
		return nil, nil
	}
	
	analyticsPosts, err := r.GetAnalyticsData("last90days", dbPosts)
	if err != nil {
		log.Printf("Error fetching analytics data: %v", err)
		return nil, err
	}
	return analyticsPosts, nil
}

func (r *Repository) UpdateLast180DaysViews() ([]models.Post, error) {
	var dbPosts []models.Post
	oneHundredEightyDaysAgo := time.Now().AddDate(0, 0, -180).Format("2006-01-02")
	err := r.DB.Where("DATE(published_at) = ?", oneHundredEightyDaysAgo).Find(&dbPosts).Error	
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		return nil, err
	}
	if len(dbPosts) == 0 {
		log.Println("No posts found for one hundred eighty days ago")
		return nil, nil
	}

	analyticsPosts, err := r.GetAnalyticsData("last180days", dbPosts)
	if err != nil {
		log.Printf("Error fetching analytics data: %v", err)
		return nil, err
	}
	return analyticsPosts, nil
}

func (r *Repository) UpdateLast365DaysViews() ([]models.Post, error) {
	var dbPosts []models.Post
	threeHundredSixtyFiveDaysAgo := time.Now().AddDate(0, 0, -365).Format("2006-01-02")
	err := r.DB.Where("DATE(published_at) = ?", threeHundredSixtyFiveDaysAgo).Find(&dbPosts).Error
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		return nil, err
	}
	if len(dbPosts) == 0 {
		log.Println("No posts found for three hundred sixty five days ago")
		return nil, nil
	}
	
	analyticsPosts, err := r.GetAnalyticsData("last365days", dbPosts)
	if err != nil {
		log.Printf("Error fetching analytics data: %v", err)
		return nil, err
	}
	return analyticsPosts, nil
}


func (r *Repository) GetPosts(context *fiber.Ctx) error {
	posts, err := r.GetPostsFromDatabase()
	
	if err != nil {
		context.Status(http.StatusBadRequest).JSON(
			&fiber.Map{"message": "could not get books"},
		)
		return err
	}
	
	context.Status(http.StatusOK).JSON(&fiber.Map{
		"message": "posts fetched successfully",
		"data": posts,
	})
	return nil
}

func (r *Repository) GetPostByAuthor(context *fiber.Ctx) error {
	authorId := context.Params("authorId")
	postModel := &models.Post{}
	if authorId == ""{
		context.Status(http.StatusInternalServerError).JSON(&fiber.Map{
			"message": "Author Id cannot be empty",
		})
		return nil
	}

	fmt.Println("the Author ID is", authorId)

	err := r.DB.Where("authorId = ?", authorId).Find(postModel).Error
	if err != nil {
		context.Status(http.StatusBadRequest).JSON(
			&fiber.Map{"message": "could not get books"},
		)
	}
	context.Status(http.StatusOK).JSON(&fiber.Map{
		"message": "posts by author id fetched successfully",
		"data": postModel,
	})
	return nil
}

func (r *Repository) CreatePostRoute(context *fiber.Ctx) error {
	posts, err := r.CreatePost()
	if err != nil {
		context.Status(http.StatusBadRequest).JSON(
			&fiber.Map{"message": "could not create posts"},
		)
		return err
	}
	context.Status(http.StatusOK).JSON(&fiber.Map{
		"message": "posts created successfully",
		"data": posts,
	})
	return nil
}

func(r *Repository) SetupRoutes(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/posts", r.GetPosts)
	api.Post("/posts", r.CreatePostRoute)
	api.Get("/get_posts/:authorId", r.GetPostByAuthor)
} 

func main(){
	err := godotenv.Load(".env")
	
	if err != nil {
		log.Fatal(err)
	}
	config := &storage.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Password: os.Getenv("DB_PASS"),
		User:     os.Getenv("DB_USER"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:   os.Getenv("DB_NAME"),
	}
	db, err := storage.NewConnection(config)

	if err != nil {
		log.Fatal("could not load the database")
	}
	err = models.MigratePosts(db)

	if err != nil {
		log.Fatal("could not migrate db")
	}

	r := Repository {
		DB: db,
	}

	app := fiber.New()

	r.SetupRoutes(app)

	// Set up cron jobs
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}
	cronJob := cron.New(cron.WithLocation(loc))

	// Fetch posts daily at 8:00 PM
	cronJob.AddFunc("0 20 * * *", func() {
		log.Println("Fetching posts")
		posts, err := r.CreatePost()
		if err != nil {
			log.Printf("Error fetching posts: %v", err)
		}
		log.Printf("Posts fetched: %v", posts)
	})

	// Fetch Google Analytics data daily at 11:30 PM
	cronJob.AddFunc("10 23 * * *", func() {
		log.Println("Fetching Google Analytics data For Yesterday")
		
		_, err := r.UpdateYesterdayViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully")
	})

	cronJob.AddFunc("20 23 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 7 Days")
		
		_, err := r.UpdateLastSevenDaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully")
	})

	cronJob.AddFunc("30 23 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 14 Days")
		
		_, err := r.UpdateLast14DaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully")
	})

	cronJob.AddFunc("40 23 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 30 Days")
		
		_, err := r.UpdateLast30DaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully")
	})

	cronJob.AddFunc("50 23 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 90 Days")
		
		_, err := r.UpdateLast90DaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully")
	})

	cronJob.AddFunc("0 0 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 180 Days")
		
		_, err := r.UpdateLast180DaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully")
	})

	cronJob.AddFunc("10 0 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 365 Days")
		
		_, err := r.UpdateLast365DaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully")
	})

	cronJob.Start()

	// Log a message to confirm the server is starting
	log.Println("Server starting on :8000")

	// Use go routine to start the server
	go func() {
		if err := app.Listen(":8000"); err != nil {
			log.Fatal(err)
		}
	}()

	// Keep the main function running
	select {}
}