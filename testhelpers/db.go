package testhelpers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	adminURL := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	adminPool, err := pgxpool.New(context.Background(), adminURL)
	require.NoError(t, err)

	dbName := "test_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
	_, err = adminPool.Exec(context.Background(), fmt.Sprintf(`CREATE DATABASE "%s"`, dbName))
	require.NoError(t, err)

	testDBURL := fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable", dbName)
	pool, err := pgxpool.New(context.Background(), testDBURL)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
		_, _ = adminPool.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
		adminPool.Close()
	})

	err = applyEmbeddedSchema(pool)
	require.NoError(t, err)

	return pool
}

func applyEmbeddedSchema(pool *pgxpool.Pool) error {
	schema := `
CREATE TABLE users (
    id          SERIAL PRIMARY KEY,
    user_id     VARCHAR(255) UNIQUE NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE balances (
    id          SERIAL PRIMARY KEY,
    user_id     VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    currency    VARCHAR(10)  NOT NULL DEFAULT 'USDT',
    available   BIGINT       NOT NULL DEFAULT 0 CHECK (available >= 0),
    locked      BIGINT       NOT NULL DEFAULT 0 CHECK (locked >= 0),
    updated_at  TIMESTAMPTZ  DEFAULT NOW(),
    CONSTRAINT  unique_user_currency UNIQUE (user_id, currency)
);

CREATE TABLE withdrawals (
    id              VARCHAR(36) PRIMARY KEY,
    user_id         VARCHAR(255)    NOT NULL REFERENCES users(user_id),
    amount          BIGINT          NOT NULL CHECK (amount > 0),
    currency        VARCHAR(10)     NOT NULL DEFAULT 'USDT',
    destination     TEXT            NOT NULL,
    idempotency_key VARCHAR(255)    NOT NULL UNIQUE,
    status          VARCHAR(20)     NOT NULL 
        CHECK (status IN ('pending', 'confirmed', 'rejected', 'failed')),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_user_idempotency UNIQUE (user_id, idempotency_key),
    CONSTRAINT chk_currency_usdt CHECK (currency = 'USDT')
);

CREATE INDEX idx_withdrawals_user_id    ON withdrawals(user_id);
CREATE INDEX idx_withdrawals_status     ON withdrawals(status);
CREATE INDEX idx_withdrawals_created_at ON withdrawals(created_at);

INSERT INTO users (user_id) 
VALUES ('user-123')
ON CONFLICT DO NOTHING;

INSERT INTO balances (user_id, currency, available, locked, updated_at)
VALUES ('user-123', 'USDT', 10, 0, NOW())
ON CONFLICT (user_id, currency) DO NOTHING;
`
	_, err := pool.Exec(context.Background(), schema)
	return err
}

func CreateBalance(t *testing.T, pool *pgxpool.Pool, userID, currency string, amount int64) {
	t.Helper()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO balances (user_id, currency, available) 
         VALUES ($1, $2, $3) 
         ON CONFLICT (user_id, currency) DO UPDATE SET available = EXCLUDED.available`,
		userID, currency, amount)
	require.NoError(t, err)
}
