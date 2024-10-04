package sanity

import (
	"context"
	"fmt"
	"log"

	sanity "github.com/sanity-io/client-go"
)

// Client wraps the Sanity client
type Client struct {
	client *sanity.Client
}

// NewClient creates a new Sanity client
func NewClient() (*Client, error) {
	projectID := "6oftkxoa"
	if projectID == "" {
		return nil, fmt.Errorf("SANITY_PROJECT_ID environment variable is not set")
	}

	client, err := sanity.New(projectID, sanity.WithCallbacks(
		sanity.Callbacks{
			OnQueryResult: func(result *sanity.QueryResult) {
				log.Printf("Sanity queried in %d ms!", result.Time.Milliseconds())
			},
		},
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create Sanity client: %w", err)
	}

	return &Client{client: client}, nil
}

// Query executes a GROQ query against the Sanity dataset
func (c *Client) Query(query string, params map[string]interface{}) (*sanity.QueryResult, error) {
	ctx := context.Background()
	q := c.client.Query(query)

	for k, v := range params {
		q = q.Param(k, v)
	}

	result, err := q.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Sanity query: %w", err)
	}

	return result, nil
}

// Unmarshal unmarshals the query result into the provided interface
func (c *Client) Unmarshal(result *sanity.QueryResult, v interface{}) error {
	err := result.Unmarshal(v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Sanity result: %w", err)
	}
	return nil
}