package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Lesson represents a lesson schedule.
type Lesson struct {
	Course     string
	Day        string
	StartTime  string
	EndTime    string
	LessonType int
	WeekOffset int
}

// ScrapeLessons logs in to the website and scrapes lessons for the current and next week.
func ScrapeLessons(username, password string) []Lesson {
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
	dataURLCurrentWeek := fmt.Sprintf("%s/year:2023-24", baseURL)

	if !login(session, loginURL, username, password) {
		fmt.Println("Login failed. Check your credentials and try again.")
		return nil
	}

	lessonsCurrentWeek := scrapeLessons(dataURLCurrentWeek, session, 0)

	// Get the current week number
	resp, err := session.Get(dataURLCurrentWeek)
	if err != nil {
		fmt.Println("Error fetching current week data:", err)
		return nil
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("Error parsing current week HTML:", err)
		return nil
	}

	currentWeekNumber, _ := doc.Find("#TutorWeek option[selected]").Attr("value")
	nextWeekNumber := fmt.Sprintf("%d", stringToInt(currentWeekNumber)+1)
	dataURLNextWeek := fmt.Sprintf("%s/year:2023-24/term:3/week:%s", baseURL, nextWeekNumber)
	lessonsNextWeek := scrapeLessons(dataURLNextWeek, session, 1)

	return append(lessonsCurrentWeek, lessonsNextWeek...)
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

// scrapeLessons scrapes lessons from the provided URL.
func scrapeLessons(dataURL string, session *http.Client, weekOffset int) []Lesson {
	resp, err := session.Get(dataURL)
	if err != nil {
		fmt.Println("Error fetching data:", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		fmt.Println("Error parsing HTML:", err)
		return nil
	}

	var lessons []Lesson
	doc.Find("h4.panel-title").Each(func(i int, s *goquery.Selection) {
		lessonInfo := s.Find("span").Text()
		lessonInfo = strings.TrimSpace(lessonInfo)
		lessonParts := strings.Split(lessonInfo, " • ")
		if len(lessonParts) < 4 {
			return
		}
		course := lessonParts[0] + " " + lessonParts[1]
		day := convertToFullWeekday(strings.TrimSpace(lessonParts[2]))
		timeRange := strings.TrimSpace(lessonParts[3])
		startTime, endTime := timeRange, timeRange
		if strings.Contains(timeRange, "-") {
			times := strings.Split(timeRange, "-")
			startTime, endTime = strings.TrimSpace(times[0]), strings.TrimSpace(times[1])
		}

		lessonType := getLessonType(s)

		lesson := Lesson{
			Course:     course,
			Day:        day,
			StartTime:  startTime,
			EndTime:    endTime,
			LessonType: lessonType,
			WeekOffset: weekOffset,
		}
		lessons = append(lessons, lesson)
	})

	return lessons
}

// convertToFullWeekday converts abbreviated weekday to full name.
func convertToFullWeekday(abbreviatedDay string) string {
	daysMapping := map[string]string{
		"Mon": "Monday", "Tue": "Tuesday", "Wed": "Wednesday",
		"Thu": "Thursday", "Fri": "Friday", "Sat": "Saturday", "Sun": "Sunday",
	}
	return daysMapping[abbreviatedDay]
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

// stringToInt converts a string to an integer.
func stringToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
