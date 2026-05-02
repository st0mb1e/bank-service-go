package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"github.com/st0mb1e/bank-service-go/config"
	"github.com/st0mb1e/bank-service-go/dao/repo"
	"github.com/st0mb1e/bank-service-go/handler/auth"
	"github.com/st0mb1e/bank-service-go/service"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}

	dbConfig := config.NewDbConfigFromEnv()
	connectionString := dbConfig.GetDBUrl()

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("sql open: %v", err)
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("db ping (is Postgres running and DATABASE/DB_* correct?): %v\nconnection string: %s", err, connectionString)
	}

	// DI
	userRepo := repo.NewUserRepo(db)
	authService := service.NewAuthService(userRepo)

	authHandlers := auth.NewAuthHandlers(authService)

	router := mux.NewRouter()
	router.HandleFunc("/auth/register", authHandlers.Register).Methods("POST")
	router.HandleFunc("/auth/login", authHandlers.Login).Methods("POST")

	appConfig := config.NewAppConfigFromEnv()
	log.Printf("Starting on %s", appConfig.Port)
	http.ListenAndServe(fmt.Sprintf(":%s", appConfig.Port), router)
}
