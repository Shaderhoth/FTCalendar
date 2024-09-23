package scraper

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

func AddLessonsToGoogleCalendar(service *calendar.Service, calendarID string, lessons []Lesson, clearAll bool) error {
	if clearAll {
		err := ClearCalendar(service, calendarID)
		if err != nil {
			return fmt.Errorf("error clearing Google Calendar: %v", err)
		}
	}

	// Fetch all existing events from Google Calendar
	existingEvents, err := GetAllEvents(service, calendarID)
	if err != nil {
		return fmt.Errorf("error fetching all events from Google Calendar: %v", err)
	}

	// Map to store existing Google Calendar events with their generated IDs
	existingEventsMap := make(map[string]*calendar.Event)
	for _, event := range existingEvents {
		if event == nil || event.Status == "cancelled" {
			continue
		}

		if event.Start == nil || event.End == nil {
			continue
		}

		// Generate an event ID for existing Google Calendar event
		eventID := generateEventID(event.Summary, event.Start.DateTime, event.End.DateTime)
		existingEventsMap[eventID] = event
		fmt.Printf("Existing Google Event: ID: %s, Summary: %s, Start: %s, End: %s\n", eventID, event.Summary, event.Start.DateTime, event.End.DateTime)
	}

	lessonsMap := make(map[string]*calendar.Event)

	fmt.Println("Lessons:")
	for _, lesson := range lessons {
		startDateTime, endDateTime, err := getEventTimes(lesson.Date, lesson.StartTime, lesson.EndTime)
		if err != nil {
			fmt.Printf("Error parsing event times: %v\n", err)
			continue
		}

		// Convert times to Europe/London timezone
		loc, _ := time.LoadLocation("Europe/London")
		start := startDateTime.In(loc)
		end := endDateTime.In(loc)

		fmt.Printf("Converted Start: %s, End: %s\n", start, end)

		// Ensure the end time is after the start time
		if !end.After(start) {
			end = start.Add(time.Hour) // Adjust end time to be one hour after start time
		}

		summary := lesson.Course
		startStr := start.Format(time.RFC3339)
		endStr := end.Format(time.RFC3339)
		eventID := generateEventID(summary, startStr, endStr)

		colorID := getColorIDForLessonType(lesson.LessonType)

		lessonsMap[eventID] = &calendar.Event{
			Summary: summary,
			Start: &calendar.EventDateTime{
				DateTime: startStr,
				TimeZone: "Europe/London",
			},
			End: &calendar.EventDateTime{
				DateTime: endStr,
				TimeZone: "Europe/London",
			},
			ColorId: colorID,
		}

		fmt.Printf("Lesson: ID: %s, Summary: %s, Start: %s, End: %s\n", eventID, summary, startStr, endStr)
	}

	// Delete events in Google Calendar that are not in the lessons data
	for eventID, existingEvent := range existingEventsMap {
		if _, found := lessonsMap[eventID]; !found {
			fmt.Printf("Deleting event '%s' (ID: %s)\n", existingEvent.Summary, eventID)
			err := service.Events.Delete(calendarID, existingEvent.Id).Do()
			if err != nil {
				return fmt.Errorf("error deleting event from Google Calendar: %v", err)
			}
		}
	}

	for eventID, gEvent := range lessonsMap {
		if existingEvent, found := existingEventsMap[eventID]; found {
			// Check if the event needs updating
			if existingEvent.Summary != gEvent.Summary || existingEvent.Start.DateTime != gEvent.Start.DateTime || existingEvent.End.DateTime != gEvent.End.DateTime {
				fmt.Printf("Updating event '%s' (ID: %s)\n", gEvent.Summary, eventID)
				_, err = service.Events.Update(calendarID, existingEvent.Id, gEvent).Do()
				if err != nil {
					return fmt.Errorf("error updating event in Google Calendar: %v", err)
				}
			}
		} else {
			// Insert new event if not found in existing events
			fmt.Printf("Inserting new event '%s' (ID: %s)\n", gEvent.Summary, eventID)
			_, err = service.Events.Insert(calendarID, gEvent).Do()
			if err != nil {
				return fmt.Errorf("error inserting event into Google Calendar: %v", err)
			}
		}
	}

	fmt.Println("Lessons successfully synced with Google Calendar.")
	return nil
}

func generateEventID(summary, start, end string) string {
	hash := md5.New()
	hash.Write([]byte(summary + start + end))
	return hex.EncodeToString(hash.Sum(nil))
}

func getColorIDForLessonType(lessonType int) string {
	switch lessonType {
	case 1:
		return "9" // Blue (darker)
	case 2:
		return "5" // Yellow
	case 3:
		return "11" // Red
	default:
		return "2" // Green (default)
	}
}

// ClearCalendar deletes all events from the specified Google Calendar.
func ClearCalendar(service *calendar.Service, calendarID string) error {
	pageToken := ""
	for {
		events, err := service.Events.List(calendarID).PageToken(pageToken).Do()
		if err != nil {
			return fmt.Errorf("error fetching events from Google Calendar: %v", err)
		}

		for _, event := range events.Items {
			if event == nil || event.Status == "cancelled" {
				continue
			}
			err = service.Events.Delete(calendarID, event.Id).Do()
			if err != nil {
				if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 410 {
					fmt.Printf("Event '%s' (ID: %s) already deleted from Google Calendar.\n", event.Summary, event.Id)
					continue
				}
				return fmt.Errorf("error deleting event from Google Calendar: %v", err)
			}
			fmt.Printf("Event '%s' (ID: %s) removed from Google Calendar.\n", event.Summary, event.Id)
		}

		pageToken = events.NextPageToken
		if pageToken == "" {
			break
		}
	}

	fmt.Println("All events cleared from Google Calendar.")
	return nil
}

// GetAllEvents retrieves all events from the specified Google Calendar.
func GetAllEvents(service *calendar.Service, calendarID string) ([]*calendar.Event, error) {
	var allEvents []*calendar.Event
	pageToken := ""
	for {
		events, err := service.Events.List(calendarID).PageToken(pageToken).Do()
		if err != nil {
			return nil, fmt.Errorf("error fetching events from Google Calendar: %v", err)
		}
		allEvents = append(allEvents, events.Items...)
		fmt.Printf("Fetched %d events from Google Calendar\n", len(events.Items))

		pageToken = events.NextPageToken
		if pageToken == "" {
			break
		}
	}
	fmt.Printf("Total events fetched: %d\n", len(allEvents))
	return allEvents, nil
}
