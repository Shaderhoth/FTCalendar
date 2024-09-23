package scraper

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Week represents a specific week within a term and its date range.
type Week struct {
	Term       int
	WeekNumber int
	StartDate  string
	URL        string
}

// Term represents a term in the availability (e.g., Term Time, Summer, Easter, Xmas).
type Term struct {
	Name string
	URL  string
}

func ScrapeAvailability(username, password string) ([]Term, map[string][]Week, string) {
	jar, _ := cookiejar.New(nil)
	session := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.Header.Add("User-Agent", "Mozilla/5.0")
			return nil
		},
	}

	loginURL := "https://funtech.co.uk/tutors"
	availabilityURL := "https://funtech.co.uk/tutor/tutor_available_times"

	// Perform login
	if !login(session, loginURL, username, password) {
		fmt.Println("Login failed. Check your credentials and try again.")
		return nil, nil, ""
	}

	// Get the availability page
	resp, err := session.Get(availabilityURL)
	if err != nil {
		fmt.Println("Error fetching availability page:", err)
		return nil, nil, ""
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("Error parsing availability HTML:", err)
		return nil, nil, ""
	}

	// Extract the academic year
	year := extractYear(doc)
	fmt.Printf("Extracted Academic Year: %s\n", year)

	// Extract terms and their links
	terms := extractTerms(doc)
	fmt.Printf("Extracted Terms:\n")
	for _, term := range terms {
		fmt.Printf("Term Name: %s, URL: %s\n", term.Name, term.URL)
	}

	// Scrape weeks for each term
	weeksByTerm := map[string][]Week{}
	for _, term := range terms {
		var weeks []Week
		if term.Name == "Term Time" {
			// Use the Term Time-specific function
			weeks = extractWeeksForTermTime(session, term.URL, term.Name, year)
		} else {
			// Use the generic function for other terms
			weeks = extractWeeksForTerm(session, term.URL, term.Name, year)
		}

		fmt.Printf("Weeks for Term: %s\n", term.Name)
		for _, week := range weeks {
			fmt.Printf("Week Number: %d, Start Date: %s, URL: %s\n", week.WeekNumber, week.StartDate, week.URL)
		}
		weeksByTerm[term.Name] = weeks
	}

	fmt.Printf("Total Terms: %d, Total Weeks Collected: %d\n", len(terms), len(weeksByTerm))
	return terms, weeksByTerm, year
}

// extractYear extracts the academic year from the availability page.
func extractYear(doc *goquery.Document) string {
	year := doc.Find("h1.no-margin-top small").Text()
	year = strings.TrimSpace(year)
	year = strings.Replace(year, "Year ", "", 1)
	return year
}

// extractTerms extracts all available terms from the availability page.
func extractTerms(doc *goquery.Document) []Term {
	var terms []Term
	doc.Find("ul.nav-tabs li a").Each(func(i int, s *goquery.Selection) {
		termName := s.Text()
		termURL, _ := s.Attr("href")
		terms = append(terms, Term{Name: strings.TrimSpace(termName), URL: "https://funtech.co.uk" + termURL})
	})
	return terms
}

func extractWeeksForTerm(session *http.Client, termURL string, termName string, year string) []Week {
	resp, err := session.Get(termURL)
	if err != nil {
		fmt.Println("Error fetching term page:", err)
		return nil
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("Error parsing term page HTML:", err)
		return nil
	}

	var weeks []Week
	termIndex := extractTermIndex(termURL)

	// Iterate over each week's availability section in the term's table
	doc.Find("table tbody tr").Each(func(i int, row *goquery.Selection) {
		// Find the "View" link for the week
		viewLink := row.Find("td .dropdown-menu li a").FilterFunction(func(_ int, s *goquery.Selection) bool {
			return strings.Contains(s.Text(), "View")
		}).AttrOr("href", "")

		if viewLink != "" {
			// Construct the full URL to access the detailed "View" page for the week
			weekURL := fmt.Sprintf("https://funtech.co.uk%s", viewLink)
			startDate := fetchWeekDates(session, weekURL)

			if startDate != "" {
				weekNumber := i + 1
				week := Week{
					Term:       termIndex,
					WeekNumber: weekNumber,
					StartDate:  startDate,
					URL:        fmt.Sprintf("https://funtech.co.uk/tutor/tutors/tt_week_schedule/year:%s/term:%d/week:%d", year, termIndex, weekNumber),
				}
				weeks = append(weeks, week)
			}
		}
	})

	return weeks
}

