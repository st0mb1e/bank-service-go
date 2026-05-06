package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/st0mb1e/bank-service-go/dao/entity"
	"github.com/shopspring/decimal"
)

type CreditRepo interface {
	InsertCredit(ctx context.Context, q SQLQuerier, c *entity.Credit) error
	InsertSchedule(ctx context.Context, q SQLQuerier, row *entity.PaymentScheduleRow) error
	GetByIDForUser(ctx context.Context, creditID, userID string) (*entity.Credit, error)
	ListSchedule(ctx context.Context, creditID string) ([]entity.PaymentScheduleRow, error)
	ListPendingOverdue(ctx context.Context, before time.Time) ([]entity.PaymentScheduleRow, error)
	MarkSchedulePaid(ctx context.Context, q SQLQuerier, scheduleID string) error
	BumpOverdueAmount(ctx context.Context, q SQLQuerier, scheduleID string, addToDue decimal.Decimal) error
	GetRepaymentAccountByScheduleID(ctx context.Context, q SQLQuerier, scheduleID string) (repaymentAccountID string, totalDue decimal.Decimal, err error)
	SumActivePrincipalForUser(ctx context.Context, userID string) (decimal.Decimal, error)
	SumPendingPaymentsUntil(ctx context.Context, accountID string, until time.Time) (decimal.Decimal, error)
}

type creditRepo struct {
	db *sql.DB
}

func NewCreditRepo(db *sql.DB) CreditRepo {
	return &creditRepo{db: db}
}

