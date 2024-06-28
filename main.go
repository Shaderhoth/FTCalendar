package main

import (
    "funtech-scraper/config"
    "funtech-scraper/scraper"
    "funtech-scraper/uploader"
    "fmt"
    "time"
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

        // Wait for an hour before running again
        time.Sleep(1 * time.Hour)
    }
}
