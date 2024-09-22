-- +migrate Up
-- nit(reddec): IF NOT EXISTS left for previous compatibility when migrations were not yet used
CREATE TABLE IF NOT EXISTS data
(
    id           TEXT NOT NULL,
    type         TEXT NOT NULL,
    fileName     TEXT NOT NULL,
    filePath     TEXT NOT NULL,
    burn         TEXT NOT NULL,
    expire       TEXT NOT NULL,
    passwordHash TEXT NOT NULL,
    passwordSalt TEXT NOT NULL,
    encryptSalt  TEXT NOT NULL
);