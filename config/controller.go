package config

import (
	"encoding/json"
	"os"
	"slices"
)

type Exclusion struct {
	Card          string `json:"card"`
	Flat          int    `json:"flat"`
	Comment       string `json:"comment"`
	TransactionID string `json:"transactionID"`
	Amount        int    `json:"amount"`
}

type Config struct {
	XToken     string      `json:"xToken"`
	JarName    string      `json:"jarName"`
	Exclusions []Exclusion `json:"exclusions"`
}

func SetConfig(path string, config Config) error {
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func GetConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func AddExclusion(path string, e Exclusion) error {
	config, err := GetConfig(path)
	if err != nil {
		return err
	}

	contains := slices.ContainsFunc(config.Exclusions, func(exclusion Exclusion) bool {
		if e.TransactionID == exclusion.TransactionID {
			return true
		}
		return false
	})

	if contains {
		return nil
	}

	config.Exclusions = append(config.Exclusions, e)
	return SetConfig(path, *config)
}
