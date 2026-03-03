package entity

import (
	"time"
)

type WithdrawalStatus string

const (
	WithdrawalPending   WithdrawalStatus = "pending"
	WithdrawalConfirmed WithdrawalStatus = "confirmed"
	WithdrawalRejected  WithdrawalStatus = "rejected"
	WithdrawalFailed    WithdrawalStatus = "failed"
)

type Withdrawal struct {
	ID             string           `json:"id"`
	UserID         string           `json:"user_id"`
	Amount         int64            `json:"amount"`
	Currency       string           `json:"currency"`
	Destination    string           `json:"destination"`
	IdempotencyKey string           `json:"idempotency_key"`
	Status         WithdrawalStatus `json:"status"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at,omitempty"`
}
