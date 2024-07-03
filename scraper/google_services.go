package scraper

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"funtech-scraper/config"
	"funtech-scraper/site"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

var (
	oauthConfig *oauth2.Config
)

func getClient(oauth2Config *oauth2.Config, userCfg *config.UserConfig) (*http.Client, error) {
	token := &oauth2.Token{
		AccessToken:  userCfg.AccessToken,
		TokenType:    userCfg.TokenType,
		RefreshToken: userCfg.RefreshToken,
	}
	token.Expiry, _ = time.Parse(time.RFC3339, userCfg.Expiry)

	if token.Valid() {
		return oauth2Config.Client(context.Background(), token), nil
	}

	tokSource := oauth2Config.TokenSource(context.Background(), token)
	newToken, err := tokSource.Token()
	if err != nil {
		fmt.Printf("Token invalid for user: %s, requesting new token...\n", userCfg.Username)
		code, ok := site.GetAuthCode(userCfg.Username)
		if !ok {
			return nil, fmt.Errorf("authorization code not found for user: %s", userCfg.Username)
		}

		newToken, err = oauth2Config.Exchange(context.Background(), code)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
		}

		userCfg.AccessToken = newToken.AccessToken
		userCfg.TokenType = newToken.TokenType
		userCfg.RefreshToken = newToken.RefreshToken
		userCfg.Expiry = newToken.Expiry.Format(time.RFC3339)

		if err := config.SaveUserConfig(userCfg.Username, userCfg); err != nil {
			return nil, fmt.Errorf("unable to save user config: %v", err)
		}
	}

	return oauth2Config.Client(context.Background(), newToken), nil
}

// getConfig constructs the OAuth2 configuration.
func getConfig(cfg *config.CommonConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURI,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}
}

// GetCalendarService returns a Google Calendar service
func GetCalendarService(commonCfg *config.CommonConfig, userCfg *config.UserConfig) (*calendar.Service, error) {
	oauthConfig = getConfig(commonCfg)
	client, err := getClient(oauthConfig, userCfg)
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
