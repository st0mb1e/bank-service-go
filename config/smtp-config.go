package config

import (
	"os"
	"strconv"
)

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	Enabled  bool
}

func NewSMTPConfigFromEnv() *SMTPConfig {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}
	host := os.Getenv("SMTP_HOST")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = user
	}
	return &SMTPConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
		From:     from,
		Enabled:  host != "" && user != "" && pass != "",
	}
}
