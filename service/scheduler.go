package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/st0mb1e/bank-service-go/dao/repo"
)

func StartCreditScheduler(log *logrus.Logger, db *sql.DB, creditRepo repo.CreditRepo, accRepo repo.AccountRepo, txRepo repo.TransactionRepo) {
	t := time.NewTicker(12 * time.Hour)
	go func() {
		defer t.Stop()
		run := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := ProcessDuePayments(ctx, db, creditRepo, accRepo, txRepo); err != nil {
				log.Errorf("credit due payments: %v", err)
			}
		}
		run()
		for range t.C {
			run()
		}
	}()
}
