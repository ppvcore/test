package entity

import "time"

type Balance struct {
	UserID    string    `db:"user_id"`
	Currency  string    `db:"currency"`
	Available int64     `db:"available"`
	Locked    int64     `db:"locked"`
	UpdatedAt time.Time `db:"updated_at"`
}
