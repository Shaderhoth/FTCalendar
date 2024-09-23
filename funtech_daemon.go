package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"funtech-scraper/config"
	"funtech-scraper/scraper"
)

func main() {
	clearAll := false // Set this to true if you want to clear the calendar first

	// Load common configuration
	commonCfg, err := config.LoadCommonConfig("config/common_config.json")
	if err != nil {
		log.Fatalf("Error loading common config: %v", err)
	}

	for {
		// Get the list of user config files
		userConfigFiles, err := filepath.Glob("config/user_configs/*.json")
		if err != nil {
			log.Fatalf("Error reading user config files: %v", err)
		}

		for _, userConfigFile := range userConfigFiles {
			// Load user configuration
			userCfg, err := config.LoadUserConfig(userConfigFile)
			if err != nil {
				fmt.Printf("Error loading user config (%s): %v\n", userConfigFile, err)
				continue
			}

			// Scrape availability to get weeks and year
			_, weeksByTerm, year := scraper.ScrapeAvailability(userCfg.Username, userCfg.Password)

			// Run the scraper to get lessons
			var allLessons []scraper.Lesson
			for _, weeks := range weeksByTerm {
				lessons := scraper.ScrapeLessons(userCfg.Username, userCfg.Password, weeks, year)
				allLessons = append(allLessons, lessons...)
			}

			// Retry logic for getting the Google Calendar service
			maxRetries := 3
			for retries := 0; retries < maxRetries; retries++ {
				// Get Google Calendar service
				service, err := scraper.GetCalendarService(commonCfg, userCfg, config.GetAuthCode, config.SaveUserConfig)
				if err != nil {
					fmt.Printf("Error getting Google Calendar service for user (%s), attempt %d: %v\n", userCfg.Username, retries+1, err)
					// Wait a bit before retrying in case it's an intermittent issue
					time.Sleep(5 * time.Second)
					continue
				}

				// Clear all events before syncing, if needed
				err = scraper.AddLessonsToGoogleCalendar(service, userCfg.GoogleCalendarID, allLessons, clearAll)
				if err != nil {
					fmt.Printf("Error syncing lessons with Google Calendar for user (%s), attempt %d: %v\n", userCfg.Username, retries+1, err)
					// Wait before retrying in case of transient errors
					time.Sleep(5 * time.Second)
					continue
				}

				// If successful, break out of the retry loop
				fmt.Printf("Lessons successfully synced with Google Calendar for user: %s on attempt %d\n", userCfg.Username, retries+1)
				break
			}
		}

		if !clearAll {
			time.Sleep(1 * time.Minute)
		}
		clearAll = false
	}
}