func extractWeeksForTermTime(session *http.Client, termURL string, termName string, year string) []Week {
	resp, err := session.Get(termURL)
	if err != nil {
		fmt.Printf("Error fetching term page %s: %v\n", termURL, err)
		return nil
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Printf("Error parsing term page HTML for %s: %v\n", termURL, err)
		return nil
	}

	var weeks []Week
	termIndex := extractTermIndex(termURL)

	doc.Find("table tbody tr").Each(func(rowIndex int, row *goquery.Selection) {
		row.Find("td.text-center").Each(func(colIndex int, col *goquery.Selection) {
			viewLink := col.Find(".dropdown-menu li a").FilterFunction(func(_ int, s *goquery.Selection) bool {
				return strings.Contains(s.Text(), "View")
			}).AttrOr("href", "")

			if viewLink != "" {
				weekURL := fmt.Sprintf("https://funtech.co.uk%s", viewLink)
				startDate := fetchWeekDates(session, weekURL)

				if startDate != "" {
					weekNumber := colIndex + 1
					week := Week{
						Term:       termIndex,
						WeekNumber: weekNumber,
						StartDate:  startDate,
						URL:        fmt.Sprintf("https://funtech.co.uk/tutor/tutors/tt_week_schedule/year:%s/term:%d/week:%d", year, termIndex, weekNumber),
					}
					weeks = append(weeks, week)
					fmt.Printf("Week %d - Start Date: %s, View URL: %s\n", weekNumber, startDate, weekURL)
				} else {
					fmt.Printf("Week %d - No valid start date found at URL: %s\n", colIndex+1, weekURL)
				}
			} else {
				fmt.Printf("Week %d - No 'View' link found in column %d\n", rowIndex+1, colIndex+1)
			}
		})
	})

	fmt.Printf("Extracted %d weeks for Term: %s\n", len(weeks), termName)
	return weeks
}

// fetchWeekDates retrieves the week start date from the 'View' page of a specific week.
func fetchWeekDates(session *http.Client, weekURL string) string {
	resp, err := session.Get(weekURL)
	if err != nil {
		fmt.Printf("Error fetching week page %s: %v\n", weekURL, err)
		return ""
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Printf("Error parsing week page HTML for %s: %v\n", weekURL, err)
		return ""
	}

	// Find the paragraph element that contains the week dates
	dateText := doc.Find(".page-header p").Text()
	fmt.Printf("Raw date text from week page %s: %s\n", weekURL, dateText)

	// Example expected format: "Year 2024-25 | Term 1 | Week 1 | 23/09/2024 - 29/09/2024"
	parts := strings.Split(dateText, "|")
	if len(parts) < 4 {
		fmt.Printf("Error: Date string in unexpected format: %s\n", dateText)
		return ""
	}

	// Extract the date range and split to get the start date
	dateRange := strings.TrimSpace(parts[3])
	dates := strings.Split(dateRange, "-")
	if len(dates) < 2 {
		fmt.Printf("Error: Unable to extract dates from date range: %s\n", dateRange)
		return ""
	}

	// The start date is the first part
	startDate := strings.TrimSpace(dates[0])
	fmt.Printf("Extracted start date: %s from week URL: %s\n", startDate, weekURL)
	return startDate
}

// extractTermIndex extracts the term index from the term URL.
func extractTermIndex(termURL string) int {
	parts := strings.Split(termURL, "/")
	lastPart := parts[len(parts)-1]
	termIndex := strings.TrimPrefix(lastPart, "index/")
	term, _ := strconv.Atoi(termIndex)
	return term
}
