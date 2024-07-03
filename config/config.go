package config

import (
	"encoding/json"
	"os"
)

var AuthCode string

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
		return nil, err
	}

	return config, nil
}

func SaveUserConfig(username string, config *UserConfig) error {
	filePath := "config/user_configs/" + username + ".json"
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(config)
}
