package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"test/internal/core/balances/entity"
	"test/internal/postgres"
)

var (
	ErrBalanceNotFound   = errors.New("balance not found")
	ErrInsufficientFunds = errors.New("insufficient balance")
)

type BalanceRepo interface {
	GetForUpdate(ctx context.Context, q postgres.Querier, userID, currency string) (*entity.Balance, error)
	DecreaseAvailable(ctx context.Context, q postgres.Querier, userID, currency string, amount int64) error
}

type balanceRepo struct{}

func NewBalanceRepo() BalanceRepo {
	return &balanceRepo{}
}

func (r *balanceRepo) GetForUpdate(ctx context.Context, q postgres.Querier, userID, currency string) (*entity.Balance, error) {
	b := &entity.Balance{}
	err := q.QueryRow(ctx,
		`SELECT user_id, currency, available, locked, updated_at
		 FROM balances
		 WHERE user_id = $1 AND currency = $2
		 FOR UPDATE`,
		userID, currency,
	).Scan(&b.UserID, &b.Currency, &b.Available, &b.Locked, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBalanceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("cannot get balance: %w", err)
	}
	return b, nil
}
func (r *balanceRepo) DecreaseAvailable(ctx context.Context, q postgres.Querier, userID, currency string, amount int64) error {
	res, err := q.Exec(ctx,
		`UPDATE balances
		 SET available = available - $1, updated_at = $2
		 WHERE user_id = $3 AND currency = $4 AND available >= $1`,
		amount, time.Now().UTC(), userID, currency,
	)
	if err != nil {
		return fmt.Errorf("cannot decrease balance: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrInsufficientFunds
	}
	return nil
}
