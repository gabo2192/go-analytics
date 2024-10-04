package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"thedefiant.io/analytics/models"
	repository "thedefiant.io/analytics/repositories"
)

type PostHandler struct {
	Repo *repository.PostRepository
}

func NewPostHandler(repo *repository.PostRepository) *PostHandler {
	return &PostHandler{Repo: repo}
}

func (h *PostHandler) GetPosts(c *fiber.Ctx) error {
	posts, err := h.Repo.GetPostsFromDatabase()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching posts",
			"error":   err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"message": "Posts fetched successfully",
		"data":    posts,
	})
}

func (h *PostHandler) CreatePost(c *fiber.Ctx) error {
	posts, err := h.Repo.CreatePost()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error creating posts",
			"error":   err.Error(),
		})
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "Posts created successfully",
		"data":    posts,
	})
}

func (h *PostHandler) UpdateAnalytics(c *fiber.Ctx) error {
	dateRange := c.Query("dateRange", "yesterday")
	var err error
	var posts []models.Post

	switch dateRange {
	case "yesterday":
		posts, err = h.Repo.UpdateYesterdayViews()
	case "last7days":
		posts, err = h.Repo.UpdateLastSevenDaysViews()
	case "last14days":
		posts, err = h.Repo.UpdateLast14DaysViews()
	case "last30days":
		posts, err = h.Repo.UpdateLast30DaysViews()
	case "last90days":
		posts, err = h.Repo.UpdateLast90DaysViews()
	case "last180days":
		posts, err = h.Repo.UpdateLast180DaysViews()
	case "last365days":
		posts, err = h.Repo.UpdateLast365DaysViews()
	default:
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid date range",
		})
	}

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error updating analytics",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Analytics updated successfully",
		"data":    posts,
	})
}

