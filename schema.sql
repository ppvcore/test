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