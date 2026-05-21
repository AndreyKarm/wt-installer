package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	UserSkins string `json:"UserSkins"`
	Cookies   string `json:"Cookies"`
}

var (
	SkinPathInput string
	Cookies       string
)

const configName = "config.json"

func GetDefaultConfig() *Config {
	return &Config{
		UserSkins: "C:/Program Files (x86)/Steam/steamapps/common/War Thunder/UserSkins",
		Cookies:   "",
	}
}

func LoadConfig() (*Config, error) {
	if _, err := os.Stat(configName); os.IsNotExist(err) {
		cfg := GetDefaultConfig()
		SaveConfig(cfg)
		return cfg, nil
	}

	data, err := os.ReadFile(configName)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func SaveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configName, data, 0644)
}
