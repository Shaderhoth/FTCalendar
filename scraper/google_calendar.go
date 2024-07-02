package scraper

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"funtech-scraper/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

var (
	oauthConfig *oauth2.Config
	authCode    string
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
		// Calculate the event date based on the day of the week and the week offset
		dayIndex := getDayIndex(lesson.Day)
		eventDate := time.Now().AddDate(0, 0, dayIndex+(lesson.WeekOffset*7)-1) // Correct the day offset

		startDateTime, endDateTime, err := getEventTimes(eventDate, lesson.StartTime, lesson.EndTime)
		if err != nil {
			fmt.Printf("error parsing event times: %v\n", err)
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

func getDayIndex(day string) int {
	daysMapping := map[string]int{
		"Monday": 0, "Tuesday": 1, "Wednesday": 2,
		"Thursday": 3, "Friday": 4, "Saturday": 5, "Sunday": 6,
	}
	return daysMapping[day]
}

// getClient retrieves a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, userCfg *config.UserConfig) *http.Client {
	tokenFile := fmt.Sprintf("config/user_configs/%s_token.json", userCfg.Username)
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok, err = getTokenFromConfig(userCfg)
		if err != nil {
			tok = getTokenFromWeb(config, userCfg)
		}
		saveToken(tokenFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromConfig(userCfg *config.UserConfig) (*oauth2.Token, error) {
	if userCfg.AccessToken == "" || userCfg.RefreshToken == "" {
		return nil, fmt.Errorf("no token found in config")
	}
	expiry, err := time.Parse(time.RFC3339, userCfg.Expiry)
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken:  userCfg.AccessToken,
		TokenType:    userCfg.TokenType,
		RefreshToken: userCfg.RefreshToken,
		Expiry:       expiry,
	}, nil
}

// getTokenFromWeb requests a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config, userCfg *config.UserConfig) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		authCode = r.URL.Query().Get("code")
		fmt.Fprintf(w, "Authorization completed. You can close this window.")
	})

	server := &http.Server{Addr: ":8080"}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	for authCode == "" {
		time.Sleep(time.Second)
	}

	server.Shutdown(context.Background())

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}

	// Save the token details in user config
	userCfg.AccessToken = tok.AccessToken
	userCfg.TokenType = tok.TokenType
	userCfg.RefreshToken = tok.RefreshToken
	userCfg.Expiry = tok.Expiry.Format(time.RFC3339)

	if err := saveUserConfig(userCfg.Username, userCfg); err != nil {
		log.Fatalf("Unable to save user config: %v", err)
	}

	return tok
}

// tokenFromFile retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// getConfig constructs the OAuth2 configuration.
func getConfig(commonCfg *config.CommonConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     commonCfg.GoogleClientID,
		ClientSecret: commonCfg.GoogleClientSecret,
		RedirectURL:  commonCfg.GoogleRedirectURI,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}
}

// GetCalendarService returns a Google Calendar service
func GetCalendarService(commonCfg *config.CommonConfig, userCfg *config.UserConfig) (*calendar.Service, error) {
	oauthConfig = getConfig(commonCfg)
	client := getClient(oauthConfig, userCfg)
	srv, err := calendar.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %v", err)
	}
	fmt.Println("Google Calendar client retrieved successfully.")
	return srv, nil
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
