package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Username           string `json:"username"`
	Password           string `json:"password"`
	GithubToken        string `json:"github_token"`
	GithubRepo         string `json:"github_repo"`
	GithubPath         string `json:"github_path"`
	GoogleClientID     string `json:"google_client_id"`
	GoogleClientSecret string `json:"google_client_secret"`
	GoogleRedirectURI  string `json:"google_redirect_uri"`
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
