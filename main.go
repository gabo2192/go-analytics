package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"thedefiant.io/analytics/handlers"
	"thedefiant.io/analytics/models"
	repository "thedefiant.io/analytics/repositories"
	"thedefiant.io/analytics/services/analytics"
	"thedefiant.io/analytics/services/sanity"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Database connection
	dbURL := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto Migrate
	err = db.AutoMigrate(&models.Post{}, &models.Author{})
	if err != nil {
		log.Fatalf("Failed to auto migrate: %v", err)
	}

	sanityClient, err := sanity.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Sanity client: %v", err)
	}

	analyticsClient, err := analytics.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Analytics client: %v", err)
	}

	postRepo := repository.NewPostRepository(db, sanityClient, analyticsClient)
	authorRepo := repository.NewAuthorRepository(db, sanityClient, analyticsClient)

	postHandler := handlers.NewPostHandler(postRepo)
	authorHandler := handlers.NewAuthorHandler(authorRepo)

	// Set up cron jobs
	cronJob := cron.New(cron.WithLocation(time.UTC))

	// Fetch posts daily at 8:00 PM UTC
	_, err = cronJob.AddFunc("0 20 * * *", func() {
		log.Println("Fetching posts")
		_, err := postRepo.CreatePost()
		if err != nil {
			log.Printf("Error fetching posts: %v", err)
		}
		log.Println("Posts fetched successfully")
	})
	if err != nil {
		log.Printf("Error setting up post fetch cron job: %v", err)
	}

	// Update analytics data
	_, err = cronJob.AddFunc("10 23 * * *", func() {
		log.Println("Updating yesterday's analytics")
		_, err := postRepo.UpdateYesterdayViews()
		if err != nil {
			log.Printf("Error updating yesterday's views: %v", err)
		}
		log.Println("Yesterday's analytics updated successfully")
	})
	if err != nil {
		log.Printf("Error setting up yesterday analytics update cron job: %v", err)
	}

	_, err = cronJob.AddFunc("20 23 * * *", func() {
		log.Println("Updating last 7 days analytics")
		_, err := postRepo.UpdateLastSevenDaysViews()
		if err != nil {
			log.Printf("Error updating last 7 days views: %v", err)
		}
		log.Println("Last 7 days analytics updated successfully")
	})
	if err != nil {
		log.Printf("Error setting up last 7 days analytics update cron job: %v", err)
	}

	_, err = cronJob.AddFunc("30 23 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 14 Days")
		_, err := postRepo.UpdateLast14DaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully for Last 14 Days")
	})
	if err != nil {
		log.Printf("Error setting up last 14 days analytics update cron job: %v", err)
	}

	_, err = cronJob.AddFunc("40 23 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 30 Days")
		_, err := postRepo.UpdateLast30DaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully for Last 30 Days")
	})
	if err != nil {
		log.Printf("Error setting up last 30 days analytics update cron job: %v", err)
	}

	_, err = cronJob.AddFunc("50 23 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 90 Days")
		_, err := postRepo.UpdateLast90DaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully for Last 90 Days")
	})
	if err != nil {
		log.Printf("Error setting up last 90 days analytics update cron job: %v", err)
	}

	_, err = cronJob.AddFunc("0 0 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 180 Days")
		_, err := postRepo.UpdateLast180DaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully for Last 180 Days")
	})
	if err != nil {
		log.Printf("Error setting up last 180 days analytics update cron job: %v", err)
	}

	_, err = cronJob.AddFunc("10 0 * * *", func() {
		log.Println("Fetching Google Analytics data For Last 365 Days")
		_, err := postRepo.UpdateLast365DaysViews()
		if err != nil {
			log.Printf("Error fetching Google Analytics data: %v", err)
		}
		log.Println("Google Analytics data fetched successfully for Last 365 Days")
	})
	if err != nil {
		log.Printf("Error setting up last 365 days analytics update cron job: %v", err)
	}
	_, err = cronJob.AddFunc("0 6 1 * *", func() {
		log.Println("Updating monthly author analytics")
		_, err := authorRepo.UpdateAnalyticsViews()
		if err != nil {
			log.Printf("Error updating monthly author analytics: %v", err)
		} else {
			log.Println("Monthly author analytics updated successfully")
		}
	})
	if err != nil {
		log.Printf("Error setting up monthly author analytics update cron job: %v", err)
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" // Default port if not specified
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}