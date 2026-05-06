package service

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/shopspring/decimal"
	"github.com/st0mb1e/bank-service-go/dao/entity"
	"github.com/st0mb1e/bank-service-go/dao/repo"
	"github.com/st0mb1e/bank-service-go/integration/cbr"
)

type CreditService interface {
	Issue(ctx context.Context, userID string, principal decimal.Decimal, termMonths int, disburseAccountID, repaymentAccountID string, marginPercent float64) (*entity.Credit, error)
	Schedule(ctx context.Context, userID, creditID string) ([]entity.PaymentScheduleRow, error)
}

type creditService struct {
	db         *sql.DB
	creditRepo repo.CreditRepo
	accRepo    repo.AccountRepo
	txRepo     repo.TransactionRepo
}

func NewCreditService(db *sql.DB, creditRepo repo.CreditRepo, accRepo repo.AccountRepo, txRepo repo.TransactionRepo) CreditService {
	return &creditService{db: db, creditRepo: creditRepo, accRepo: accRepo, txRepo: txRepo}
}

func annuityMonthly(principal decimal.Decimal, annualRatePct float64, months int) decimal.Decimal {
	r := annualRatePct / 12.0 / 100.0
	n := float64(months)
	pv := principal.InexactFloat64()
	if months <= 0 {
		return decimal.Zero
	}
	if math.Abs(r) < 1e-12 {
		return principal.Div(decimal.NewFromInt(int64(months))).Round(2)
	}
	pow := math.Pow(1+r, n)
	pay := pv * (r * pow) / (pow - 1)
	return decimal.NewFromFloat(pay).Round(2)
}

func (s *creditService) Issue(ctx context.Context, userID string, principal decimal.Decimal, termMonths int, disburseAccountID, repaymentAccountID string, marginPercent float64) (*entity.Credit, error) {
	if !principal.GreaterThan(decimal.Zero) || termMonths <= 0 {
		return nil, ErrValidation
	}
	da, err := s.accRepo.GetByIDForUser(ctx, s.db, disburseAccountID, userID)
	if err != nil {
		return nil, err
	}
	if da == nil {
		return nil, ErrForbidden
	}
	ra, err := s.accRepo.GetByIDForUser(ctx, s.db, repaymentAccountID, userID)
	if err != nil {
		return nil, err
	}
	if ra == nil {
		return nil, ErrForbidden
	}
	baseRate, err := cbr.FetchKeyRatePercent()
	if err != nil {
		return nil, err
	}
	annual := baseRate + marginPercent
	monthly := annuityMonthly(principal, annual, termMonths)

	cr := &entity.Credit{
		UserID:                userID,
		DisbursementAccountID: disburseAccountID,
		RepaymentAccountID:    repaymentAccountID,
		Principal:             principal,
		AnnualRatePercent:     decimal.NewFromFloat(annual),
		TermMonths:            termMonths,
		MonthlyPayment:        monthly,
		Status:                "active",
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := s.creditRepo.InsertCredit(ctx, tx, cr); err != nil {
		return nil, err
	}
	creditID := cr.ID
	now := time.Now().UTC()
	for i := 1; i <= termMonths; i++ {
		due := now.AddDate(0, i, 0)
		row := &entity.PaymentScheduleRow{
			CreditID:      creditID,
			InstallmentNo: i,
			DueDate:       time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, time.UTC),
			AmountDue:     monthly,
			Status:        "pending",
		}
		if err := s.creditRepo.InsertSchedule(ctx, tx, row); err != nil {
			return nil, err
		}
	}
	if _, err := s.accRepo.AddBalance(ctx, tx, disburseAccountID, principal); err != nil {
		return nil, err
	}
	crid := creditID
	if _, err := s.txRepo.Insert(ctx, tx, disburseAccountID, principal, "credit_disbursement", nil, &crid, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return cr, nil
}

func (s *creditService) Schedule(ctx context.Context, userID, creditID string) ([]entity.PaymentScheduleRow, error) {
	c, err := s.creditRepo.GetByIDForUser(ctx, creditID, userID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	return s.creditRepo.ListSchedule(ctx, creditID)
}

func ProcessDuePayments(ctx context.Context, db *sql.DB, creditRepo repo.CreditRepo, accRepo repo.AccountRepo, txRepo repo.TransactionRepo) error {
	rows, err := creditRepo.ListPendingOverdue(ctx, time.Now())
	if err != nil {
		return err
	}
	for _, row := range rows {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		repAcc, due, err := creditRepo.GetRepaymentAccountByScheduleID(ctx, tx, row.ID)
		if err != nil {
			_ = tx.Rollback()
			continue
		}
		if !due.GreaterThan(decimal.Zero) {
			_ = creditRepo.MarkSchedulePaid(ctx, tx, row.ID)
			_ = tx.Commit()
			continue
		}
		_, errPay := accRepo.AddBalance(ctx, tx, repAcc, due.Neg())
		if errPay == nil {
			cid := row.CreditID
			if _, errIns := txRepo.Insert(ctx, tx, repAcc, due, "credit_payment", nil, &cid, nil); errIns != nil {
				_ = tx.Rollback()
				continue
			}
			if errMark := creditRepo.MarkSchedulePaid(ctx, tx, row.ID); errMark != nil {
				_ = tx.Rollback()
				continue
			}
			_ = tx.Commit()
			continue
		}
		if !errors.Is(errPay, repo.ErrInsufficientFunds) {
			_ = tx.Rollback()
			continue
		}
		owing := row.AmountDue.Add(row.Penalty).Sub(row.AmountPaid)
		bump := owing.Mul(decimal.RequireFromString("0.1")).Round(2)
		if bump.LessThanOrEqual(decimal.Zero) {
			_ = tx.Rollback()
			continue
		}
		if err := creditRepo.BumpOverdueAmount(ctx, tx, row.ID, bump); err != nil {
			_ = tx.Rollback()
			continue
		}
		_ = tx.Commit()
	}
	return nil
}
