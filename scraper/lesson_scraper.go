package scraper

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
)

// ScrapeLessonsWithClient scrapes lessons for all weeks in the term or year using Playwright.
func ScrapeLessonsWithClient(browser playwright.Browser, username, password string, weeks []Week, year string) []Lesson {
	baseURL := "https://funtech.co.uk/tutor/tutors/tt_week_schedule"

	// Perform login using the Playwright login function
	page, err := login(browser, username, password)
	if err != nil {
		fmt.Println("Login failed. Check your credentials and try again.")
		fmt.Println("Credentials: ", username, password)
		return nil
	}

	var allLessons []Lesson
	for _, week := range weeks {
		dataURL := fmt.Sprintf("%s/year:%s/term:%d/week:%d", baseURL, year, week.Term, week.WeekNumber)
		fmt.Printf("Accessing URL: %s\n", dataURL)
		lessonsForWeek := scrapeLessonsPlaywright(page, dataURL, week)
		fmt.Printf("Lessons retrieved from URL %s: %d\n", dataURL, len(lessonsForWeek))
		allLessons = append(allLessons, lessonsForWeek...)
	}

	// Log total number of lessons retrieved across all weeks
	fmt.Printf("Total lessons retrieved across all weeks: %d\n", len(allLessons))

	// Log out after scraping is complete
	logoutURL := "https://funtech.co.uk/tutor/tutors/logout"
	if _, err := page.Goto(logoutURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); err != nil {
		fmt.Printf("Error logging out: %v\n", err)
	}

	return allLessons
}

// scrapeLessonsPlaywright scrapes lessons from the provided URL using Playwright.
func scrapeLessonsPlaywright(page playwright.Page, dataURL string, week Week) []Lesson {
	// Navigate to the lesson page
	if _, err := page.Goto(dataURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad, // Wait until the "load" event
	}); err != nil {
		fmt.Printf("Error navigating to lessons page: %v\n", err)
		return nil
	}

	// Get the page content dynamically rendered via JavaScript
	pageHTML, err := page.Content()
	if err != nil {
		fmt.Printf("Error retrieving content for URL %s: %v\n", dataURL, err)
		return nil
	}

	// Parse the page HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		fmt.Printf("Error parsing HTML for URL %s: %v\n", dataURL, err)
		return nil
	}

	var lessons []Lesson
	doc.Find("h4.panel-title").Each(func(i int, s *goquery.Selection) {
		lessonInfo := s.Find("span").Text()
		lessonInfo = strings.TrimSpace(lessonInfo)

		// Log the raw lesson info being processed
		fmt.Printf("Processing raw lesson info: %s\n", lessonInfo)

		lessonParts := strings.Split(lessonInfo, " • ")
		if len(lessonParts) < 4 {
			// Log if a lesson is skipped due to incomplete data
			fmt.Printf("Skipping lesson due to incomplete data: %v\n", lessonParts)
			return
		}

		// Proceed to extract lesson details
		course := lessonParts[0] + " " + lessonParts[1]
		day := convertToFullWeekday(strings.TrimSpace(lessonParts[2]))
		timeRange := strings.TrimSpace(lessonParts[3])
		startTime, endTime := parseTimeRange(timeRange)

		lessonType := getLessonType(s)

		// Calculate lesson date based on the week start date
		lessonDate := calculateLessonDate(week, day)

		lesson := Lesson{
			Course:     course,
			Day:        day,
			StartTime:  startTime,
			EndTime:    endTime,
			Date:       lessonDate,
			LessonType: lessonType,
		}

		// Log each lesson's complete data
		fmt.Printf("Retrieved Lesson - Course: %s, Day: %s, Start: %s, End: %s, Date: %v, Lesson Type: %d\n",
			lesson.Course, lesson.Day, lesson.StartTime, lesson.EndTime, lesson.Date, lesson.LessonType)

		lessons = append(lessons, lesson)
	})

	// Log the total number of lessons retrieved from the URL
	fmt.Printf("Total lessons retrieved from URL %s: %d\n", dataURL, len(lessons))

	return lessons
}
