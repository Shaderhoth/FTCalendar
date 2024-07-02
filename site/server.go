package site

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"funtech-scraper/config"
)

var (
	templates = template.Must(template.ParseGlob("site/templates/*.html"))
	mu        sync.Mutex
	users     = make(map[string]*config.UserConfig)
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

func saveUserConfig(username string, userCfg *config.UserConfig) error {
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

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		userCfg, ok := users[username]
		if !ok || userCfg.Password != password {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		http.Redirect(w, r, "/config?username="+username, http.StatusSeeOther)
		return
	}

	templates.ExecuteTemplate(w, "login.html", nil)
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	userCfg, ok := users[username]
	if !ok {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodPost {
		userCfg.GoogleCalendarID = r.FormValue("google_calendar_id")
		userCfg.AccessToken = r.FormValue("access_token")
		userCfg.RefreshToken = r.FormValue("refresh_token")

		if err := saveUserConfig(username, userCfg); err != nil {
			http.Error(w, "Error saving config", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Config saved successfully for user: %s", username)
		return
	}

	templates.ExecuteTemplate(w, "config.html", userCfg)
}

func StartServer() {
	// Load user configurations
	if err := loadUserConfigs(); err != nil {
		log.Fatalf("Error loading user configs: %v", err)
	}

	// HTTP handlers
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/config", configHandler)

	// Load SSL certificates
	certFile := "cert.pem"
	keyFile := "key.pem"

	// Set up HTTPS server
	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	log.Println("Starting server on https://localhost")
	log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
}
