package repo

import (
	"database/sql"

	"github.com/shopspring/decimal"
)

func scanDecimal(ns sql.NullString) (decimal.Decimal, error) {
	if !ns.Valid {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(ns.String)
}
