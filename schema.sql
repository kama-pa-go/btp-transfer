-- Init wallets table
CREATE TABLE IF NOT EXISTS wallets (
                                       address VARCHAR(255) PRIMARY KEY,
    balance BIGINT NOT NULL CHECK (balance >= 0)
    );