package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

type Credit struct {
	ID                      string
	UserID                  string
	DisbursementAccountID   string
	RepaymentAccountID      string
	Principal               decimal.Decimal
	AnnualRatePercent       decimal.Decimal
	TermMonths              int
	MonthlyPayment          decimal.Decimal
	Status                  string
	CreatedAt               time.Time
}

type PaymentScheduleRow struct {
	ID           string
	CreditID     string
	InstallmentNo int
	DueDate      time.Time
	AmountDue    decimal.Decimal
	AmountPaid   decimal.Decimal
	Penalty      decimal.Decimal
	Status       string
	PaidAt       *time.Time
}
