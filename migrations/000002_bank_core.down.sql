DROP TABLE IF EXISTS payment_schedules;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_card;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_credit;
DROP TABLE IF EXISTS cards;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS credits;
DROP TABLE IF EXISTS accounts;
ALTER TABLE users RENAME COLUMN password_hash TO password;
