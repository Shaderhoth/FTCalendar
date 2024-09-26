package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Handles the login process and returns whether it was successful
func login(session *http.Client, loginURL, username, password string) bool {
	// Step 1: Fetch the login page with appropriate headers to get necessary cookies
	req, err := http.NewRequest("GET", loginURL, nil)
	if err != nil {
		fmt.Println("Error creating GET request:", err)
		return false
	}

	// Add headers to mimic a real browser (necessary to prevent 403)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Upgrade-Insecure-Requests", "1")
	req.Header.Add("Referer", loginURL)

	// Send GET request
	resp, err := session.Do(req)
	if err != nil {
		fmt.Println("Error fetching login page:", err)
		return false
	}
	defer resp.Body.Close()

	// Log the initial response status
	fmt.Printf("Initial GET request returned status: %s\n", resp.Status)

	// Read the body of the response for debugging (optional)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return false
	}
	body := string(bodyBytes)
	fmt.Printf("Initial GET response HTML content: %.200s\n", body) // Log the first 200 characters

	// Check cookies after fetching the login page
	loginURLParsed, _ := url.Parse(loginURL)
	cookies := session.Jar.Cookies(loginURLParsed)
	fmt.Printf("Cookies after initial GET request: %v\n", cookies)

	if len(cookies) == 0 {
		fmt.Println("No cookies were set during the initial GET request. Aborting.")
		return false
	}

	// Step 2: Prepare the login form data
	formData := url.Values{
		"_method":               {"POST"},
		"data[Tutor][username]": {username},
		"data[Tutor][password]": {password},
	}

	// Step 3: Create a POST request to log in
	req, err = http.NewRequest("POST", loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		fmt.Println("Error creating login POST request:", err)
		return false
	}

	// Add headers to mimic a real browser
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Upgrade-Insecure-Requests", "1")
	req.Header.Add("Referer", loginURL)

	// Step 4: Send the POST request to log in
	time.Sleep(time.Second * 2) // Add a delay to simulate human behavior
	resp, err = session.Do(req)
	if err != nil {
		fmt.Println("Error logging in:", err)
		return false
	}
	defer resp.Body.Close()

	// Step 5: Log the status and check cookies after login to ensure the session is established
	fmt.Printf("Login POST request returned status: %s\n", resp.Status)
	loginBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading login response body:", err)
		return false
	}

	// Log the cookies after the POST request (successful login)
	cookies = session.Jar.Cookies(loginURLParsed)
	fmt.Printf("Cookies after login POST request: %v\n", cookies)

	// Log the first 200 characters of the login response for debugging
	fmt.Printf("Login response HTML content: %.200s\n", string(loginBody))

	// Check if login was successful by looking for a specific element or text in the response
	if strings.Contains(string(loginBody), "Please sign in") {
		fmt.Println("Login unsuccessful. Please check your username and password.")
		return false
	}

	fmt.Println("Login successful.")
	return true
}
