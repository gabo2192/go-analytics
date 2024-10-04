package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	repository "thedefiant.io/analytics/repositories"
)

type AuthorHandler struct {
	Repo *repository.AuthorRepository
}

func NewAuthorHandler(repo *repository.AuthorRepository) *AuthorHandler {
	return &AuthorHandler{Repo: repo}
}

func (h *AuthorHandler) GetAuthors(c *fiber.Ctx) error {
	authors, err := h.Repo.GetAuthorsFromDatabase()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching authors",
			"error":   err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"message": "Authors fetched successfully",
		"data":    authors,
	})
}

func (h *AuthorHandler) CreateAuthor(c *fiber.Ctx) error {
	authors, err := h.Repo.CreateAuthor()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error creating authors",
			"error":   err.Error(),
		})
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "Authors created successfully",
		"data":    authors,
	})
}

func (h *AuthorHandler) GetAuthorByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Author ID cannot be empty",
		})
	}

	author, err := h.Repo.GetAuthorByID(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching author",
			"error":   err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"message": "Author fetched successfully",
		"data":    author,
	})
}