func (r *creditRepo) InsertCredit(ctx context.Context, q SQLQuerier, c *entity.Credit) error {
	return q.QueryRowContext(ctx, `
		INSERT INTO credits (user_id, disbursement_account_id, repayment_account_id, principal, annual_rate_percent, term_months, monthly_payment, status)
		VALUES ($1, $2, $3, $4::numeric, $5::numeric, $6, $7::numeric, $8)
		RETURNING id, created_at`,
		c.UserID, c.DisbursementAccountID, c.RepaymentAccountID, c.Principal, c.AnnualRatePercent, c.TermMonths, c.MonthlyPayment, c.Status,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *creditRepo) InsertSchedule(ctx context.Context, q SQLQuerier, row *entity.PaymentScheduleRow) error {
	return q.QueryRowContext(ctx, `
		INSERT INTO payment_schedules (credit_id, installment_no, due_date, amount_due, status)
		VALUES ($1, $2, $3::date, $4::numeric, $5)
		RETURNING id`,
		row.CreditID, row.InstallmentNo, row.DueDate, row.AmountDue, row.Status,
	).Scan(&row.ID)
}

func (r *creditRepo) GetByIDForUser(ctx context.Context, creditID, userID string) (*entity.Credit, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, disbursement_account_id, repayment_account_id,
			principal::text, annual_rate_percent::text, term_months, monthly_payment::text, status, created_at
		FROM credits WHERE id = $1 AND user_id = $2`,
		creditID, userID,
	)
	var c entity.Credit
	var p, ar, mp string
	if err := row.Scan(&c.ID, &c.UserID, &c.DisbursementAccountID, &c.RepaymentAccountID, &p, &ar, &c.TermMonths, &mp, &c.Status, &c.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var err error
	c.Principal, err = decimal.NewFromString(p)
	if err != nil {
		return nil, err
	}
	c.AnnualRatePercent, err = decimal.NewFromString(ar)
	if err != nil {
		return nil, err
	}
	c.MonthlyPayment, err = decimal.NewFromString(mp)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *creditRepo) ListSchedule(ctx context.Context, creditID string) ([]entity.PaymentScheduleRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, credit_id, installment_no, due_date, amount_due::text, amount_paid::text, penalty::text, status, paid_at
		FROM payment_schedules WHERE credit_id = $1 ORDER BY installment_no`,
		creditID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScheduleRows(rows)
}

func scanScheduleRows(rows *sql.Rows) ([]entity.PaymentScheduleRow, error) {
	var list []entity.PaymentScheduleRow
	for rows.Next() {
		var row entity.PaymentScheduleRow
		var ad, ap, pen string
		var paidAt sql.NullTime
		if err := rows.Scan(&row.ID, &row.CreditID, &row.InstallmentNo, &row.DueDate, &ad, &ap, &pen, &row.Status, &paidAt); err != nil {
			return nil, err
		}
		var err error
		row.AmountDue, err = decimal.NewFromString(ad)
		if err != nil {
			return nil, err
		}
		row.AmountPaid, err = decimal.NewFromString(ap)
		if err != nil {
			return nil, err
		}
		row.Penalty, err = decimal.NewFromString(pen)
		if err != nil {
			return nil, err
		}
		if paidAt.Valid {
			t := paidAt.Time
			row.PaidAt = &t
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *creditRepo) ListPendingOverdue(ctx context.Context, asOf time.Time) ([]entity.PaymentScheduleRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, credit_id, installment_no, due_date, amount_due::text, amount_paid::text, penalty::text, status, paid_at
		FROM payment_schedules
		WHERE status IN ('pending','overdue') AND due_date <= $1::date`,
		asOf.UTC().Format("2006-01-02"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScheduleRows(rows)
}

func (r *creditRepo) MarkSchedulePaid(ctx context.Context, q SQLQuerier, scheduleID string) error {
	_, err := q.ExecContext(ctx, `
		UPDATE payment_schedules
		SET status = 'paid', amount_paid = amount_due + penalty, paid_at = now()
		WHERE id = $1 AND status IN ('pending','overdue')`,
		scheduleID,
	)
	return err
}

func (r *creditRepo) GetRepaymentAccountByScheduleID(ctx context.Context, q SQLQuerier, scheduleID string) (string, decimal.Decimal, error) {
	row := q.QueryRowContext(ctx, `
		SELECT c.repayment_account_id, (ps.amount_due + ps.penalty - ps.amount_paid)::text
		FROM payment_schedules ps
		JOIN credits c ON c.id = ps.credit_id
		WHERE ps.id = $1`,
		scheduleID,
	)
	var acc, amt string
	if err := row.Scan(&acc, &amt); err != nil {
		return "", decimal.Zero, err
	}
	d, err := decimal.NewFromString(amt)
	if err != nil {
		return "", decimal.Zero, err
	}
	return acc, d, nil
}

func (r *creditRepo) BumpOverdueAmount(ctx context.Context, q SQLQuerier, scheduleID string, addToDue decimal.Decimal) error {
	_, err := q.ExecContext(ctx, `
		UPDATE payment_schedules
		SET status = 'overdue', amount_due = amount_due + $2::numeric
		WHERE id = $1`,
		scheduleID, addToDue,
	)
	return err
}

func (r *creditRepo) SumActivePrincipalForUser(ctx context.Context, userID string) (decimal.Decimal, error) {
	var s sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(principal), 0)::text FROM credits WHERE user_id = $1::uuid AND status = 'active'`,
		userID,
	).Scan(&s)
	if err != nil {
		return decimal.Zero, err
	}
	if !s.Valid || s.String == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s.String)
}

func (r *creditRepo) SumPendingPaymentsUntil(ctx context.Context, accountID string, until time.Time) (decimal.Decimal, error) {
	var s sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(ps.amount_due + ps.penalty - ps.amount_paid), 0)::text
		FROM payment_schedules ps
		JOIN credits c ON c.id = ps.credit_id
		WHERE c.repayment_account_id = $1::uuid
		  AND ps.status IN ('pending','overdue')
		  AND ps.due_date <= $2::date`,
		accountID, until.Format("2006-01-02"),
	).Scan(&s)
	if err != nil {
		return decimal.Zero, err
	}
	if !s.Valid {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s.String)
}
