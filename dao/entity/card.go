package entity

import "time"

type Card struct {
	ID              string
	UserID          string
	AccountID       string
	PANEncrypted    string
	ExpiryEncrypted string
	CVVHash         string
	IntegrityHMAC   string
	Last4           string
	CreatedAt       time.Time
}
