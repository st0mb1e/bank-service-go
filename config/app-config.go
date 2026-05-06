package config

import (
	"os"

	"github.com/sirupsen/logrus"
)

type AppConfig struct {
	Port      string
	LogLevel  string
	LogrusLvl logrus.Level
}

func NewAppConfigFromEnv() *AppConfig {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	lvl := os.Getenv("LOG_LEVEL")
	if lvl == "" {
		lvl = "info"
	}
	parsed, err := logrus.ParseLevel(lvl)
	if err != nil {
		parsed = logrus.InfoLevel
	}
	return &AppConfig{Port: port, LogLevel: lvl, LogrusLvl: parsed}
}
