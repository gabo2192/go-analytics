package utils

import (
	"time"
)

// GetDateRange returns the start and end dates for a given range type
func GetDateRange(rangeType string) (string, string) {
	endDate := "yesterday"
	switch rangeType {
	case "yesterday":
		return "yesterday", endDate
	case "last7days":
		return "7daysAgo", endDate
	case "last14days":
		return "14daysAgo", endDate
	case "last30days":
		return "30daysAgo", endDate
	case "last90days":
		return "90daysAgo", endDate
	case "last180days":
		return "180daysAgo", endDate
	case "last365days":
		return "365daysAgo", endDate
	default:
		return "", ""
	}
}


func GetDaysFromRangeType(rangeType string) int {
	switch rangeType {
	case "yesterday":
		return 1
	case "last7days":
		return 7
	case "last14days":
		return 14
	case "last30days":
		return 30
	case "last90days":
		return 90
	case "last180days":
		return 180
	case "last365days":
		return 365
	default:
		return 0
	}
}
// GetDBFieldName returns the database field name for a given range type
func GetDBFieldName(rangeType string) string {
	switch rangeType {
	case "yesterday":
		return "yesterday_views"
	case "last7days":
		return "last_seven_days_views"
	case "last14days":
		return "last_14_days_views"
	case "last30days":
		return "last_30_days_views"
	case "last90days":
		return "last_90_days_views"
	case "last180days":
		return "last_180_days_views"
	case "last365days":
		return "last_365_days_views"
	default:
		return ""
	}
}

// FormatDate returns a formatted date string in the format "2006-01-02"
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// GetYesterdayDate returns yesterday's date as a string in the format "2006-01-02"
func GetYesterdayDate() string {
	return FormatDate(time.Now().AddDate(0, 0, -1))
}

// GetDateNDaysAgo returns the date N days ago as a string in the format "2006-01-02"
func GetDateNDaysAgo(n int) string {
	return FormatDate(time.Now().AddDate(0, 0, -n))
}

// ParseDate parses a date string in the format "2006-01-02" and returns a time.Time
func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// IsValidDateRange checks if startDate is before or equal to endDate
func IsValidDateRange(startDate, endDate string) bool {
	start, err1 := ParseDate(startDate)
	end, err2 := ParseDate(endDate)
	if err1 != nil || err2 != nil {
		return false
	}
	return start.Before(end) || start.Equal(end)
}

// ContainsString checks if a string slice contains a specific string
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}