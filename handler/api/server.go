package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/st0mb1e/bank-service-go/config"
	"github.com/st0mb1e/bank-service-go/dao/repo"
	"github.com/st0mb1e/bank-service-go/httputil"
	"github.com/st0mb1e/bank-service-go/integration/mail"
	"github.com/st0mb1e/bank-service-go/middleware"
	"github.com/st0mb1e/bank-service-go/service"
)

type Server struct {
	Log       *logrus.Logger
	Secrets   *config.SecretsConfig
	Mailer    *mail.Mailer
	UserRepo  repo.UserRepo
	Accounts  service.AccountService
	Transfer  service.TransferService
	Cards     service.CardService
	Credits   service.CreditService
	Analytics service.AnalyticsService
}

func NewServer(
	log *logrus.Logger,
	secrets *config.SecretsConfig,
	mailer *mail.Mailer,
	userRepo repo.UserRepo,
	accounts service.AccountService,
	transfer service.TransferService,
	cards service.CardService,
	credits service.CreditService,
	analytics service.AnalyticsService,
) *Server {
	return &Server{
		Log:       log,
		Secrets:   secrets,
		Mailer:    mailer,
		UserRepo:  userRepo,
		Accounts:  accounts,
		Transfer:  transfer,
		Cards:     cards,
		Credits:   credits,
		Analytics: analytics,
	}
}

func (s *Server) RegisterProtected(r *mux.Router) {
	r.HandleFunc("/accounts", s.createAccount).Methods(http.MethodPost)
	r.HandleFunc("/accounts", s.listAccounts).Methods(http.MethodGet)
	r.HandleFunc("/accounts/{accountId}/deposit", s.deposit).Methods(http.MethodPost)
	r.HandleFunc("/accounts/{accountId}/withdraw", s.withdraw).Methods(http.MethodPost)
	r.HandleFunc("/accounts/{accountId}/predict", s.predict).Methods(http.MethodGet)

	r.HandleFunc("/transfer", s.transfer).Methods(http.MethodPost)

	r.HandleFunc("/cards", s.issueCard).Methods(http.MethodPost)
	r.HandleFunc("/cards", s.listCards).Methods(http.MethodGet)
	r.HandleFunc("/cards/{cardId}", s.viewCard).Methods(http.MethodGet)
	r.HandleFunc("/cards/{cardId}/pay", s.payCard).Methods(http.MethodPost)

	r.HandleFunc("/credits", s.issueCredit).Methods(http.MethodPost)
	r.HandleFunc("/credits/{creditId}/schedule", s.creditSchedule).Methods(http.MethodGet)

	r.HandleFunc("/analytics", s.analytics).Methods(http.MethodGet)
	r.HandleFunc("/analytics/credit-load", s.creditLoad).Methods(http.MethodGet)
}

func userID(r *http.Request) (string, bool) {
	return middleware.UserIDFromContext(r.Context())
}

type amountBody struct {
	Amount string `json:"amount"`
}

func parseAmount(b amountBody) (decimal.Decimal, error) {
	if b.Amount == "" {
		return decimal.Zero, errors.New("empty amount")
	}
	return decimal.NewFromString(b.Amount)
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	a, err := s.Accounts.Create(r.Context(), uid)
	if err != nil {
		s.Log.Errorf("accounts create: %v", err)
		httputil.ErrorJSON(w, http.StatusInternalServerError, "failed")
		return
	}
	httputil.JSON(w, http.StatusCreated, map[string]any{"id": a.ID, "balance": a.Balance.String(), "currency": "RUB"})
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	list, err := s.Accounts.List(r.Context(), uid)
	if err != nil {
		httputil.ErrorJSON(w, http.StatusInternalServerError, "failed")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, map[string]any{"id": a.ID, "balance": a.Balance.String(), "currency": "RUB"})
	}
	httputil.JSON(w, http.StatusOK, out)
}

