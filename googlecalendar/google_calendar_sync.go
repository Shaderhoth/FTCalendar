package googlecalendar

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"google.golang.org/api/calendar/v3"
)

func AddICSEventsToCalendar(service *calendar.Service, calendarID, filename string, clearAll bool) error {
	if clearAll {
		err := ClearCalendar(service, calendarID)
		if err != nil {
			return fmt.Errorf("error clearing Google Calendar: %v", err)
		}
	}

	icsData, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading ICS file: %v", err)
	}

	cal, err := ics.ParseCalendar(strings.NewReader(string(icsData)))
	if err != nil {
		return fmt.Errorf("error parsing ICS data: %v", err)
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

	icsEventsMap := make(map[string]*calendar.Event)

	fmt.Println("ICS Events:")
	for _, event := range cal.Events() {
		if event == nil {
			continue
		}
		startProperty := event.GetProperty(ics.ComponentPropertyDtStart)
		endProperty := event.GetProperty(ics.ComponentPropertyDtEnd)
		if startProperty == nil || endProperty == nil {
			continue
		}
		start, err := time.Parse("20060102T150405Z", startProperty.Value)
		if err != nil {
			fmt.Printf("error parsing event start time: %v\n", err)
			continue
		}
		end, err := time.Parse("20060102T150405Z", endProperty.Value)
		if err != nil {
			fmt.Printf("error parsing event end time: %v\n", err)
			continue
		}

		// Convert times to Europe/London timezone
		loc, _ := time.LoadLocation("Europe/London")
		start = start.In(loc)
		end = end.In(loc)

		fmt.Printf("Converted Start: %s, End: %s\n", start, end)

		// Ensure the end time is after the start time
		if !end.After(start) {
			end = start.Add(time.Hour) // Adjust end time to be one hour after start time
		}

		summary := event.GetProperty(ics.ComponentPropertySummary).Value
		startStr := start.Format(time.RFC3339)
		endStr := end.Format(time.RFC3339)
		eventID := generateEventID(summary, startStr, endStr)

		icsEventsMap[eventID] = &calendar.Event{
			Summary: summary,
			Start: &calendar.EventDateTime{
				DateTime: startStr,
				TimeZone: "Europe/London",
			},
			End: &calendar.EventDateTime{
				DateTime: endStr,
				TimeZone: "Europe/London",
			},
		}

		fmt.Printf("ICS Event: ID: %s, Summary: %s, Start: %s, End: %s\n", eventID, summary, startStr, endStr)
	}

	// Delete events in Google Calendar that are not in the ICS file
	for eventID, existingEvent := range existingEventsMap {
		if _, found := icsEventsMap[eventID]; !found {
			fmt.Printf("Deleting event '%s' (ID: %s)\n", existingEvent.Summary, eventID)
			err := service.Events.Delete(calendarID, existingEvent.Id).Do()
			if err != nil {
				return fmt.Errorf("error deleting event from Google Calendar: %v", err)
			}
		}
	}

	for eventID, gEvent := range icsEventsMap {
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

	fmt.Println("ICS file synced with Google Calendar successfully.")
	return nil
}

func generateEventID(summary, start, end string) string {
	hash := md5.New()
	hash.Write([]byte(summary + start + end))
	return hex.EncodeToString(hash.Sum(nil))
}
