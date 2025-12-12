-- init.sql
CREATE DATABASE walletsdb;

CREATE TABLE IF NOT EXISTS wallets (
    uuid UUID PRIMARY KEY,
    balance DECIMAL NOT NULL DEFAULT 0
);

CREATE INDEX idx_wallets_uuid ON wallets (uuid);