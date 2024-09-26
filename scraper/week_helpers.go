package scraper

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
)

// calculateLessonDate calculates the date of the lesson based on the week start date and the day.
func calculateLessonDate(week Week, day string) time.Time {
	// Parse the week start date (format: "02/01/2006")
	startDate, err := time.Parse("02/01/2006", week.StartDate)
	if err != nil {
		fmt.Printf("Error parsing start date for week %v: %v\n", week, err)
		return time.Time{} // Return zero value for time.Time if parsing fails
	}

	// Convert the lesson day into a time.Weekday
	lessonDay := map[string]time.Weekday{
		"Monday":    time.Monday,
		"Tuesday":   time.Tuesday,
		"Wednesday": time.Wednesday,
		"Thursday":  time.Thursday,
		"Friday":    time.Friday,
		"Saturday":  time.Saturday,
		"Sunday":    time.Sunday,
	}[day]

	// Find the difference in days between the start of the week and the lesson day.
	weekdayOffset := int(lessonDay - startDate.Weekday())

	// Add the offset to the start date to get the actual lesson date.
	lessonDate := startDate.AddDate(0, 0, weekdayOffset)

	fmt.Printf("Week Start Date: %v, Lesson Day: %s, Calculated Lesson Date: %v\n", startDate, day, lessonDate)

	return lessonDate
}

// convertToFullWeekday converts abbreviated weekday to full name.
func convertToFullWeekday(abbreviatedDay string) string {
	daysMapping := map[string]string{
		"Mon": "Monday", "Tue": "Tuesday", "Wed": "Wednesday",
		"Thu": "Thursday", "Fri": "Friday", "Sat": "Saturday", "Sun": "Sunday",
	}
	return daysMapping[abbreviatedDay]
}

// parseTimeRange parses a time range string like "09:00 - 10:00".
func parseTimeRange(timeRange string) (string, string) {
	times := strings.Split(timeRange, "-")
	startTime := strings.TrimSpace(times[0])
	endTime := strings.TrimSpace(times[1])
	return startTime, endTime
}

// getLessonType determines the lesson type based on the parent class.
func getLessonType(s *goquery.Selection) int {
	parentClass := s.Parent().Parent().AttrOr("class", "")
	switch {
	case strings.Contains(parentClass, "panel-info"):
		return 1
	case strings.Contains(parentClass, "panel-warning"):
		return 2
	case strings.Contains(parentClass, "panel-danger"):
		return 3
	default:
		return 0
	}
}

// fetchWeekDatesPlaywright uses Playwright to extract the start date for a specific week.
func fetchWeekDatesPlaywright(page playwright.Page, weekURL string) string {
	fmt.Printf("Fetching week data from URL: %s\n", weekURL)

	// Navigate to the week's page
	if _, err := page.Goto(weekURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		fmt.Printf("Could not navigate to week page: %v\n", err)
		return ""
	}

	// Scrape the content of the week page
	weekHTML, err := page.Content()
	if err != nil {
		fmt.Printf("Could not get week page content: %v\n", err)
		return ""
	}

	// Parse the week HTML content
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(weekHTML))
	if err != nil {
		fmt.Println("Error parsing week page HTML:", err)
		return ""
	}

	// Find the paragraph element that contains the week dates
	dateText := doc.Find(".page-header p").Text()
	fmt.Printf("Raw date text from week page %s: %s\n", weekURL, dateText)

	// Example expected format: "Year 2024-25 | Term 1 | Week 1 | 23/09/2024 - 29/09/2024"
	parts := strings.Split(dateText, "|")
	if len(parts) < 4 {
		fmt.Printf("Error: Date string in unexpected format: %s\n", dateText)
		return ""
	}

	// Extract the date range and split to get the start date
	dateRange := strings.TrimSpace(parts[3])
	dates := strings.Split(dateRange, "-")
	if len(dates) < 2 {
		fmt.Printf("Error: Unable to extract dates from date range: %s\n", dateRange)
		return ""
	}

	// The start date is the first part
	startDate := strings.TrimSpace(dates[0])
	fmt.Printf("Extracted start date: %s from week URL: %s\n", startDate, weekURL)
	return startDate
}

// extractTermIndex extracts the term index from the term URL.
func extractTermIndex(termURL string) int {
	parts := strings.Split(termURL, "/")
	lastPart := parts[len(parts)-1]
	termIndex := strings.TrimPrefix(lastPart, "index/")
	term, _ := strconv.Atoi(termIndex)
	return term
}
