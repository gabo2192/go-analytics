package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"thedefiant.io/analytics/models"
	"thedefiant.io/analytics/storage"
)

type Post struct {
	Title string `json:"title"`
	Slug string `json:"slug"`
	AuthorId string `json:"authorId"`
}

type Repository struct {
	DB *gorm.DB
}

func (r *Repository) CreatePost(context *fiber.Ctx) error {
	post := Post{}
	err := context.BodyParser(&post)
	if err != nil {
		context.Status(http.StatusUnprocessableEntity).JSON(
			&fiber.Map{"message": "request failed"})
		return err
	}
	err = r.DB.Create(&post).Error
	if err != nil {
		context.Status(http.StatusBadRequest).JSON(
			&fiber.Map{"message": "could not create book"})
		return err
	}
	context.Status(http.StatusCreated).JSON(&fiber.Map{
		"message": "book has been added"})
	return nil
}

func (r *Repository) GetPosts(context *fiber.Ctx) error {
	postModels := &[]models.Posts{}

	err := r.DB.Find(postModels).Error
	if err != nil {
		context.Status(http.StatusBadRequest).JSON(
			&fiber.Map{"message": "could not get books"},
		)
		return err
	}
	context.Status(http.StatusOK).JSON(&fiber.Map{
		"message": "books fetched successfully",
		"data": postModels,
	})
	return nil
}

func (r *Repository) GetPostByAuthor(context *fiber.Ctx) error {
	authorId := context.Params("authorId")
	postModel := &models.Posts{}
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




func(r *Repository) SetupRoutes(app *fiber.App) {
	api := app.Group("/api")
	api.Post("/create_post", r.CreatePost)
	api.Get("/posts", r.GetPosts)
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

	log.Fatal(app.Listen(":3000"))
	
	// loc, err := time.LoadLocation("America/New_York")
	
	// if err != nil {
	// 	panic(err)
	// }
	// cronJob := cron.NewWithLocation(loc)
	
	// cronJob.AddFunc("* * * * *", func() {
	// 	fmt.Println("Hey Jude")
	// })
	// cronJob.Start()
	// select{}
}