package dto

type Request struct {
	UserID         string `json:"user_id"`
	Amount         int64  `json:"amount"`
	Destination    string `json:"destination"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotency_key"`
}
