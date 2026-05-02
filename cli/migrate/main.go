package main

import (
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"github.com/st0mb1e/bank-service-go/config"
)

func main() {
	_ = godotenv.Load()

	dbConfig := config.NewDbConfigFromEnv()

	m, err := migrate.New("file://migrations", dbConfig.GetDBUrl())
	if err != nil {
		log.Fatalf("Failed to migrate due to error: %v", err)
		return
	}
	if err := m.Up(); err != nil {
		log.Fatalf("Failed to migrate up due to error: %v", err)
		return
	}
	log.Println("Migration completed successfully")
}
