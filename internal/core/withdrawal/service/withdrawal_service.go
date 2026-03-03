package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	bRepo "test/internal/core/balances/repository"
	"test/internal/core/withdrawal/dto"
	"test/internal/core/withdrawal/entity"
	wRepo "test/internal/core/withdrawal/repository"
)

var (
	ErrInvalidAmount       = errors.New("amount must be positive")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidPayload      = errors.New("idempotency key exists but payload differs")
)

type WithdrawalService struct {
	wRepo wRepo.WithdrawalRepo
	bRepo bRepo.BalanceRepo
	pool  *pgxpool.Pool
}

func NewWithdrawalService(wRepo wRepo.WithdrawalRepo, bRepo bRepo.BalanceRepo, pool *pgxpool.Pool) *WithdrawalService {
	return &WithdrawalService{
		wRepo: wRepo,
		bRepo: bRepo,
		pool:  pool,
	}
}

func (s *WithdrawalService) Create(ctx context.Context, req dto.Request) (*entity.Withdrawal, bool, error) {
	if req.Amount <= 0 {
		return nil, false, ErrInvalidAmount
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	w := &entity.Withdrawal{
		ID:             uuid.NewString(),
		UserID:         req.UserID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Destination:    req.Destination,
		IdempotencyKey: req.IdempotencyKey,
		Status:         entity.WithdrawalPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	created, existing, err := s.wRepo.CreateIdempotent(ctx, tx, w)
	if err != nil {
		return nil, false, err
	}

	if !created {
		if !payloadMatches(existing, req) {
			return nil, false, ErrInvalidPayload
		}
		tx.Commit(ctx)
		return existing, false, nil
	}

	balance, err := s.bRepo.GetForUpdate(ctx, tx, req.UserID, req.Currency)
	if err != nil {
		return nil, false, err
	}

	if balance.Available < req.Amount {
		return nil, false, ErrInsufficientBalance
	}

	if err := s.bRepo.DecreaseAvailable(ctx, tx, req.UserID, req.Currency, req.Amount); err != nil {
		return nil, false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}

	return w, true, nil
}

func (s *WithdrawalService) GetByID(ctx context.Context, id string) (*entity.Withdrawal, error) {
	return s.wRepo.GetByID(ctx, s.pool, id)
}

func payloadMatches(w *entity.Withdrawal, req dto.Request) bool {
	return w.UserID == req.UserID &&
		w.Amount == req.Amount &&
		w.Destination == req.Destination &&
		w.Currency == req.Currency
}
