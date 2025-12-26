-- Add first wallet with 1,000,000 tokens
-- ON CONFLICT DO NOTHING - ensures idempotency
INSERT INTO wallets (address, balance)
VALUES ('0x0000000000000000000000000000000000000000', 1000000)
    ON CONFLICT (address) DO NOTHING;