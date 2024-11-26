// services/beehiiv/client.go
package beehiiv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Client struct {
	apiKey        string
	publicationID string
	baseURL       string
	httpClient    *http.Client
}

type PostResponse struct {
	Data []Post `json:"data"`
}

type Post struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Slug        string      `json:"slug"`
	PublishDate int64       `json:"publish_date"`
	Stats       PostMetrics `json:"stats"`
}

type PostMetrics struct {
	Email EmailMetrics `json:"email"`
	Web   WebMetrics  `json:"web"`
}

type EmailMetrics struct {
	Recipients    int `json:"recipients"`
	Delivered    int `json:"delivered"`
	Opens        int `json:"opens"`
	UniqueOpens  int `json:"unique_opens"`
	Clicks       int `json:"clicks"`
	UniqueClicks int `json:"unique_clicks"`
}

type WebMetrics struct {
	Views  int `json:"views"`
	Clicks int `json:"clicks"`
}

func NewClient() (*Client, error) {
	apiKey := os.Getenv("BEEHIIV_API_KEY")
	publicationID := os.Getenv("BEEHIIV_PUBLICATION_ID")
	if apiKey == "" {
		return nil, fmt.Errorf("BEEHIIV_API_KEY environment variable is not set")
	}
	if publicationID == "" {
		return nil, fmt.Errorf("BEEHIIV_PUBLICATION_ID environment variable is not set")
	}

	return &Client{
		apiKey:        apiKey,
		publicationID: publicationID,
		baseURL:       "https://api.beehiiv.com/v2",
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}, nil
}

func (c *Client) GetPosts(page int) (*PostResponse, error) {
	endpoint := fmt.Sprintf("%s/publications/%s/posts?expand=stats&limit=100&page=%d&direction=desc&order_by=publish_date", 
		c.baseURL, 
		c.publicationID,
		page,
	)
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var postResp PostResponse
	if err := json.NewDecoder(resp.Body).Decode(&postResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &postResp, nil
}

func (c *Client) GetPostByID(postID string) (*Post, error) {
	endpoint := fmt.Sprintf("%s/publications/%s/posts/%s?expand=stats", 
		c.baseURL, 
		c.publicationID,
		postID,
	)
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var postResp struct {
		Data Post `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&postResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &postResp.Data, nil
}