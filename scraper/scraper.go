package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"google.golang.org/api/calendar/v3"
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

// AddLessonsToGoogleCalendar adds lessons directly to Google Calendar.
func AddLessonsToGoogleCalendar(service *calendar.Service, calendarID string, lessons []Lesson) error {
	// Get the start of the current week (Monday)
	currentDate := time.Now()
	weekStartDate := currentDate
	if currentDate.Weekday() != time.Monday {
		offset := int(time.Monday - currentDate.Weekday())
		if offset > 0 {
			offset = -6
		}
		weekStartDate = currentDate.AddDate(0, 0, offset)
	}

	fmt.Printf("Current Date: %v, Week Start Date: %v\n", currentDate, weekStartDate)

	events := make([]*calendar.Event, 0)
	for _, lesson := range lessons {
		// Calculate the event date based on the day of the week and the week offset
		dayIndex := getDayIndex(lesson.Day)
		eventDate := weekStartDate.AddDate(0, 0, dayIndex+(lesson.WeekOffset*7))

		startDateTime, endDateTime, err := getEventTimes(eventDate, lesson.StartTime, lesson.EndTime)
		if err != nil {
			fmt.Println("Error parsing event times:", err)
			continue
		}

		event := &calendar.Event{
			Summary: lesson.Course,
			Start: &calendar.EventDateTime{
				DateTime: startDateTime.Format(time.RFC3339),
				TimeZone: "Europe/London",
			},
			End: &calendar.EventDateTime{
				DateTime: endDateTime.Format(time.RFC3339),
				TimeZone: "Europe/London",
			},
			ColorId: getColorIDForLessonType(lesson.LessonType),
		}
		events = append(events, event)
	}

	// Add events to Google Calendar
	for _, event := range events {
		_, err := service.Events.Insert(calendarID, event).Do()
		if err != nil {
			fmt.Printf("Error creating event: %v\n", err)
			return err
		}
	}

	return nil
}

// getEventTimes parses and returns the start and end times for the event.
func getEventTimes(eventDate time.Time, startTimeStr, endTimeStr string) (startDateTime, endDateTime time.Time, err error) {
	startTime, err := time.Parse("15:04", startTimeStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing start time: %v", err)
	}
	endTime, err := time.Parse("15:04", endTimeStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing end time: %v", err)
	}

	startDateTime = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), startTime.Hour(), startTime.Minute(), 0, 0, time.Local)
	endDateTime = time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), endTime.Hour(), endTime.Minute(), 0, 0, time.Local)

	return startDateTime, endDateTime, nil
}

// getDayIndex returns the index of the day in the week.
func getDayIndex(day string) int {
	daysMapping := map[string]int{
		"Monday": 0, "Tuesday": 1, "Wednesday": 2,
		"Thursday": 3, "Friday": 4, "Saturday": 5, "Sunday": 6,
	}
	return daysMapping[day]
}

// stringToInt converts a string to an integer.
func stringToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// getColorIDForLessonType returns a color ID for the lesson type.
func getColorIDForLessonType(lessonType int) string {
	switch lessonType {
	case 1:
		return "2" // Green
	case 2:
		return "5" // Yellow
	case 3:
		return "11" // Red
	default:
		return "9" // Blue
	}
}
