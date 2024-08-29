package models

import "time"

type BankConnection struct {
	ID                         int
	PlaidItemID                string
	AccessToken                string
	ConsentExpirationTimestamp *time.Time
	LoginRequired              bool
}

type BankConnectionWriteModel struct {
	PlaidItemID                string
	AccessToken                string
	ConsentExpirationTimestamp *time.Time
	LoginRequired              bool
}
