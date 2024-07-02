package googlecalendar

import (
	"context"
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

func getConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURI,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}
}

func GetCalendarService(cfg *config.Config) (*calendar.Service, error) {
	oauthConfig = getConfig(cfg)
	client := getClient(oauthConfig)
	srv, err := calendar.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %v", err)
	}
	fmt.Println("Google Calendar client retrieved successfully.")
	return srv, nil
}

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
