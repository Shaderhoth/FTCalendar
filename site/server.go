package site

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"text/template"

	"funtech-scraper/config"
	"funtech-scraper/scraper"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

var (
	templates   = template.Must(template.ParseGlob("site/templates/*.html"))
	users       = make(map[string]*config.UserConfig)
	oauthConfig *oauth2.Config
	commonCfg   *config.CommonConfig
)

func InitOAuthConfig(cfg *config.CommonConfig) {
	commonCfg = cfg
	oauthConfig = &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURI,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}
}

func LoadUserConfigs() error {
	userConfigFiles, err := filepath.Glob("config/user_configs/*.json")
	if err != nil {
		return err
	}

	for _, userConfigFile := range userConfigFiles {
		userCfg, err := config.LoadUserConfig(userConfigFile)
		if err != nil {
			return fmt.Errorf("error loading user config (%s): %v", userConfigFile, err)
		}
		users[userCfg.Username] = userCfg
	}

	return nil
}

func AuthHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request on /auth from %s", r.RemoteAddr)
	username, err := r.Cookie("username")
	if err == nil && username != nil {
		if _, ok := users[username.Value]; ok {
			log.Printf("Redirecting to /dashboard for user: %s", username.Value)
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		} else {
			// Invalid cookie, delete it
			http.SetCookie(w, &http.Cookie{
				Name:   "username",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
			log.Printf("Invalid cookie found for user: %s, deleting cookie", username.Value)
		}
	}

	if r.Method == http.MethodPost {
		action := r.FormValue("action")
		username := r.FormValue("username")
		password := r.FormValue("password")

		if action == "login" {
			userCfg, ok := users[username]
			if !ok || userCfg.Password != password {
				log.Printf("Invalid login attempt for user: %s", username)
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}

			// Attempt to get Google Calendar service
			_, err := scraper.GetCalendarService(commonCfg, userCfg, config.GetAuthCode, config.SaveUserConfig)
			if err != nil {
				// Redirect to Google OAuth2 if authentication is needed
				authURL, needsAuth := scraper.NeedsGoogleAuth(userCfg, commonCfg)
				if needsAuth {
					log.Printf("User needs Google Auth, redirecting to: %s", authURL)
					http.Redirect(w, r, authURL, http.StatusSeeOther)
					return
				}

				log.Printf("Error getting Google Calendar service for user: %s, error: %v", username, err)
				http.Error(w, fmt.Sprintf("Error getting Google Calendar service for user: %s", username), http.StatusInternalServerError)
				return
			}

			log.Printf("Successful login for user: %s", username)
			http.SetCookie(w, &http.Cookie{
				Name:  "username",
				Value: username,
				Path:  "/",
			})

			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		} else if action == "register" {
			if _, exists := users[username]; exists {
				log.Printf("Attempt to register existing user: %s", username)
				http.Error(w, "User already exists", http.StatusBadRequest)
				return
			}

			userCfg := &config.UserConfig{
				Username: username,
				Password: password,
			}
			users[username] = userCfg
			config.SaveUserConfig(username, userCfg)

			log.Printf("New user registered: %s", username)
			http.SetCookie(w, &http.Cookie{
				Name:  "username",
				Value: username,
				Path:  "/",
			})

			// Redirect to Google OAuth2 for authorization
			authURL, _ := scraper.NeedsGoogleAuth(userCfg, commonCfg)
			log.Printf("Redirecting new user %s to Google OAuth2: %s", username, authURL)
			http.Redirect(w, r, authURL, http.StatusSeeOther)
		}
		return
	}

	templates.ExecuteTemplate(w, "auth.html", nil)
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request on /dashboard from %s", r.RemoteAddr)
	username, err := r.Cookie("username")
	if err != nil {
		log.Printf("No username cookie found, redirecting to /auth")
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	userCfg, ok := users[username.Value]
	if !ok {
		log.Printf("User not found in configs: %s, redirecting to /auth", username.Value)
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	message := r.URL.Query().Get("message")
	data := struct {
		Message          string
		Username         string
		GoogleCalendarID string
		Password         string
		Calendars        []*calendar.CalendarListEntry
	}{
		Message:          message,
		Username:         userCfg.Username,
		GoogleCalendarID: userCfg.GoogleCalendarID,
		Password:         userCfg.Password,
	}

	if r.Method == http.MethodPost {
		userCfg.GoogleCalendarID = r.FormValue("google_calendar_id")
		userCfg.Username = r.FormValue("username")
		userCfg.Password = r.FormValue("password")
		config.SaveUserConfig(username.Value, userCfg)

		log.Printf("User config saved for user: %s", username.Value)
		// Check if Google Auth is needed and redirect if so
		if authURL, needsAuth := scraper.NeedsGoogleAuth(userCfg, commonCfg); needsAuth {
			log.Printf("User needs Google Auth, redirecting to: %s", authURL)
			http.Redirect(w, r, authURL, http.StatusSeeOther)
			return
		}

		message = "Config saved successfully for user: " + username.Value
		http.Redirect(w, r, "/dashboard?message="+url.QueryEscape(message), http.StatusSeeOther)
		return
	}

	// Retrieve the list of calendars
	service, err := scraper.GetCalendarService(commonCfg, userCfg, config.GetAuthCode, config.SaveUserConfig)
	if err != nil {
		log.Printf("Error getting Google Calendar service for user (%s): %v\n", userCfg.Username, err)
		if authURL, needsAuth := scraper.NeedsGoogleAuth(userCfg, commonCfg); needsAuth {
			log.Printf("User needs Google Auth, redirecting to: %s", authURL)
			http.Redirect(w, r, authURL, http.StatusSeeOther)
			return
		}
		http.Error(w, fmt.Sprintf("Error getting Google Calendar service for user: %s", userCfg.Username), http.StatusInternalServerError)
		return
	}
	calendars, err := scraper.GetUserCalendars(service)
	if err != nil {

		if authURL, needsAuth := scraper.NeedsGoogleAuth(userCfg, commonCfg); needsAuth {
			log.Printf("User needs Google Auth, redirecting to: %s", authURL)
			http.Redirect(w, r, authURL, http.StatusSeeOther)
			return
		}
		log.Printf("Error retrieving calendars for user (%s): %v\n", userCfg.Username, err)
		http.Error(w, fmt.Sprintf("Error retrieving calendars for user (%s): %v", userCfg.Username, err), http.StatusInternalServerError)
		return
	}
	data.Calendars = calendars

	templates.ExecuteTemplate(w, "dashboard.html", data)
}

func AuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request on /auth_callback from %s", r.RemoteAddr)
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	config.SetAuthCode(state, code)

	userCfg, exists := users[state]
	if exists {
		_, err := scraper.GetCalendarService(commonCfg, userCfg, config.GetAuthCode, config.SaveUserConfig)
		if err == nil {
			log.Printf("Authorization completed for user: %s", state)
			http.Redirect(w, r, "/dashboard?message=Authorization completed. You can close this window.", http.StatusSeeOther)
		} else {
			log.Printf("Authorization failed for user: %s, error: %v", state, err)
			http.Redirect(w, r, "/dashboard?message=Authorization failed. Please try again.", http.StatusSeeOther)
		}
	} else {
		log.Printf("User not found for state: %s", state)
		http.Error(w, "User not found", http.StatusBadRequest)
	}
}

func HomeRedirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/dashboard", http.StatusMovedPermanently)
}
