package scraper

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
)

// ScrapeAvailabilityWithClient scrapes the availability data using Playwright.
func ScrapeAvailabilityWithClient(browser playwright.Browser, username, password string) ([]Term, map[string][]Week, string) {
	loginURL := "https://funtech.co.uk/tutors"
	availabilityURL := "https://funtech.co.uk/tutor/tutor_available_times"

	// Step 1: Perform login using Playwright
	fmt.Println("Attempting to login with Playwright...")

	page, err := browser.NewPage()
	if err != nil {
		fmt.Printf("Could not create new page: %v\n", err)
		return nil, nil, ""
	}
	defer page.Close()

	// Navigate to the login page and wait for it to fully load
	if _, err = page.Goto(loginURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad, // Wait until the "load" event
	}); err != nil {
		fmt.Printf("Could not navigate to login page: %v\n", err)
		return nil, nil, ""
	}

	// Fill the login form and submit it
	if err := page.Fill("input[name='data[Tutor][username]']", username); err != nil {
		fmt.Printf("Could not fill username: %v\n", err)
		return nil, nil, ""
	}
	if err := page.Fill("input[name='data[Tutor][password]']", password); err != nil {
		fmt.Printf("Could not fill password: %v\n", err)
		return nil, nil, ""
	}
	if err := page.Click("button[type='submit']"); err != nil {
		fmt.Printf("Could not submit login form: %v\n", err)
		return nil, nil, ""
	}

	// Navigate to the availability page after login and wait for it to fully load
	if _, err = page.Goto(availabilityURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad, // Wait until the "load" event
	}); err != nil {
		fmt.Printf("Could not navigate to availability page: %v\n", err)
		return nil, nil, ""
	}

	// Scrape the availability data dynamically rendered via JavaScript
	availabilityHTML, err := page.Content()
	if err != nil {
		fmt.Printf("Could not get availability page content: %v\n", err)
		return nil, nil, ""
	}

	// Step 3: Parse the availability HTML to extract data
	fmt.Println("Parsing the availability page HTML content...")
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(availabilityHTML))
	if err != nil {
		fmt.Println("Error parsing availability HTML:", err)
		return nil, nil, ""
	}

	// Step 4: Extract the academic year
	year := extractYear(doc)
	fmt.Printf("Extracted Academic Year: %s\n", year)
	if year == "" {
		fmt.Println("Warning: Could not extract academic year. HTML may be malformed or incorrect.")
	}

	// Step 5: Extract terms and their links
	terms := extractTerms(doc)
	if len(terms) == 0 {
		fmt.Println("No terms found. HTML content may have changed, or there may be an issue with the scraping logic.")
	} else {
		fmt.Printf("Extracted %d terms:\n", len(terms))
		for _, term := range terms {
			fmt.Printf("Term Name: %s, URL: %s\n", term.Name, term.URL)
		}
	}

	// Step 6: Scrape weeks for each term
	weeksByTerm := map[string][]Week{}
	for _, term := range terms {
		var weeks []Week
		if term.Name == "Term Time" {
			fmt.Printf("Scraping weeks for Term Time: %s\n", term.Name)
			weeks = extractWeeksForTermTimePlaywright(page, term.URL, term.Name, year)
		} else {
			fmt.Printf("Scraping weeks for Term: %s\n", term.Name)
			weeks = extractWeeksForTermPlaywright(page, term.URL, term.Name, year)
		}

		if len(weeks) == 0 {
			fmt.Printf("Warning: No weeks found for Term: %s\n", term.Name)
		} else {
			fmt.Printf("Extracted %d weeks for Term: %s\n", len(weeks), term.Name)
			for _, week := range weeks {
				fmt.Printf("Week Number: %d, Start Date: %s, URL: %s\n", week.WeekNumber, week.StartDate, week.URL)
			}
		}
		weeksByTerm[term.Name] = weeks
	}

	fmt.Printf("Total Terms: %d, Total Weeks Collected: %d\n", len(terms), len(weeksByTerm))
	return terms, weeksByTerm, year
}
