package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID        string
	UserID    string
	Balance   decimal.Decimal
	CreatedAt time.Time
}
