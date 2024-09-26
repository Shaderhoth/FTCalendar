package scraper

import "time"

// Term represents a term in the availability (e.g., Term Time, Summer, Easter, Xmas).
type Term struct {
	Name string
	URL  string
}

// Week represents a specific week within a term and its date range.
type Week struct {
	Term       int
	WeekNumber int
	StartDate  string
	URL        string
}

// Lesson represents a lesson schedule.
type Lesson struct {
	Course     string
	Day        string
	StartTime  string
	EndTime    string
	Date       time.Time
	LessonType int
}
