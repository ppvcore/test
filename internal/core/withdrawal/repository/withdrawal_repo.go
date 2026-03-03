package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"test/internal/core/withdrawal/entity"
	"test/internal/postgres"
)

var (
	ErrNotFound            = errors.New("withdrawal not found")
	ErrIdempotencyConflict = errors.New("idempotency key already used")
	ErrDuplicateKey        = errors.New("duplicate idempotency key")
)

type WithdrawalRepo interface {
	Create(ctx context.Context, q postgres.Querier, w *entity.Withdrawal) error
	GetByID(ctx context.Context, q postgres.Querier, id string) (*entity.Withdrawal, error)
	GetByIdempotencyKey(ctx context.Context, q postgres.Querier, key string) (*entity.Withdrawal, error)
	CreateIdempotent(ctx context.Context, q postgres.Querier, w *entity.Withdrawal) (created bool, existing *entity.Withdrawal, err error)
}

type withdrawalRepo struct{}

func NewWithdrawalRepo() WithdrawalRepo {
	return &withdrawalRepo{}
}

func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (r *withdrawalRepo) Create(ctx context.Context, q postgres.Querier, w *entity.Withdrawal) error {
	_, err := q.Exec(ctx,
		`INSERT INTO withdrawals (
			id,              user_id,     amount,     currency, 
			destination,     idempotency_key, 
			status,          created_at,  updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		w.ID, w.UserID, w.Amount, w.Currency,
		w.Destination, w.IdempotencyKey,
		w.Status, w.CreatedAt, w.UpdatedAt,
	)

	if err != nil {
		if isPgUniqueViolation(err) {
			return ErrIdempotencyConflict
		}
		return fmt.Errorf("failed to create withdrawal: %w", err)
	}

	return nil
}

func (r *withdrawalRepo) GetByID(ctx context.Context, q postgres.Querier, id string) (*entity.Withdrawal, error) {
	w := &entity.Withdrawal{}
	err := q.QueryRow(ctx,
		`SELECT 
			id, user_id, amount, currency, destination, 
			idempotency_key, status, created_at, updated_at
		 FROM withdrawals 
		 WHERE id = $1`,
		id,
	).Scan(
		&w.ID, &w.UserID, &w.Amount, &w.Currency, &w.Destination,
		&w.IdempotencyKey, &w.Status, &w.CreatedAt, &w.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get withdrawal by id: %w", err)
	}

	return w, nil
}

func (r *withdrawalRepo) GetByIdempotencyKey(ctx context.Context, q postgres.Querier, key string) (*entity.Withdrawal, error) {
	if key == "" {
		return nil, nil
	}

	w := &entity.Withdrawal{}
	err := q.QueryRow(ctx,
		`SELECT 
			id, user_id, amount, currency, destination, 
			idempotency_key, status, created_at, updated_at
		 FROM withdrawals 
		 WHERE idempotency_key = $1`,
		key,
	).Scan(
		&w.ID, &w.UserID, &w.Amount, &w.Currency, &w.Destination,
		&w.IdempotencyKey, &w.Status, &w.CreatedAt, &w.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get withdrawal by idempotency key: %w", err)
	}

	return w, nil
}
func (r *withdrawalRepo) CreateIdempotent(
	ctx context.Context,
	q postgres.Querier,
	w *entity.Withdrawal,
) (created bool, existing *entity.Withdrawal, err error) {
	row := q.QueryRow(ctx,
		`INSERT INTO withdrawals (
			id, user_id, amount, currency, destination,
			idempotency_key, status, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id, user_id, amount, currency, destination,
		          idempotency_key, status, created_at, updated_at`,
		w.ID, w.UserID, w.Amount, w.Currency,
		w.Destination, w.IdempotencyKey,
		w.Status, w.CreatedAt, w.UpdatedAt,
	)

	existing = &entity.Withdrawal{}
	err = row.Scan(
		&existing.ID, &existing.UserID, &existing.Amount, &existing.Currency,
		&existing.Destination, &existing.IdempotencyKey,
		&existing.Status, &existing.CreatedAt, &existing.UpdatedAt,
	)
	if err == nil {
		return true, nil, nil
	}

	if errors.Is(err, pgx.ErrNoRows) || isPgUniqueViolation(err) {
		existing, err = r.GetByIdempotencyKey(ctx, q, w.IdempotencyKey)
		if err != nil {
			return false, nil, fmt.Errorf("failed to get existing withdrawal: %w", err)
		}
		if existing == nil {
			return false, nil, errors.New("idempotency key conflict but record disappeared")
		}
		return false, existing, nil
	}

	return false, nil, fmt.Errorf("insert idempotent failed: %w", err)
}
