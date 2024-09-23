package scraper

import (
	"fmt"
	"time"
)

// CalculateEventDate calculates the event date based on the lesson day and the week start date.
func CalculateEventDate(weekStartDate time.Time, lessonDay string) time.Time {
	loc, _ := time.LoadLocation("Europe/London")
	dayIndex := getDayIndex(lessonDay)
	weekStartWeekday := int(weekStartDate.In(loc).Weekday())
	if weekStartWeekday == 0 {
		weekStartWeekday = 7 // Adjust for Sunday to be the end of the week
	}

	// Calculate the date difference from the start of the week to the target lesson day
	daysUntilLesson := dayIndex - weekStartWeekday

	// Add days to the week start date to get the exact lesson date
	eventDate := weekStartDate.AddDate(0, 0, daysUntilLesson)
	// Set eventDate to midnight of the calculated day in the specified timezone
	eventDate = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, loc)

	fmt.Printf("Calculating event date for lesson on %s: Week start day index: %d, Target day index: %d, Days until lesson: %d, Event date: %s\n", lessonDay, weekStartWeekday, dayIndex, daysUntilLesson, eventDate)

	return eventDate
}

// getDayIndex maps a weekday name to its corresponding index.
func getDayIndex(day string) int {
	daysMapping := map[string]int{
		"Monday": 1, "Tuesday": 2, "Wednesday": 3,
		"Thursday": 4, "Friday": 5, "Saturday": 6, "Sunday": 7,
	}
	return daysMapping[day]
}

// getEventTimes calculates the start and end times for a lesson based on the event date.
func getEventTimes(eventDate time.Time, startTimeStr, endTimeStr string) (startDateTime, endDateTime time.Time, err error) {
	loc, _ := time.LoadLocation("Europe/London")
	startTime, err := time.ParseInLocation("15:04", startTimeStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing start time: %v", err)
	}
	endTime, err := time.ParseInLocation("15:04", endTimeStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing end time: %v", err)
	}

	// Calculate the start and end datetime by combining the lesson time with the event date
	startDateTime = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), startTime.Hour(), startTime.Minute(), 0, 0, loc)
	endDateTime = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), endTime.Hour(), endTime.Minute(), 0, 0, loc)

	return startDateTime, endDateTime, nil
}
