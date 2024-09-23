package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Lesson represents a lesson schedule.
type Lesson struct {
	Course     string
	Day        string
	StartTime  string
	EndTime    string
	Date       time.Time
	LessonType int
}

// login performs the login to the website.
func login(session *http.Client, loginURL, username, password string) bool {
	formData := url.Values{
		"_method":               {"POST"},
		"data[Tutor][username]": {username},
		"data[Tutor][password]": {password},
	}

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		fmt.Println("Error creating login request:", err)
		return false
	}

	// Add headers to mimic a real browser
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := session.Do(req)
	if err != nil {
		fmt.Println("Error logging in:", err)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading login response body:", err)
		return false
	}

	// Check if login was successful by looking for a specific element or text
	if strings.Contains(string(body), "Please sign in") {
		fmt.Println("Login unsuccessful. Please check your username and password.")
		return false
	}

	return true
}

// ScrapeLessons logs in to the website and scrapes lessons for all weeks in the term or year.
func ScrapeLessons(username, password string, weeks []Week, year string) []Lesson {
	jar, _ := cookiejar.New(nil)
	session := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.Header.Add("User-Agent", "Mozilla/5.0")
			return nil
		},
	}

	loginURL := "https://funtech.co.uk/tutors"
	baseURL := "https://funtech.co.uk/tutor/tutors/tt_week_schedule"

	if !login(session, loginURL, username, password) {
		fmt.Println("Login failed. Check your credentials and try again.")
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
		startTime, endTime := timeRange, timeRange
		if strings.Contains(timeRange, "-") {
			times := strings.Split(timeRange, "-")
			startTime, endTime = strings.TrimSpace(times[0]), strings.TrimSpace(times[1])
		}

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

func calculateLessonDate(week Week, day string) time.Time {
	// Parse the week start date (format: "02/01/2006")
	startDate, err := time.Parse("02/01/2006", week.StartDate)
	if err != nil {
		fmt.Printf("Error parsing start date for week %v: %v\n", week, err)
		return time.Time{} // Return zero value for time.Time if parsing fails
	}

	// Convert the lesson day into a time.Weekday
	lessonDay := map[string]time.Weekday{
		"Monday":    time.Monday,
		"Tuesday":   time.Tuesday,
		"Wednesday": time.Wednesday,
		"Thursday":  time.Thursday,
		"Friday":    time.Friday,
		"Saturday":  time.Saturday,
		"Sunday":    time.Sunday,
	}[day]

	// Find the difference in days between the start of the week and the lesson day.
	weekdayOffset := int(lessonDay - startDate.Weekday())

	// Add the offset to the start date to get the actual lesson date.
	lessonDate := startDate.AddDate(0, 0, weekdayOffset)

	fmt.Printf("Week Start Date: %v, Lesson Day: %s, Calculated Lesson Date: %v\n", startDate, day, lessonDate)

	return lessonDate
}

// convertToFullWeekday converts abbreviated weekday to full name.
func convertToFullWeekday(abbreviatedDay string) string {
	daysMapping := map[string]string{
		"Mon": "Monday", "Tue": "Tuesday", "Wed": "Wednesday",
		"Thu": "Thursday", "Fri": "Friday", "Sat": "Saturday", "Sun": "Sunday",
	}
	return daysMapping[abbreviatedDay]
}

// parseTimeRange parses a time range string like "09:00 - 10:00".
func parseTimeRange(timeRange string) (string, string) {
	times := strings.Split(timeRange, "-")
	startTime := strings.TrimSpace(times[0])
	endTime := strings.TrimSpace(times[1])
	return startTime, endTime
}

// getLessonType determines the lesson type based on the parent class.
func getLessonType(s *goquery.Selection) int {
	parentClass := s.Parent().Parent().AttrOr("class", "")
	switch {
	case strings.Contains(parentClass, "panel-info"):
		return 1
	case strings.Contains(parentClass, "panel-warning"):
		return 2
	case strings.Contains(parentClass, "panel-danger"):
		return 3
	default:
		return 0
	}
}
