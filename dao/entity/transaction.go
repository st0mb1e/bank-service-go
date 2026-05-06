package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID                    string
	AccountID             string
	Amount                decimal.Decimal
	TxType                string
	CounterpartyAccountID *string
	CreditID              *string
	CardID                *string
	CreatedAt             time.Time
}
