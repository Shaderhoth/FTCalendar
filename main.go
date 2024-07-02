package main

import (
	"fmt"
	"time"

	"funtech-scraper/config"
	"funtech-scraper/googlecalendar"
	"funtech-scraper/scraper"
	"funtech-scraper/uploader"
)

func main() {
	clearAll := false // Set this to true if you want to clear the calendar first
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

		// Generate ICS file
		scraper.GenerateICSFile(lessons, "calendar.ics")

		// Upload ICS file to GitHub
		err = uploader.UploadToGitHub(cfg.GithubToken, cfg.GithubRepo, cfg.GithubPath, "calendar.ics")
		if err != nil {
			fmt.Println("Error uploading to GitHub:", err)
		}

		// Sync ICS file with Google Calendar
		service, err := googlecalendar.GetCalendarService(cfg)
		if err != nil {
			fmt.Println("Error getting Google Calendar service:", err)
			return
		}

		// Clear all events before syncing, if needed
		err = googlecalendar.AddICSEventsToCalendar(service, cfg.GoogleCalendarID, "calendar.ics", clearAll)
		if err != nil {
			fmt.Println("Error syncing ICS file with Google Calendar:", err)
		} else {
			fmt.Println("ICS file synced with Google Calendar successfully.")
		}
		if !clearAll {
			time.Sleep(1 * time.Minute)
		}
		clearAll = false
	}
}
