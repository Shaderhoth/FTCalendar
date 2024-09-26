package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ScrapeLessonsWithClient scrapes lessons for all weeks in the term or year using a provided HTTP client session.
func ScrapeLessonsWithClient(session *http.Client, username, password string, weeks []Week, year string) []Lesson {
	loginURL := "https://funtech.co.uk/tutors"
	baseURL := "https://funtech.co.uk/tutor/tutors/tt_week_schedule"

	if !login(session, loginURL, username, password) {
		fmt.Println("Login failed. Check your credentials and try again.")
		fmt.Println("Credentials: ", username, password)
		return nil
	}

	var allLessons []Lesson
	for _, week := range weeks {
		dataURL := fmt.Sprintf("%s/year:%s/term:%d/week:%d", baseURL, year, week.Term, week.WeekNumber)
		fmt.Printf("Accessing URL: %s\n", dataURL)
		lessonsForWeek := scrapeLessons(dataURL, session, week)
		fmt.Printf("Lessons retrieved from URL %s: %d\n", dataURL, len(lessonsForWeek))
		allLessons = append(allLessons, lessonsForWeek...)
	}

	// Log total number of lessons retrieved across all weeks
	fmt.Printf("Total lessons retrieved across all weeks: %d\n", len(allLessons))

	return allLessons
}

// scrapeLessons scrapes lessons from the provided URL for a specific week.
func scrapeLessons(dataURL string, session *http.Client, week Week) []Lesson {
	// Log the URL being accessed
	fmt.Printf("Accessing URL: %s\n", dataURL)

	resp, err := session.Get(dataURL)
	if err != nil {
		fmt.Printf("Error fetching data from URL %s: %v\n", dataURL, err)
		return nil
	}
	defer resp.Body.Close()

	// Log the HTTP response status
	fmt.Printf("HTTP Response Status from URL %s: %s\n", dataURL, resp.Status)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body from URL %s: %v\n", dataURL, err)
		return nil
	}

	// Log a portion of the raw HTML content for debugging
	fmt.Printf("Raw HTML content snippet from URL %s: %.200s\n", dataURL, string(body)) // First 200 characters

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		fmt.Printf("Error parsing HTML from URL %s: %v\n", dataURL, err)
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
