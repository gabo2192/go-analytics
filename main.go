package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"thedefiant.io/analytics/handlers"
	repository "thedefiant.io/analytics/repositories"
	"thedefiant.io/analytics/services/analytics"
	"thedefiant.io/analytics/services/beehiiv"
	"thedefiant.io/analytics/services/sanity"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Database connection
	dbURL := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	sanityClient, err := sanity.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Sanity client: %v", err)
	}

	analyticsClient, err := analytics.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Analytics client: %v", err)
	}

	beehiivClient, err := beehiiv.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Beehiiv client: %v", err)
	}

	postRepo := repository.NewPostRepository(db, sanityClient, analyticsClient)
	authorRepo := repository.NewAuthorRepository(db, sanityClient, analyticsClient)
	beehiivRepo := repository.NewBeehiivMetricsRepository(db, beehiivClient)

	postHandler := handlers.NewPostHandler(postRepo)
	authorHandler := handlers.NewAuthorHandler(authorRepo)
	beehiivHandler := handlers.NewBeehiivHandler(beehiivRepo)

	// Set up cron jobs
	cronJob := cron.New(cron.WithLocation(time.UTC))

	_, err = cronJob.AddFunc("0 12 * * 0", func() {
		log.Println("Fetching Beehiiv metrics")
		sevenDaysAgo := time.Now().AddDate(0, 0, -7)
		err := beehiivRepo.UpdatePostMetrics(&sevenDaysAgo)
		if err != nil {
			log.Printf("Error fetching Beehiiv metrics: %v", err)
		}
		log.Println("Beehiiv metrics fetched successfully")
	})
	

	if err != nil {
		log.Printf("Error setting up cron job: %v", err)
	}

	// Start the cron job scheduler
	cronJob.Start()

	app := fiber.New()

	// Post routes
	app.Get("/api/posts", postHandler.GetPosts)
	app.Post("/api/posts", postHandler.CreatePost)
	app.Post("/api/posts/update-analytics", postHandler.UpdateAnalytics)

	// Author routes
	app.Get("/api/authors", authorHandler.GetAuthors)
	app.Post("/api/authors", authorHandler.CreateAuthor)
	app.Get("/api/authors/:id", authorHandler.GetAuthorByID)

	// Beehiiv
	app.Get("/api/beehiiv/update",beehiivHandler.UpdatePostMetrics)
	app.Get("/api/beehiiv/posts",beehiivHandler.GetWeekPostMetrics)
	app.Get("/api/beehiiv/alpha",beehiivHandler.GetWeekAlphaMetrics)
	app.Get("/api/beehiiv/free", beehiivHandler.GetFreePostsMetrics)
	app.Get("/api/beehiiv/free-month", beehiivHandler.GeMonthFreeMetrics)
	app.Get("/api/beehiiv/alpha-month", beehiivHandler.GeMonthAlphaMetrics)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" // Default port if not specified
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}