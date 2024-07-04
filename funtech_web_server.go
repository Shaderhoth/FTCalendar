package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"funtech-scraper/config"
	"funtech-scraper/scraper"
)

var (
	templates = template.Must(template.ParseGlob("site/templates/*.html"))
	mu        sync.Mutex
	users     = make(map[string]*config.UserConfig)
	authCodes = make(map[string]string)
	commonCfg *config.CommonConfig
)

func loadUserConfigs() error {
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

func SaveUserConfig(username string, userCfg *config.UserConfig) error {
	mu.Lock()
	defer mu.Unlock()

	filePath := fmt.Sprintf("config/user_configs/%s.json", username)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(userCfg)
}

func GetAuthCode(username string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	code, exists := authCodes[username]
	return code, exists
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request on /auth from %s", r.RemoteAddr)
	username, err := r.Cookie("username")
	if err == nil && username != nil {
		log.Printf("Redirecting to /dashboard for user: %s", username.Value)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
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
			SaveUserConfig(username, userCfg)

			log.Printf("New user registered: %s", username)
			http.SetCookie(w, &http.Cookie{
				Name:  "username",
				Value: username,
				Path:  "/",
			})

			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		}
		return
	}

	templates.ExecuteTemplate(w, "auth.html", nil)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
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

	if r.Method == http.MethodPost {
		userCfg.GoogleCalendarID = r.FormValue("google_calendar_id")
		userCfg.Username = r.FormValue("username")
		userCfg.Password = r.FormValue("password")
		SaveUserConfig(username.Value, userCfg)

		log.Printf("User config saved for user: %s", username.Value)
		// Check if Google Auth is needed and redirect if so
		if authURL, needsAuth := scraper.NeedsGoogleAuth(userCfg); needsAuth {
			log.Printf("User needs Google Auth, redirecting to: %s", authURL)
			http.Redirect(w, r, authURL, http.StatusSeeOther)
			return
		}

		fmt.Fprintf(w, "Config saved successfully for user: %s", username.Value)
		return
	}

	templates.ExecuteTemplate(w, "dashboard.html", userCfg)
}

func authCallbackHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request on /auth_callback from %s", r.RemoteAddr)
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	mu.Lock()
	authCodes[state] = code
	mu.Unlock()

	userCfg, exists := users[state]
	if exists {
		_, err := scraper.GetCalendarService(commonCfg, userCfg, GetAuthCode, SaveUserConfig)
		if err == nil {
			fmt.Fprintf(w, "Authorization completed. You can close this window.")
		} else {
			log.Printf("Authorization failed for user: %s, error: %v", state, err)
			fmt.Fprintf(w, "Authorization failed. Please try again.")
		}
	} else {
		log.Printf("User not found for state: %s", state)
		http.Error(w, "User not found", http.StatusBadRequest)
	}
}

func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	log.Printf("Redirecting HTTP request from %s to HTTPS", r.RemoteAddr)
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}

func StartServer(httpPort, httpsPort string) {
	// Load common configuration
	var err error
	commonCfg, err = config.LoadCommonConfig("config/common_config.json")
	if err != nil {
		log.Fatalf("Error loading common config: %v", err)
	}

	// Load user configurations
	if err := loadUserConfigs(); err != nil {
		log.Fatalf("Error loading user configs: %v", err)
	}

	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/dashboard", dashboardHandler)
	http.HandleFunc("/auth_callback", authCallbackHandler)

	certFile := "cert.pem"
	keyFile := "key.pem"

	// HTTPS server
	go func() {
		httpsServer := &http.Server{
			Addr: ":" + httpsPort,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		log.Printf("Starting HTTPS server on https://localhost:%s\n", httpsPort)
		log.Fatal(httpsServer.ListenAndServeTLS(certFile, keyFile))
	}()

	// HTTP server
	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: http.HandlerFunc(redirectToHTTPS),
	}
	log.Printf("Starting HTTP server on http://localhost:%s\n", httpPort)
	log.Fatal(httpServer.ListenAndServe())
}

func main() {
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8000" // Default HTTP port if not specified
	}

	httpsPort := os.Getenv("HTTPS_PORT")
	if httpsPort == "" {
		httpsPort = "8100" // Default HTTPS port if not specified
	}

	StartServer(httpPort, httpsPort)
}
