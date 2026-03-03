package service

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bRepo "test/internal/core/balances/repository"
	"test/internal/core/withdrawal/dto"
	"test/internal/core/withdrawal/entity"
	wRepo "test/internal/core/withdrawal/repository"
	"test/testhelpers"
)

func TestWithdrawalService_Create(t *testing.T) {
	pool := testhelpers.NewTestDB(t)
	wRepo := wRepo.NewWithdrawalRepo()
	bRepo := bRepo.NewBalanceRepo()

	srv := NewWithdrawalService(wRepo, bRepo, pool)
	ctx := context.Background()

	testhelpers.CreateBalance(t, pool, "user-123", "USDT", 100)

	t.Run("success + idempotency repeat", func(t *testing.T) {
		idem := "idem-" + uuid.NewString()
		req := dto.Request{
			UserID:         "user-123",
			Amount:         5,
			Currency:       "USDT",
			Destination:    "0xdef456",
			IdempotencyKey: idem,
		}

		w, created, err := srv.Create(ctx, req)
		require.NoError(t, err)
		assert.True(t, created)
		assert.Equal(t, entity.WithdrawalPending, w.Status)

		w2, created2, err := srv.Create(ctx, req)
		require.NoError(t, err)
		assert.False(t, created2)
		assert.Equal(t, w.ID, w2.ID)
	})

	t.Run("different payload same key returns error", func(t *testing.T) {
		idem := "idem-conflict-" + uuid.NewString()
		req1 := dto.Request{
			UserID:         "user-123",
			Amount:         10,
			Currency:       "USDT",
			Destination:    "0xabc",
			IdempotencyKey: idem,
		}
		_, _, err := srv.Create(ctx, req1)
		require.NoError(t, err)

		req2 := req1
		req2.Amount = 20

		_, _, err = srv.Create(ctx, req2)
		assert.ErrorIs(t, err, ErrInvalidPayload)
	})

	t.Run("negative amount returns error", func(t *testing.T) {
		req := dto.Request{
			UserID:         "user-123",
			Amount:         -10,
			Currency:       "USDT",
			IdempotencyKey: "idem-neg",
		}
		_, _, err := srv.Create(ctx, req)
		assert.ErrorIs(t, err, ErrInvalidAmount)
	})

	t.Run("insufficient balance returns error", func(t *testing.T) {
		req := dto.Request{
			UserID:         "user-123",
			Amount:         1000,
			Currency:       "USDT",
			IdempotencyKey: "idem-insuff",
		}
		_, _, err := srv.Create(ctx, req)
		assert.ErrorIs(t, err, ErrInsufficientBalance)
	})
}

func TestWithdrawalService_ConcurrentSameIdempotencyKey(t *testing.T) {
	pool := testhelpers.NewTestDB(t)
	wRepo := wRepo.NewWithdrawalRepo()
	bRepo := bRepo.NewBalanceRepo()

	srv := NewWithdrawalService(wRepo, bRepo, pool)

	const goroutines = 10
	const amount = 100
	const userID = "user-conc"
	const currency = "USDT"
	idemKey := "idem-concurrent-test"

	testhelpers.CreateBalance(t, pool, userID, currency, amount*int64(goroutines))

	errCh := make(chan error, goroutines)
	successCount := 0
	var mu sync.Mutex

	req := dto.Request{
		UserID:         userID,
		Amount:         amount,
		Currency:       currency,
		Destination:    "0xconcurrent",
		IdempotencyKey: idemKey,
	}

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w, created, err := srv.Create(context.Background(), req)
			if err != nil {
				errCh <- err
				return
			}
			if created {
				mu.Lock()
				successCount++
				mu.Unlock()
			} else {
				if w.Amount != amount {
					errCh <- fmt.Errorf("wrong amount in existing record: got %d", w.Amount)
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("goroutine error: %v", err)
	}

	assert.Equal(t, 1, successCount)

	var available int64
	err := pool.QueryRow(context.Background(),
		"SELECT available FROM balances WHERE user_id = $1 AND currency = $2",
		userID, currency,
	).Scan(&available)
	require.NoError(t, err)
	expectedAvailable := int64(amount * (goroutines - 1))
	assert.Equal(t, expectedAvailable, available)
}
