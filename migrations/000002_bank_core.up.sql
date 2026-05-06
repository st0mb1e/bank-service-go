CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE users RENAME COLUMN password TO password_hash;

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    balance NUMERIC(18, 2) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_accounts_user_id ON accounts (user_id);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    amount NUMERIC(18, 2) NOT NULL,
    tx_type VARCHAR(32) NOT NULL,
    counterparty_account_id UUID REFERENCES accounts (id),
    credit_id UUID,
    card_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_account_id ON transactions (account_id);
CREATE INDEX idx_transactions_created_at ON transactions (created_at);

CREATE TABLE cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    pan_encrypted TEXT NOT NULL,
    expiry_encrypted TEXT NOT NULL,
    cvv_hash TEXT NOT NULL,
    integrity_hmac TEXT NOT NULL,
    last4 CHAR(4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cards_user_id ON cards (user_id);
CREATE INDEX idx_cards_account_id ON cards (account_id);

CREATE TABLE credits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    disbursement_account_id UUID NOT NULL REFERENCES accounts (id),
    repayment_account_id UUID NOT NULL REFERENCES accounts (id),
    principal NUMERIC(18, 2) NOT NULL CHECK (principal > 0),
    annual_rate_percent NUMERIC(10, 4) NOT NULL,
    term_months INT NOT NULL CHECK (term_months > 0),
    monthly_payment NUMERIC(18, 2) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_credits_user_id ON credits (user_id);

ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_credit FOREIGN KEY (credit_id) REFERENCES credits (id);

ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_card FOREIGN KEY (card_id) REFERENCES cards (id);

CREATE TABLE payment_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credit_id UUID NOT NULL REFERENCES credits (id) ON DELETE CASCADE,
    installment_no INT NOT NULL,
    due_date DATE NOT NULL,
    amount_due NUMERIC(18, 2) NOT NULL,
    amount_paid NUMERIC(18, 2) NOT NULL DEFAULT 0,
    penalty NUMERIC(18, 2) NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    paid_at TIMESTAMPTZ,
    UNIQUE (credit_id, installment_no)
);

CREATE INDEX idx_payment_schedules_credit_due ON payment_schedules (credit_id, due_date);
CREATE INDEX idx_payment_schedules_status_due ON payment_schedules (status, due_date);
