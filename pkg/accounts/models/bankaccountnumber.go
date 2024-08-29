package models

type BankAccountNumber struct {
	ID                int
	BankAccountId     int
	AccountNumberType string
	Account           *string
	Routing           *string
	WireRouting       *string
	Institution       *string
	Branch            *string
	Bic               *string
	Iban              *string
	SortCode          *string
}
