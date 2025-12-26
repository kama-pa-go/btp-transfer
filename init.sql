-- Add first wallet with 1,000,000 tokens
-- ON CONFLICT DO NOTHING - ensures idempotency
INSERT INTO wallets (address, balance)
VALUES ('0x0000000000000000000000000000000000000000', 1000000)
    ON CONFLICT (address) DO NOTHING;

-- Database for tests --
CREATE DATABASE btp_test;
\c btp_test;

-- Tworzymy tę samą tabelę w bazie testowej
CREATE TABLE IF NOT EXISTS wallets (
    address VARCHAR(255) PRIMARY KEY,
    balance BIGINT NOT NULL CHECK (balance >= 0)
);