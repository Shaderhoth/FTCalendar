package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"funtech-scraper/config"
	"funtech-scraper/scraper"

	"github.com/playwright-community/playwright-go"
)

const availabilityCheckInterval = 24 * time.Hour // Interval for checking availability
const maxAvailabilityRetries = 3                 // Maximum retries for availability scraping

func main() {
	clearAll := false // Set this to true if you want to clear the calendar first

	// Load common configuration
	commonCfg, err := config.LoadCommonConfig("config/common_config.json")
	if err != nil {
		log.Fatalf("Error loading common config: %v", err)
	}

	// Variables to store shared availability data
	var sharedWeeksByTerm map[string][]scraper.Week
	var sharedYear string
	lastAvailabilityRun := time.Time{} // Track the last time availability was run

	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Couldn't start Playwright: %v", err)
	}
	defer pw.Stop()

	// Set up the browser (shared across all users)
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), // Set to false if you want to see the browser in action
	})
	if err != nil {
		log.Fatalf("Could not launch browser: %v", err)
	}
	defer browser.Close()

	for {
		// Get the list of user config files
		userConfigFiles, err := filepath.Glob("config/user_configs/*.json")
		if err != nil {
			log.Fatalf("Error reading user config files: %v", err)
		}

		// Run ScrapeAvailability once for all users, if needed (e.g., every 24 hours)
		if sharedWeeksByTerm == nil || sharedYear == "" || time.Since(lastAvailabilityRun) >= availabilityCheckInterval {
			fmt.Println("Running ScrapeAvailability to fetch availability data for all users...")

			// Retry logic for ScrapeAvailability
			for retry := 0; retry < maxAvailabilityRetries; retry++ {
				if len(userConfigFiles) > 0 {
					userCfg, err := config.LoadUserConfig(userConfigFiles[0])
					if err != nil {
						log.Fatalf("Error loading user config for scraping availability: %v", err)
					}

					// Scrape availability data using Playwright
					_, weeksByTerm, year := scraper.ScrapeAvailabilityWithClient(browser, userCfg.Username, userCfg.Password)

					if weeksByTerm != nil && year != "" {
						sharedWeeksByTerm = weeksByTerm
						sharedYear = year
						lastAvailabilityRun = time.Now()
						fmt.Println("ScrapeAvailability completed. Data shared across all users.")
						break
					} else {
						fmt.Printf("Availability scraping failed on attempt %d. Retrying...\n", retry+1)
					}

				} else {
					fmt.Println("No user configurations found. Cannot scrape availability.")
					return
				}
				// Wait before retrying if the last attempt failed
				time.Sleep(5 * time.Second)
			}

			// If after retries, still no availability data
			if sharedWeeksByTerm == nil || sharedYear == "" {
				fmt.Println("ScrapeAvailability failed after all retries. Sleeping.")
				time.Sleep(10 * time.Minute)
			}
		}

		// Run ScrapeLessons for each user individually, using the shared availability data
		for _, userConfigFile := range userConfigFiles {
			// Load user configuration
			userCfg, err := config.LoadUserConfig(userConfigFile)
			if err != nil {
				fmt.Printf("Error loading user config (%s): %v\n", userConfigFile, err)
				continue
			}

			// Ensure we have shared availability data before proceeding
			if sharedWeeksByTerm == nil || sharedYear == "" {
				fmt.Println("No availability data available. Skipping lesson scraping.")
				continue
			}

			// Run the scraper to get lessons for the current user
			var allLessons []scraper.Lesson
			for _, weeks := range sharedWeeksByTerm {
				// Using the shared browser for scraping lessons
				lessons := scraper.ScrapeLessonsWithClient(browser, userCfg.Username, userCfg.Password, weeks, sharedYear)
				allLessons = append(allLessons, lessons...)
			}

			// Sync with Google Calendar
			// Retry logic for getting the Google Calendar service
			maxRetries := 3
			for retries := 0; retries < maxRetries; retries++ {
				service, err := scraper.GetCalendarService(commonCfg, userCfg, config.GetAuthCode, config.SaveUserConfig)
				if err != nil {
					fmt.Printf("Error getting Google Calendar service for user (%s), attempt %d: %v\n", userCfg.Username, retries+1, err)
					time.Sleep(5 * time.Second)
					continue
				}

				err = scraper.AddLessonsToGoogleCalendar(service, userCfg.GoogleCalendarID, allLessons, clearAll)
				if err != nil {
					fmt.Printf("Error syncing lessons with Google Calendar for user (%s), attempt %d: %v\n", userCfg.Username, retries+1, err)
					time.Sleep(5 * time.Second)
					continue
				}

				fmt.Printf("Lessons successfully synced with Google Calendar for user: %s on attempt %d\n", userCfg.Username, retries+1)
				break
			}
		}

		// Wait before the next iteration
		if !clearAll {
			time.Sleep(10 * time.Minute)
		}
		clearAll = false
	}
}
