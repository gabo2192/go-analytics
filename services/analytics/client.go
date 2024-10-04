package analytics

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/oauth2/google"
	analyticsdata "google.golang.org/api/analyticsdata/v1beta"
	"google.golang.org/api/option"
	"thedefiant.io/analytics/utils"
)

// Client wraps the Google Analytics Data API client
type Client struct {
	service *analyticsdata.Service
	propID  string
}

// NewClient creates a new Google Analytics Data API client
func NewClient() (*Client, error) {
	ctx := context.Background()

	credJSON := []byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON"))
	creds, err := google.CredentialsFromJSON(ctx, credJSON, analyticsdata.AnalyticsReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	service, err := analyticsdata.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create analytics service: %w", err)
	}

	propID := os.Getenv("GA4_PROPERTY_ID")
	if propID == "" {
		return nil, fmt.Errorf("GA4_PROPERTY_ID environment variable is not set")
	}

	return &Client{
		service: service,
		propID:  propID,
	}, nil
}

// GetPageViews retrieves page views for the given slugs within the specified date range
func (c *Client) GetPageViews(dateRange string, slugs []string) (map[string]int64, error) {
	startDate, endDate := utils.GetDateRange(dateRange)
	fmt.Println(c)

	req := &analyticsdata.RunReportRequest{
		Property: "properties/" + c.propID,
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
					Values: slugs,
				},
			},
		},
	}
	resp, err := c.service.Properties.RunReport(req.Property, req).Do()
	
	if err != nil {
		return nil, fmt.Errorf("failed to run analytics report: %w", err)
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
	viewCounts := make(map[string]int64)
    for _, data := range analyticsData {
        viewCounts[data.PagePath] = data.ScreenPageViews
    }
	return viewCounts, nil
}

