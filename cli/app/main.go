package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/st0mb1e/bank-service-go/config"
	"github.com/st0mb1e/bank-service-go/dao/repo"
	"github.com/st0mb1e/bank-service-go/handler/api"
	"github.com/st0mb1e/bank-service-go/handler/auth"
	"github.com/st0mb1e/bank-service-go/integration/mail"
	"github.com/st0mb1e/bank-service-go/middleware"
	"github.com/st0mb1e/bank-service-go/service"
)

func main() {
	_ = godotenv.Load()

	appCfg := config.NewAppConfigFromEnv()
	secrets := config.NewSecretsConfigFromEnv()
	if secrets.JWTSecret == "" || secrets.HMACSecret == "" || secrets.PGPPassphrase == "" {
		log.Fatal("JWT_SECRET, HMAC_SECRET and PGP_PASSPHRASE must be set")
	}

	logrus.SetLevel(appCfg.LogrusLvl)
	lg := logrus.StandardLogger()

	dbCfg := config.NewDbConfigFromEnv()
	db, err := sql.Open("postgres", dbCfg.GetDBUrl())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatal(err)
	}

	userRepo := repo.NewUserRepo(db)
	accRepo := repo.NewAccountRepo(db)
	txRepo := repo.NewTransactionRepo(db)
	cardRepo := repo.NewCardRepo(db)
	creditRepo := repo.NewCreditRepo(db)

	authSvc := service.NewAuthService(userRepo)
	accSvc := service.NewAccountService(db, accRepo, txRepo)
	transferSvc := service.NewTransferService(db, accRepo, txRepo)
	cardSvc := service.NewCardService(db, cardRepo, accRepo, txRepo)
	creditSvc := service.NewCreditService(db, creditRepo, accRepo, txRepo)
	analyticsSvc := service.NewAnalyticsService(db, accRepo, txRepo, creditRepo)

	mailer := mail.NewMailer(config.NewSMTPConfigFromEnv(), lg)
	apiSrv := api.NewServer(lg, secrets, mailer, userRepo, accSvc, transferSvc, cardSvc, creditSvc, analyticsSvc)

	authHandlers := auth.NewAuthHandlers(authSvc, []byte(secrets.JWTSecret))

	router := mux.NewRouter()
	router.HandleFunc("/register", authHandlers.Register).Methods(http.MethodPost)
	router.HandleFunc("/login", authHandlers.Login).Methods(http.MethodPost)

	protected := router.PathPrefix("/").Subrouter()
	protected.Use(middleware.AuthMiddleware([]byte(secrets.JWTSecret)))
	apiSrv.RegisterProtected(protected)

	service.StartCreditScheduler(lg, db, creditRepo, accRepo, txRepo)

	addr := ":" + appCfg.Port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
