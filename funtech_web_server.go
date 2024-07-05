package main

import (
	"log"
	"net/http"
	"os"

	"funtech-scraper/config"
	"funtech-scraper/site"
)

func StartServer(httpPort string) {
	// Load common configuration
	var err error
	commonCfg, err := config.LoadCommonConfig("config/common_config.json")
	if err != nil {
		log.Fatalf("Error loading common config: %v", err)
	}

	// Initialize OAuth configuration
	site.InitOAuthConfig(commonCfg)

	// Load user configurations
	if err := site.LoadUserConfigs(); err != nil {
		log.Fatalf("Error loading user configs: %v", err)
	}

	http.HandleFunc("/", site.HomeRedirectHandler)
	http.HandleFunc("/auth", site.AuthHandler)
	http.HandleFunc("/dashboard", site.DashboardHandler)
	http.HandleFunc("/auth_callback", site.AuthCallbackHandler)

	fs := http.FileServer(http.Dir("site/templates"))
	http.Handle("/site/templates/", http.StripPrefix("/site/templates/", fs))

	log.Printf("Starting HTTP server on http://localhost:%s\n", httpPort)
	log.Fatal(http.ListenAndServe(":"+httpPort, nil))
}

func main() {
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8100" // Default HTTP port if not specified
	}

	StartServer(httpPort)
}
