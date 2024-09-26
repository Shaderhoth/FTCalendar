package scraper

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

// login uses Playwright to handle the login process and returns the page after a successful login.
func login(browser playwright.Browser, username, password string) (playwright.Page, error) {
	loginURL := "https://funtech.co.uk/tutors"

	// Step 1: Create a new page
	fmt.Println("Attempting to login with Playwright...")

	page, err := browser.NewPage()
	if err != nil {
		fmt.Printf("Could not create a new page: %v\n", err)
		return nil, err
	}

	// Step 2: Navigate to the login page and wait for it to fully load
	if _, err = page.Goto(loginURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad, // Wait until the "load" event
	}); err != nil {
		fmt.Printf("Could not navigate to login page: %v\n", err)
		return nil, err
	}

	// Step 3: Fill the login form and submit it
	if err := page.Fill("input[name='data[Tutor][username]']", username); err != nil {
		fmt.Printf("Could not fill username: %v\n", err)
		return nil, err
	}
	if err := page.Fill("input[name='data[Tutor][password]']", password); err != nil {
		fmt.Printf("Could not fill password: %v\n", err)
		return nil, err
	}
	if err := page.Click("button[type='submit']"); err != nil {
		fmt.Printf("Could not submit login form: %v\n", err)
		return nil, err
	}

	// Return the page so it can be used for further actions after login
	return page, nil
}
