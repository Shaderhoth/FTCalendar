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
	clearAll := true // Set this to true if you want to clear the calendar first

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

			// Run the scraper
			lessons := scraper.ScrapeLessons(userCfg.Username, userCfg.Password)
			if len(lessons) == 0 {
				fmt.Printf("No lessons found for user: %s\n", userCfg.Username)
				continue
			}

			// Get Google Calendar service
			service, err := scraper.GetCalendarService(commonCfg, userCfg)
			if err != nil {
				fmt.Printf("Error getting Google Calendar service for user (%s): %v\n", userCfg.Username, err)
				continue
			}

			// Clear all events before syncing, if needed
			err = scraper.AddLessonsToGoogleCalendar(service, userCfg.GoogleCalendarID, lessons, clearAll)
			if err != nil {
				fmt.Printf("Error syncing lessons with Google Calendar for user (%s): %v\n", userCfg.Username, err)
			} else {
				fmt.Printf("Lessons successfully synced with Google Calendar for user: %s\n", userCfg.Username)
			}
		}

		if !clearAll {
			time.Sleep(10 * time.Minute)
		}
		clearAll = false
	}
}
