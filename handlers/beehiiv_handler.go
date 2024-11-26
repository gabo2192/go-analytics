// handlers/beehiiv_handler.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	repository "thedefiant.io/analytics/repositories"
)

type BeehiivHandler struct {
	Repo *repository.BeehiivMetricsRepository
}

func NewBeehiivHandler(repo *repository.BeehiivMetricsRepository) *BeehiivHandler {
	return &BeehiivHandler{Repo: repo}
}

// GetAllPostMetrics retrieves metrics for all posts within a specified time range
func (h *BeehiivHandler) GetAllPostMetrics(c *fiber.Ctx) error {
	// Default to 30 days if no days parameter is provided
	days, err := strconv.Atoi(c.Query("days", "30"))
	if err != nil || days <= 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid days parameter",
			"error":   "Days must be a positive integer",
		})
	}

	metrics, err := h.Repo.GetPostMetrics(days)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching post metrics",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Post metrics fetched successfully",
		"data":    metrics,
	})
}

func (h *BeehiivHandler) GetWeekPostMetrics(c *fiber.Ctx) error {
	metrics, err := h.Repo.GetWeekPostMetrics()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching post metrics",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Post metrics fetched successfully",
		"data":    metrics,
	})
}
func (h *BeehiivHandler) GetWeekAlphaMetrics(c *fiber.Ctx) error {
	metrics, err := h.Repo.GetWeekAlphaMetrics()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching post metrics",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Post metrics fetched successfully",
		"data":    metrics,
	})
}

// GetPostMetricsByID retrieves metrics for a specific post
func (h *BeehiivHandler) GetPostMetricsByID(c *fiber.Ctx) error {
	postID := c.Params("postId")
	if postID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Post ID cannot be empty",
		})
	}

	metrics, err := h.Repo.GetMetricsByPostID(postID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching post metrics",
			"error":   err.Error(),
		})
	}

	if len(metrics) == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"message": "No metrics found for this post",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Post metrics fetched successfully",
		"data":    metrics,
	})
}

// GetTopPerformingPosts retrieves the top performing posts based on email open rate
func (h *BeehiivHandler) GetTopPerformingPosts(c *fiber.Ctx) error {
	// Default to top 10 if no limit parameter is provided
	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit <= 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid limit parameter",
			"error":   "Limit must be a positive integer",
		})
	}

	metrics, err := h.Repo.GetTopPerformingPosts(limit)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching top performing posts",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Top performing posts fetched successfully",
		"data":    metrics,
	})
}

// UpdatePostMetrics triggers a manual update of post metrics
func (h *BeehiivHandler) UpdatePostMetrics(c *fiber.Ctx) error {
	err := h.Repo.UpdatePostMetrics()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error updating post metrics",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Post metrics updated successfully",
	})
}