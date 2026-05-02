package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/st0mb1e/bank-service-go/config"
	"github.com/st0mb1e/bank-service-go/handler/auth"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}
	router := mux.NewRouter()

	router.HandleFunc("/auth/register", auth.RegisterHandler).Methods("POST")
	router.HandleFunc("/auth/login", auth.LoginHandler).Methods("POST")

	appConfig := config.NewAppConfigFromEnv()
	log.Printf("Starting on %s", appConfig.Port)
	http.ListenAndServe(fmt.Sprintf(":%s", appConfig.Port), router)
}
