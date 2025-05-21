package models

import "time"

type Post struct {
	ID                 *string   `json:"id"`
	Title              *string   `json:"title"`
	Slug               *string   `json:"slug"`
	AuthorId           *string   `json:"authorId"`
	MainCategory       *string   `json:"mainCategory"`
	SubCategory        *string   `json:"subCategory"`
	PublishedAt        time.Time `json:"publishedAt"`
	YesterdayViews     int64     `json:"yesterdayViews"`
	LastSevenDaysViews int64     `json:"lastSevenDaysViews"`
	Last14DaysViews    int64     `json:"last14DaysViews"`
	Last30DaysViews    int64     `json:"last30DaysViews"`
	Last90DaysViews    int64     `json:"last90DaysViews"`
	Last180DaysViews   int64     `json:"last180DaysViews"`
	Last365DaysViews   int64     `json:"last365DaysViews"`
	CreatedAt          time.Time `json:"createdAt"`
}
