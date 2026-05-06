package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestAnnuityMonthly(t *testing.T) {
	t.Parallel()
	v := annuityMonthly(decimal.RequireFromString("100000"), 12.0, 12)
	if !v.GreaterThan(decimal.Zero) {
		t.Fatal(v)
	}
}

func TestAnnuityMonthly_zeroInterestSplit(t *testing.T) {
	t.Parallel()
	v := annuityMonthly(decimal.RequireFromString("12000"), 0, 12)
	want := decimal.RequireFromString("1000")
	if !v.Equal(want) {
		t.Fatalf("got %v want %v", v, want)
	}
}
