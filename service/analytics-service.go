package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
	"github.com/st0mb1e/bank-service-go/dao/repo"
)

type AnalyticsSummary struct {
	Month          string          `json:"month"`
	IncomeRUB      decimal.Decimal `json:"income_rub"`
	ExpenseRUB     decimal.Decimal `json:"expense_rub"`
	ActiveCredit   decimal.Decimal `json:"active_credit_principal_rub"`
}

type BalancePredict struct {
	AccountID        string          `json:"account_id"`
	Days             int             `json:"days"`
	CurrentBalance   decimal.Decimal `json:"current_balance_rub"`
	ScheduledOutflow decimal.Decimal `json:"scheduled_outflow_rub"`
	ForecastBalance  decimal.Decimal `json:"forecast_balance_rub"`
}

type AnalyticsService interface {
	Summary(ctx context.Context, userID, yearMonth string) (*AnalyticsSummary, error)
	CreditLoad(ctx context.Context, userID string) (decimal.Decimal, error)
	PredictBalance(ctx context.Context, userID, accountID string, days int) (*BalancePredict, error)
}

type analyticsService struct {
	db         *sql.DB
	accRepo    repo.AccountRepo
	txRepo     repo.TransactionRepo
	creditRepo repo.CreditRepo
}

func NewAnalyticsService(db *sql.DB, accRepo repo.AccountRepo, txRepo repo.TransactionRepo, creditRepo repo.CreditRepo) AnalyticsService {
	return &analyticsService{db: db, accRepo: accRepo, txRepo: txRepo, creditRepo: creditRepo}
}

func (s *analyticsService) Summary(ctx context.Context, userID, yearMonth string) (*AnalyticsSummary, error) {
	if yearMonth == "" {
		yearMonth = time.Now().UTC().Format("2006-01")
	}
	inc, exp, err := s.txRepo.SumByUserMonth(ctx, userID, yearMonth)
	if err != nil {
		return nil, err
	}
	debt, err := s.creditRepo.SumActivePrincipalForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &AnalyticsSummary{
		Month:        yearMonth,
		IncomeRUB:    inc,
		ExpenseRUB:   exp,
		ActiveCredit: debt,
	}, nil
}

func (s *analyticsService) CreditLoad(ctx context.Context, userID string) (decimal.Decimal, error) {
	return s.creditRepo.SumActivePrincipalForUser(ctx, userID)
}

func (s *analyticsService) PredictBalance(ctx context.Context, userID, accountID string, days int) (*BalancePredict, error) {
	if days < 1 || days > 365 {
		return nil, ErrValidation
	}
	a, err := s.accRepo.GetByIDForUser(ctx, s.db, accountID, userID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}
	until := time.Now().UTC().AddDate(0, 0, days)
	outflow, err := s.creditRepo.SumPendingPaymentsUntil(ctx, accountID, until)
	if err != nil {
		return nil, err
	}
	fc := a.Balance.Sub(outflow)
	return &BalancePredict{
		AccountID:        accountID,
		Days:             days,
		CurrentBalance:   a.Balance,
		ScheduledOutflow: outflow,
		ForecastBalance:  fc,
	}, nil
}
