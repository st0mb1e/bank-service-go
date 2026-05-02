package config

import (
	"fmt"
	"os"
)

type DbConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

func (c *DbConfig) GetDBUrl() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", c.User, c.Password, c.Host, c.Port, c.Name)
}

func NewDbConfigFromEnv() *DbConfig {
	fmt.Println(os.Getenv("DB_USER"))
	return &DbConfig{
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Name:     os.Getenv("DB_NAME"),
	}
}
