package models

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type BankAccount struct {
	ID               int
	PlaidAccountId   string
	BankConnectionID int
	Name             string
	Mask             *string
	AccountType      AccountType
	CurrentBalance   decimal.NullDecimal
	AvailableBalance decimal.NullDecimal
	Currency         string
}

type BankAccountWriteModel struct {
	PlaidAccountId   string
	BankConnectionID int
	Name             string
	Mask             *string
	AccountType      string // TODO: maybe this should be checked before writing
	CurrentBalance   decimal.NullDecimal
	AvailableBalance decimal.NullDecimal
	Currency         string
}

type AccountType string

const (
	Investment AccountType = "investment"
	Credit     AccountType = "credit"
	Depository AccountType = "depository"
	Loan       AccountType = "loan"
	Brokerage  AccountType = "brokerage"
	Other      AccountType = "other"
)

func ParseAccountType(source string) (AccountType, error) {
	switch source {
	case string(Investment):
		return Investment, nil
	case string(Credit):
		return Credit, nil
	case string(Depository):
		return Depository, nil
	case string(Loan):
		return Loan, nil
	case string(Brokerage):
		return Brokerage, nil
	case string(Other):
		return Other, nil
	default:
		return "", fmt.Errorf("Invalid AccountType: '%s'", source)
	}
}
