-- Active: 1765509233899@@127.0.0.1@5432@walletsdb
CREATE TABLE IF NOT EXISTS wallets (
    uuid UUID PRIMARY KEY,
    balance DECIMAL NOT NULL DEFAULT 0
);

CREATE INDEX idx_wallets_uuid ON wallets (uuid);