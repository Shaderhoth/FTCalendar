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
	authCodes = make(map[string]string)
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

func GetAuthCode(username string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	code, exists := authCodes[username]
	return code, exists
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	username, err := r.Cookie("username")
	if err == nil && username != nil {
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
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:  "username",
				Value: username,
				Path:  "/",
			})

			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		} else if action == "register" {
			if _, exists := users[username]; exists {
				http.Error(w, "User already exists", http.StatusBadRequest)
				return
			}

			userCfg := &config.UserConfig{
				Username: username,
				Password: password,
			}
			users[username] = userCfg
			saveUserConfig(username, userCfg)

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
	username, err := r.Cookie("username")
	if err != nil {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	userCfg, ok := users[username.Value]
	if !ok {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		userCfg.GoogleCalendarID = r.FormValue("google_calendar_id")
		userCfg.Username = r.FormValue("username")
		userCfg.Password = r.FormValue("password")
		saveUserConfig(username.Value, userCfg)

		fmt.Fprintf(w, "Config saved successfully for user: %s", username.Value)
		return
	}

	templates.ExecuteTemplate(w, "dashboard.html", userCfg)
}

func authCallbackHandler(w http.ResponseWriter, r *http.Request) {
	//state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	username, err := r.Cookie("username")
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	mu.Lock()
	authCodes[username.Value] = code
	mu.Unlock()

	fmt.Fprintf(w, "Authorization completed. You can close this window.")
}

func StartServer(port string) {
	// Load user configurations
	if err := loadUserConfigs(); err != nil {
		log.Fatalf("Error loading user configs: %v", err)
	}

	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/dashboard", dashboardHandler)
	http.HandleFunc("/auth_callback", authCallbackHandler)

	certFile := "cert.pem"
	keyFile := "key.pem"

	server := &http.Server{
		Addr: ":" + port,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	log.Printf("Starting server on https://localhost:%s\n", port)
	log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
}
