package main

import (
	"fmt"
	"time"

	"funtech-scraper/config"
	"funtech-scraper/googlecalendar"
	"funtech-scraper/scraper"
)

func main() {
	clearAll := true // Set this to true if you want to clear the calendar first
	for {
		// Load configuration
		cfg, err := config.LoadConfig("config.json")
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		// Run the scraper
		lessons := scraper.ScrapeLessons(cfg.Username, cfg.Password)
		if len(lessons) == 0 {
			fmt.Println("No lessons found.")
			return
		}

		// Sync lessons with Google Calendar
		service, err := googlecalendar.GetCalendarService(cfg)
		if err != nil {
			fmt.Println("Error getting Google Calendar service:", err)
			return
		}

		// Clear all events before syncing, if needed
		err = googlecalendar.AddLessonsToGoogleCalendar(service, cfg.GoogleCalendarID, lessons, clearAll)
		if err != nil {
			fmt.Println("Error syncing lessons with Google Calendar:", err)
		} else {
			fmt.Println("Lessons synced with Google Calendar successfully.")
		}
		if !clearAll {
			time.Sleep(10 * time.Minute)
		}
		clearAll = false
	}
}
