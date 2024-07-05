package scraper

import (
	"fmt"
	"time"
)

// CalculateEventDate calculates the event date based on the lesson day and week offset.
func CalculateEventDate(currentTime time.Time, lessonDay string, weekOffset int) time.Time {
	loc, _ := time.LoadLocation("Europe/London")
	dayIndex := getDayIndex(lessonDay)
	currentWeekday := int(currentTime.In(loc).Weekday())
	if currentWeekday == 0 {
		currentWeekday = 7 // Adjust for Sunday to be the end of the week
	}

	// Calculate the date difference from the current date to the target lesson day
	daysUntilLesson := dayIndex - currentWeekday
	daysUntilLesson += weekOffset * 7

	eventDate := currentTime.AddDate(0, 0, daysUntilLesson)
	// Set eventDate to midnight of the calculated day in the specified timezone
	eventDate = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, loc)

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
	loc, _ := time.LoadLocation("Europe/London")
	startTime, err := time.ParseInLocation("15:04", startTimeStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing start time: %v", err)
	}
	endTime, err := time.ParseInLocation("15:04", endTimeStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing end time: %v", err)
	}

	startDateTime = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), startTime.Hour(), startTime.Minute(), 0, 0, loc)
	endDateTime = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), endTime.Hour(), endTime.Minute(), 0, 0, loc)

	return startDateTime, endDateTime, nil
}
