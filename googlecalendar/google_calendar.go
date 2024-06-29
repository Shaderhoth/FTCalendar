package googlecalendar

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"funtech-scraper/config"

	ics "github.com/arran4/golang-ical"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

var (
	oauthConfig *oauth2.Config
	authCode    string
)

func getClient(config *oauth2.Config) *http.Client {
	tokenFile := "token.json"
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
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
	return tok
}

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

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getConfig(cfg *config.Config) (*oauth2.Config, error) {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURI,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}, nil
}

func GetCalendarService(cfg *config.Config) (*calendar.Service, error) {
	oauthConfig, err := getConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret: %v", err)
	}

	client := getClient(oauthConfig)
	srv, err := calendar.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %v", err)
	}
	fmt.Println("Google Calendar client retrieved successfully.")
	return srv, nil
}

func AddICSEventsToCalendar(service *calendar.Service, calendarID, filename string) error {
	icsData, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading ICS file: %v", err)
	}

	cal, err := ics.ParseCalendar(strings.NewReader(string(icsData)))
	if err != nil {
		return fmt.Errorf("error parsing ICS data: %v", err)
	}

	for _, event := range cal.Events() {
		start, err := time.Parse("20060102T150405Z", event.GetProperty(ics.ComponentPropertyDtStart).Value)
		if err != nil {
			return fmt.Errorf("error parsing event start time: %v", err)
		}
		end, err := time.Parse("20060102T150405Z", event.GetProperty(ics.ComponentPropertyDtEnd).Value)
		if err != nil {
			return fmt.Errorf("error parsing event end time: %v", err)
		}

		// Ensure the end time is after the start time
		if !end.After(start) {
			end = start.Add(time.Hour) // Adjust end time to be one hour after start time
		}

		// Check for duplicate events
		existingEvents, err := service.Events.List(calendarID).
			TimeMin(start.Format(time.RFC3339)).
			TimeMax(end.Format(time.RFC3339)).
			Q(event.GetProperty(ics.ComponentPropertySummary).Value).
			Do()
		if err != nil {
			return fmt.Errorf("error listing events: %v", err)
		}

		if len(existingEvents.Items) > 0 {
			fmt.Printf("Duplicate event '%s' already exists. Skipping...\n", event.GetProperty(ics.ComponentPropertySummary).Value)
			continue
		}

		gEvent := &calendar.Event{
			Summary: event.GetProperty(ics.ComponentPropertySummary).Value,
			Start: &calendar.EventDateTime{
				DateTime: start.Format(time.RFC3339),
				TimeZone: "UTC",
			},
			End: &calendar.EventDateTime{
				DateTime: end.Format(time.RFC3339),
				TimeZone: "UTC",
			},
		}

		_, err = service.Events.Insert(calendarID, gEvent).Do()
		if err != nil {
			return fmt.Errorf("error inserting event into Google Calendar: %v", err)
		}
	}

	return nil
}
