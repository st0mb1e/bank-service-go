package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/shopspring/decimal"
	"github.com/st0mb1e/bank-service-go/cryptoutil"
	"github.com/st0mb1e/bank-service-go/dao/entity"
	"github.com/st0mb1e/bank-service-go/dao/repo"
	"github.com/st0mb1e/bank-service-go/integration/mail"
	"golang.org/x/crypto/bcrypt"
)

type CardService interface {
	Issue(ctx context.Context, userID, accountID string, pgpPass, hmacSecret []byte) (*CardMaskedResponse, error)
	ListMasked(ctx context.Context, userID string) ([]CardMaskedResponse, error)
	View(ctx context.Context, userID, cardID string, pgpPass, hmacSecret []byte) (*CardFullResponse, error)
	Pay(ctx context.Context, userID, cardID string, amount decimal.Decimal, mailer *mail.Mailer, userEmail string) error
}

type CardMaskedResponse struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Last4     string `json:"last4"`
	CreatedAt string `json:"created_at"`
}

type CardFullResponse struct {
	PAN         string `json:"pan"`
	Expiry      string `json:"expiry"`
	Last4       string `json:"last4"`
	IntegrityOK bool   `json:"integrity_ok"`
}

type cardService struct {
	db       *sql.DB
	cardRepo repo.CardRepo
	accRepo  repo.AccountRepo
	txRepo   repo.TransactionRepo
}

func NewCardService(db *sql.DB, cardRepo repo.CardRepo, accRepo repo.AccountRepo, txRepo repo.TransactionRepo) CardService {
	return &cardService{db: db, cardRepo: cardRepo, accRepo: accRepo, txRepo: txRepo}
}

func (s *cardService) Issue(ctx context.Context, userID, accountID string, pgpPass, hmacSecret []byte) (*CardMaskedResponse, error) {
	a, err := s.accRepo.GetByIDForUser(ctx, s.db, accountID, userID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrForbidden
	}
	pan, err := cryptoutil.GenerateLuhnPAN(16)
	if err != nil {
		return nil, err
	}
	expiry := randomExpiry()
	cvv := randomCVV()

	integrityBase := pan + "|" + expiry
	hmacVal := cryptoutil.ComputeHMAC(integrityBase, hmacSecret)

	panEnc, err := cryptoutil.EncryptPGPSymmetric([]byte(pan), pgpPass)
	if err != nil {
		return nil, err
	}
	expEnc, err := cryptoutil.EncryptPGPSymmetric([]byte(expiry), pgpPass)
	if err != nil {
		return nil, err
	}
	cvvHash, err := bcrypt.GenerateFromPassword([]byte(cvv), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	c := &entity.Card{
		UserID:          userID,
		AccountID:       accountID,
		PANEncrypted:    base64.StdEncoding.EncodeToString(panEnc),
		ExpiryEncrypted: base64.StdEncoding.EncodeToString(expEnc),
		CVVHash:         string(cvvHash),
		IntegrityHMAC:   hmacVal,
		Last4:           pan[len(pan)-4:],
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.cardRepo.Insert(ctx, tx, c); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &CardMaskedResponse{ID: c.ID, AccountID: c.AccountID, Last4: c.Last4, CreatedAt: c.CreatedAt.Format(time.RFC3339)}, nil
}

func (s *cardService) ListMasked(ctx context.Context, userID string) ([]CardMaskedResponse, error) {
	list, err := s.cardRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]CardMaskedResponse, 0, len(list))
	for _, c := range list {
		out = append(out, CardMaskedResponse{ID: c.ID, AccountID: c.AccountID, Last4: c.Last4, CreatedAt: c.CreatedAt.Format(time.RFC3339)})
	}
	return out, nil
}

func (s *cardService) View(ctx context.Context, userID, cardID string, pgpPass, hmacSecret []byte) (*CardFullResponse, error) {
	c, err := s.cardRepo.GetByIDForUser(ctx, s.db, cardID, userID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	panBytes, err := base64.StdEncoding.DecodeString(c.PANEncrypted)
	if err != nil {
		return nil, err
	}
	expBytes, err := base64.StdEncoding.DecodeString(c.ExpiryEncrypted)
	if err != nil {
		return nil, err
	}
	panB, err := cryptoutil.DecryptPGPSymmetric(panBytes, pgpPass)
	if err != nil {
		return nil, err
	}
	expB, err := cryptoutil.DecryptPGPSymmetric(expBytes, pgpPass)
	if err != nil {
		return nil, err
	}
	pan := string(panB)
	exp := string(expB)
	ok := cryptoutil.VerifyHMAC(pan+"|"+exp, hmacSecret, c.IntegrityHMAC) && cryptoutil.ValidLuhn(pan)
	return &CardFullResponse{PAN: pan, Expiry: exp, Last4: c.Last4, IntegrityOK: ok}, nil
}

func (s *cardService) Pay(ctx context.Context, userID, cardID string, amount decimal.Decimal, mailer *mail.Mailer, userEmail string) error {
	if !amount.GreaterThan(decimal.Zero) {
		return ErrValidation
	}
	c, err := s.cardRepo.GetByIDForUser(ctx, s.db, cardID, userID)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrNotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if acc, err := s.accRepo.GetByIDForUser(ctx, tx, c.AccountID, userID); err != nil || acc == nil {
		return ErrForbidden
	}
	cid := c.ID
	if _, err := s.accRepo.AddBalance(ctx, tx, c.AccountID, amount.Neg()); err != nil {
		if errors.Is(err, repo.ErrInsufficientFunds) {
			return ErrInsufficient
		}
		return err
	}
	if _, err := s.txRepo.Insert(ctx, tx, c.AccountID, amount, "card_payment", nil, nil, &cid); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if mailer != nil && userEmail != "" {
		_ = mailer.SendPaymentNotification(userEmail, amount.StringFixed(2))
	}
	return nil
}

func randomExpiry() string {
	m, _ := rand.Int(rand.Reader, big.NewInt(12))
	month := int(m.Int64()) + 1
	y, _ := rand.Int(rand.Reader, big.NewInt(5))
	year := 26 + int(y.Int64())
	return fmt.Sprintf("%02d/%02d", month, year)
}

func randomCVV() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000))
	return fmt.Sprintf("%03d", n.Int64())
}

