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

		err = googlecalendar.AddICSEventsToCalendar(service, "primary", "calendar.ics")
		if err != nil {
			fmt.Println("Error syncing ICS file with Google Calendar:", err)
		} else {
			fmt.Println("ICS file synced with Google Calendar successfully.")
		}

		// Wait for an hour before running again
		time.Sleep(1 * time.Hour)
	}
}
