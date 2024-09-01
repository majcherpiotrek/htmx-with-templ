package transactions

import (
	"time"

	"github.com/shopspring/decimal"
)

type DbTransaction struct {
	ID                 int64
	PlaidTransactionID string
	BankAccountID      int
	Amount             decimal.Decimal
	Currency           string
	DateAuthorized     time.Time
	DateTimeAuthorized *time.Time
	DatePosted         time.Time
	DateTimePosted     *time.Time
	NextCursor         string
}

type DbTransactionWriteModel struct {
	PlaidTransactionID string
	PlaidAccountID     string
	Amount             decimal.Decimal
	Currency           string
	DateAuthorized     time.Time
	DateTimeAuthorized *time.Time
	DatePosted         time.Time
	DateTimePosted     *time.Time
	NextCursor         string
}
