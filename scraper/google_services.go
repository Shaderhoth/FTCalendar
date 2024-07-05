package scraper

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"funtech-scraper/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

var (
	oauthConfig *oauth2.Config
)

func getClient(oauth2Config *oauth2.Config, userCfg *config.UserConfig, getAuthCode func(string) (string, bool), saveUserConfig func(string, *config.UserConfig) error) (*http.Client, error) {
	token := &oauth2.Token{
		AccessToken:  userCfg.AccessToken,
		TokenType:    userCfg.TokenType,
		RefreshToken: userCfg.RefreshToken,
	}
	token.Expiry, _ = time.Parse(time.RFC3339, userCfg.Expiry)

	if token.Valid() {
		log.Printf("Token is still valid for user: %s", userCfg.Username)
		return oauth2Config.Client(context.Background(), token), nil
	}

	tokSource := oauth2Config.TokenSource(context.Background(), token)
	newToken, err := tokSource.Token()
	if err != nil || !newToken.Valid() {
		log.Printf("Token invalid for user: %s, requesting new token...", userCfg.Username)
		code, ok := getAuthCode(userCfg.Username)
		if !ok {
			log.Printf("Authorization code not found for user: %s", userCfg.Username)
			return nil, fmt.Errorf("authorization code not found for user: %s", userCfg.Username)
		}

		newToken, err = oauth2Config.Exchange(context.Background(), code)
		if err != nil {
			log.Printf("Unable to retrieve token from web for user: %s, error: %v", userCfg.Username, err)
			return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
		}

		log.Printf("New token retrieved for user: %s", userCfg.Username)
		log.Printf("AccessToken: %s", newToken.AccessToken)
		log.Printf("TokenType: %s", newToken.TokenType)
		log.Printf("RefreshToken: %s", newToken.RefreshToken)
		log.Printf("Expiry: %s", newToken.Expiry.Format(time.RFC3339))

		userCfg.AccessToken = newToken.AccessToken
		userCfg.TokenType = newToken.TokenType
		userCfg.RefreshToken = newToken.RefreshToken
		userCfg.Expiry = newToken.Expiry.Format(time.RFC3339)

		if err := saveUserConfig(userCfg.Username, userCfg); err != nil {
			log.Printf("Unable to save user config for user: %s, error: %v", userCfg.Username, err)
			return nil, fmt.Errorf("unable to save user config: %v", err)
		}
		log.Printf("User config saved successfully for user: %s", userCfg.Username)
	} else {
		log.Printf("Token refreshed for user: %s", userCfg.Username)
	}

	return oauth2Config.Client(context.Background(), newToken), nil
}

func getConfig(cfg *config.CommonConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURI,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}
}

func GetCalendarService(commonCfg *config.CommonConfig, userCfg *config.UserConfig, getAuthCode func(string) (string, bool), saveUserConfig func(string, *config.UserConfig) error) (*calendar.Service, error) {
	if oauthConfig == nil {
		oauthConfig = getConfig(commonCfg)
	}
	client, err := getClient(oauthConfig, userCfg, getAuthCode, saveUserConfig)
	if err != nil {
		return nil, fmt.Errorf("authorization failed for user: %s", userCfg.Username)
	}
	srv, err := calendar.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %v", err)
	}
	fmt.Println("Google Calendar client retrieved successfully.")
	return srv, nil
}

func NeedsGoogleAuth(userCfg *config.UserConfig, commonCfg *config.CommonConfig) (string, bool) {
	if oauthConfig == nil {
		oauthConfig = getConfig(commonCfg)
	}

	expiry, err := time.Parse(time.RFC3339, userCfg.Expiry)
	if err != nil || userCfg.AccessToken == "" || expiry.Before(time.Now()) {
		authURL := oauthConfig.AuthCodeURL(userCfg.Username, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
		return authURL, true
	}
	return "", false
}

func GetUserCalendars(service *calendar.Service) ([]*calendar.CalendarListEntry, error) {
	calendarList, err := service.CalendarList.List().Do()
	if err != nil {
		return nil, err
	}
	return calendarList.Items, nil
}
