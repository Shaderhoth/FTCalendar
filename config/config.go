package config

import (
    "encoding/json"
    "os"
)

type Config struct {
    Username    string `json:"username"`
    Password    string `json:"password"`
    GithubToken string `json:"github_token"`
    GithubRepo  string `json:"github_repo"`
    GithubPath  string `json:"github_path"`
}

func LoadConfig(filename string) (*Config, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var cfg Config
    decoder := json.NewDecoder(file)
    err = decoder.Decode(&cfg)
    if err != nil {
        return nil, err
    }

    return &cfg, nil
}
