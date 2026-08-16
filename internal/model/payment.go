package model

import "time"

type PaymentRecord struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	TaskID        string     `json:"task_id"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	Method        string     `json:"method"`
	Status        string     `json:"status"`
	TransactionID string     `json:"transaction_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
}
