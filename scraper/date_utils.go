package scraper

import (
	"fmt"
	"time"
)

// CalculateEventDate calculates the event date based on the lesson day and week offset.
func CalculateEventDate(currentTime time.Time, lessonDay string, weekOffset int) time.Time {
	dayIndex := getDayIndex(lessonDay)
	currentWeekday := int(currentTime.Weekday())
	if currentWeekday == 0 {
		currentWeekday = 7 // Adjust for Sunday to be the end of the week
	}

	// Calculate the date difference from the current date to the target lesson day
	daysUntilLesson := dayIndex - currentWeekday
	daysUntilLesson += weekOffset * 7

	eventDate := currentTime.AddDate(0, 0, daysUntilLesson)
	// Set eventDate to midnight of the calculated day
	eventDate = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, eventDate.Location())

	fmt.Printf("Calculating event date for lesson on %s: Current day index: %d, Target day index: %d, Days until lesson: %d, Event date: %s\n", lessonDay, currentWeekday, dayIndex, daysUntilLesson, eventDate)

	return eventDate
}

func getDayIndex(day string) int {
	daysMapping := map[string]int{
		"Monday": 1, "Tuesday": 2, "Wednesday": 3,
		"Thursday": 4, "Friday": 5, "Saturday": 6, "Sunday": 7,
	}
	return daysMapping[day]
}

func getEventTimes(eventDate time.Time, startTimeStr, endTimeStr string) (startDateTime, endDateTime time.Time, err error) {
	startTime, err := time.Parse("15:04", startTimeStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing start time: %v", err)
	}
	endTime, err := time.Parse("15:04", endTimeStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing end time: %v", err)
	}

	startDateTime = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), startTime.Hour(), startTime.Minute(), 0, 0, time.Local)
	endDateTime = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), endTime.Hour(), endTime.Minute(), 0, 0, time.Local)

	return startDateTime, endDateTime, nil
}
