package scraper

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
)

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
		termURL, exists := s.Attr("href")
		if exists {
			terms = append(terms, Term{Name: strings.TrimSpace(termName), URL: "https://funtech.co.uk" + termURL})
		} else {
			fmt.Printf("Warning: No URL found for Term: %s\n", termName)
		}
	})
	return terms
}

// extractWeeksForTermPlaywright extracts weeks for a given term using Playwright
func extractWeeksForTermPlaywright(page playwright.Page, termURL string, termName string, year string) []Week {
	_, err := page.Goto(termURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	})
	if err != nil {
		fmt.Printf("Could not navigate to term page: %v\n", err)
		return nil
	}

	termHTML, err := page.Content()
	if err != nil {
		fmt.Printf("Could not get term page content: %v\n", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(termHTML))
	if err != nil {
		fmt.Println("Error parsing term HTML:", err)
		return nil
	}

	var weeks []Week
	doc.Find("table tbody tr").Each(func(rowIndex int, row *goquery.Selection) {
		// Get the week start date from the header
		row.Find("th").Each(func(colIndex int, col *goquery.Selection) {
			startDate := strings.TrimSpace(col.Text())

			// Process each "View" link for this week's availability
			row.Find(".dropdown-menu li a").Each(func(linkIndex int, link *goquery.Selection) {
				viewLink, exists := link.Attr("href")
				if exists && strings.Contains(viewLink, "/tutor/tutor_available_times/availability/") {
					weekNumber := linkIndex + 1
					week := Week{
						WeekNumber: weekNumber,
						StartDate:  startDate,
						URL:        "https://funtech.co.uk" + viewLink,
					}
					weeks = append(weeks, week)
					fmt.Printf("Extracted Week %d: %s (URL: %s)\n", weekNumber, startDate, week.URL)
				} else {
					fmt.Printf("Warning: No 'View' link found for Week %d in Term: %s\n", linkIndex+1, termName)
				}
			})
		})
	})

	return weeks
}

// extractWeeksForTermTime uses Playwright to extract the weeks for the "Term Time" schedule.
func extractWeeksForTermTimePlaywright(page playwright.Page, termURL string, termName string, year string) []Week {
	fmt.Printf("Fetching term page for Term Time: %s from URL: %s\n", termName, termURL)

	// Navigate to the term's availability page
	if _, err := page.Goto(termURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		fmt.Printf("Could not navigate to term page: %v\n", err)
		return nil
	}

	// Scrape the term page content after the JavaScript has loaded
	termHTML, err := page.Content()
	if err != nil {
		fmt.Printf("Could not get term page content: %v\n", err)
		return nil
	}

	// Parse the HTML content for weeks
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(termHTML))
	if err != nil {
		fmt.Println("Error parsing term page HTML:", err)
		return nil
	}

	var weeks []Week
	termIndex := extractTermIndex(termURL)

	// Iterate over each week's availability section in the term's table
	doc.Find("table tbody tr").Each(func(rowIndex int, row *goquery.Selection) {
		row.Find("td.text-center").Each(func(colIndex int, col *goquery.Selection) {
			viewLink := col.Find(".dropdown-menu li a").FilterFunction(func(_ int, s *goquery.Selection) bool {
				return strings.Contains(s.Text(), "View")
			}).AttrOr("href", "")

			if viewLink != "" {
				weekURL := fmt.Sprintf("https://funtech.co.uk%s", viewLink)
				startDate := fetchWeekDatesPlaywright(page, weekURL)

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
