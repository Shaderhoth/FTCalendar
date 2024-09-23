package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var (
	AuthCode  string
	mu        sync.Mutex
	authCodes = make(map[string]string)
)

type CommonConfig struct {
	GoogleClientID     string `json:"google_client_id"`
	GoogleClientSecret string `json:"google_client_secret"`
	GoogleRedirectURI  string `json:"google_redirect_uri"`
}

type UserConfig struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	GoogleCalendarID string `json:"google_calendar_id"`
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	RefreshToken     string `json:"refresh_token"`
	Expiry           string `json:"expiry"`
}

// LoadCommonConfig loads common configuration from a file
func LoadCommonConfig(filename string) (*CommonConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &CommonConfig{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// LoadUserConfig loads the user configuration from a file
func LoadUserConfig(filename string) (*UserConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &UserConfig{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, fmt.Errorf("error decoding user config for %s: %v", filename, err)
	}

	return config, nil
}

// SaveUserConfig atomically saves the user configuration to a file
func SaveUserConfig(username string, config *UserConfig) error {
	mu.Lock()
	defer mu.Unlock()

	// Write to a temporary file to avoid incomplete writes
	tmpFilePath := "config/user_configs/" + username + ".json.tmp"
	file, err := os.Create(tmpFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp config file for %s: %v", username, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Optional: Makes the JSON more readable
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("error encoding user config for %s: %v", username, err)
	}

	// Rename the temp file to the final file path
	finalFilePath := "config/user_configs/" + username + ".json"
	if err := os.Rename(tmpFilePath, finalFilePath); err != nil {
		return fmt.Errorf("failed to rename temp config file for %s: %v", username, err)
	}

	return nil
}

// GetAuthCode retrieves the stored auth code for a given username
func GetAuthCode(username string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	code, exists := authCodes[username]
	return code, exists
}

// SetAuthCode sets an auth code for a user
func SetAuthCode(username, code string) {
	mu.Lock()
	defer mu.Unlock()
	authCodes[username] = code
}
