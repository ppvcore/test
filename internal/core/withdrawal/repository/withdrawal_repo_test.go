package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"test/internal/core/withdrawal/entity"
	"test/testhelpers"
)

func TestWithdrawalRepo_CreateIdempotent(t *testing.T) {
	pool := testhelpers.NewTestDB(t)
	repo := NewWithdrawalRepo()
	ctx := context.Background()

	idempKey := "idem-" + uuid.NewString()
	now := time.Now().UTC().Truncate(time.Second)

	w1 := &entity.Withdrawal{
		ID:             uuid.NewString(),
		UserID:         "user-123",
		Amount:         10,
		Currency:       "USDT",
		Destination:    "0xabc123",
		IdempotencyKey: idempKey,
		Status:         entity.WithdrawalPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	t.Run("first insert succeeds", func(t *testing.T) {
		created, existing, err := repo.CreateIdempotent(ctx, pool, w1)
		require.NoError(t, err)
		assert.True(t, created)
		assert.Nil(t, existing)
	})

	t.Run("second insert same key returns existing", func(t *testing.T) {
		created, existing, err := repo.CreateIdempotent(ctx, pool, w1)
		require.NoError(t, err)
		assert.False(t, created)
		require.NotNil(t, existing)
		assert.Equal(t, w1.ID, existing.ID)
		assert.Equal(t, idempKey, existing.IdempotencyKey)
	})

	t.Run("different payload same key returns existing", func(t *testing.T) {
		w2 := *w1
		w2.ID = uuid.NewString()
		w2.Amount = 5

		created, existing, err := repo.CreateIdempotent(ctx, pool, &w2)
		require.NoError(t, err)
		assert.False(t, created)
		require.NotNil(t, existing)
		assert.NotEqual(t, w2.Amount, existing.Amount)
	})
}
