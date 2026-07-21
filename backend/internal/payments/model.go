package payments

import "time"

type Payment struct {
	ID            string    `json:"id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CustomerEmail string    `json:"customerEmail,omitempty"`
	PaymentMethod string    `json:"paymentMethod,omitempty"`
	LastError     string    `json:"lastError,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	LastEventID   string    `json:"lastEventId,omitempty"`
	LastEventType string    `json:"lastEventType,omitempty"`
}
