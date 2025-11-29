--From "recruitment task":
--"Initially, there is only one wallet (...) holding 1,000,000 BTP tokens".
-- Init wallets table
CREATE TABLE IF NOT EXISTS wallets (
                                       address VARCHAR(255) PRIMARY KEY,
    -- No requirements about balance size.
    -- If preferable INT32 may be changed for BIGINT
    balance INT32 NOT NULL CHECK (balance >= 0)
    );