func (s *Server) deposit(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var b amountBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	amt, err := parseAmount(b)
	if err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid amount")
		return
	}
	accID := mux.Vars(r)["accountId"]
	if err := s.Accounts.Deposit(r.Context(), uid, accID, amt); err != nil {
		s.mapSvcErr(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) withdraw(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var b amountBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	amt, err := parseAmount(b)
	if err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid amount")
		return
	}
	accID := mux.Vars(r)["accountId"]
	if err := s.Accounts.Withdraw(r.Context(), uid, accID, amt); err != nil {
		s.mapSvcErr(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type transferBody struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        string `json:"amount"`
}

func (s *Server) transfer(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var b transferBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	amt, err := decimal.NewFromString(b.Amount)
	if err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid amount")
		return
	}
	if err := s.Transfer.Transfer(r.Context(), uid, b.FromAccountID, b.ToAccountID, amt); err != nil {
		s.mapSvcErr(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type issueCardBody struct {
	AccountID string `json:"account_id"`
}

func (s *Server) issueCard(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var b issueCardBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	if s.Secrets.PGPPassphrase == "" || s.Secrets.HMACSecret == "" {
		httputil.ErrorJSON(w, http.StatusInternalServerError, "server crypto not configured")
		return
	}
	res, err := s.Cards.Issue(r.Context(), uid, b.AccountID, []byte(s.Secrets.PGPPassphrase), []byte(s.Secrets.HMACSecret))
	if err != nil {
		s.mapSvcErr(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, res)
}

func (s *Server) listCards(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	list, err := s.Cards.ListMasked(r.Context(), uid)
	if err != nil {
		httputil.ErrorJSON(w, http.StatusInternalServerError, "failed")
		return
	}
	httputil.JSON(w, http.StatusOK, list)
}

func (s *Server) viewCard(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	cid := mux.Vars(r)["cardId"]
	if s.Secrets.PGPPassphrase == "" || s.Secrets.HMACSecret == "" {
		httputil.ErrorJSON(w, http.StatusInternalServerError, "server crypto not configured")
		return
	}
	v, err := s.Cards.View(r.Context(), uid, cid, []byte(s.Secrets.PGPPassphrase), []byte(s.Secrets.HMACSecret))
	if err != nil {
		s.mapSvcErr(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, v)
}

func (s *Server) payCard(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var b amountBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	amt, err := parseAmount(b)
	if err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid amount")
		return
	}
	cid := mux.Vars(r)["cardId"]
	u, err := s.UserRepo.GetByID(r.Context(), uid)
	if err != nil || u == nil {
		httputil.ErrorJSON(w, http.StatusInternalServerError, "user lookup failed")
		return
	}
	if err := s.Cards.Pay(r.Context(), uid, cid, amt, s.Mailer, u.Email); err != nil {
		s.mapSvcErr(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type creditBody struct {
	Principal             string `json:"principal"`
	TermMonths            int    `json:"term_months"`
	DisbursementAccountID string `json:"disbursement_account_id"`
	RepaymentAccountID    string `json:"repayment_account_id"`
}

func (s *Server) issueCredit(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var b creditBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	principal, err := decimal.NewFromString(b.Principal)
	if err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid principal")
		return
	}
	cr, err := s.Credits.Issue(r.Context(), uid, principal, b.TermMonths, b.DisbursementAccountID, b.RepaymentAccountID, s.Secrets.CBRMarginPercent)
	if err != nil {
		s.mapSvcErr(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, map[string]any{
		"id": cr.ID, "monthly_payment": cr.MonthlyPayment.String(), "annual_rate_percent": cr.AnnualRatePercent.String(),
	})
}

func (s *Server) creditSchedule(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	creditID := mux.Vars(r)["creditId"]
	rows, err := s.Credits.Schedule(r.Context(), uid, creditID)
	if err != nil {
		s.mapSvcErr(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, rows)
}

func (s *Server) analytics(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	month := r.URL.Query().Get("month")
	sum, err := s.Analytics.Summary(r.Context(), uid, month)
	if err != nil {
		httputil.ErrorJSON(w, http.StatusInternalServerError, "failed")
		return
	}
	httputil.JSON(w, http.StatusOK, sum)
}

func (s *Server) creditLoad(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	d, err := s.Analytics.CreditLoad(r.Context(), uid)
	if err != nil {
		httputil.ErrorJSON(w, http.StatusInternalServerError, "failed")
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"active_principal_rub": d.String()})
}

func (s *Server) predict(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httputil.ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days == 0 {
		days = 30
	}
	accID := mux.Vars(r)["accountId"]
	p, err := s.Analytics.PredictBalance(r.Context(), uid, accID, days)
	if err != nil {
		s.mapSvcErr(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, p)
}

func (s *Server) mapSvcErr(w http.ResponseWriter, err error) {
	switch err {
	case service.ErrValidation:
		httputil.ErrorJSON(w, http.StatusBadRequest, err.Error())
	case service.ErrUnauthorized:
		httputil.ErrorJSON(w, http.StatusUnauthorized, err.Error())
	case service.ErrForbidden:
		httputil.ErrorJSON(w, http.StatusForbidden, err.Error())
	case service.ErrNotFound:
		httputil.ErrorJSON(w, http.StatusNotFound, err.Error())
	case service.ErrInsufficient:
		httputil.ErrorJSON(w, http.StatusBadRequest, "insufficient funds")
	case service.ErrConflict:
		httputil.ErrorJSON(w, http.StatusConflict, err.Error())
	default:
		s.Log.Errorf("api: %v", err)
		httputil.ErrorJSON(w, http.StatusInternalServerError, "failed")
	}
}
