package repositories

import (
	"context"
	"fmt"
	"nerdmoney/banking/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type BankAccountRepository struct {
	pool *pgxpool.Pool
	log  echo.Logger
}

func NewBankAccountRepository(pool *pgxpool.Pool, log echo.Logger) BankAccountRepository {
	return BankAccountRepository{pool, log}
}

func (r *BankAccountRepository) Save(writeModel models.BankAccountWriteModel) (models.BankAccount, error) {
	r.log.Debugf("Attempting to save a new BankAccount: %+v", writeModel)

	query := `
	INSERT INTO bank_account (plaid_account_id, bank_connection_id, name, mask, account_type) 
	VALUES ($1, $2, $3, $4, $5) 
	RETURNING id, plaid_account_id, bank_connection_id, name, mask, account_type`

	var id int
	var plaidAccountID string
	var bankConnectionID int
	var name string
	var mask *string
	var accountTypeStr string

	err := r.pool.QueryRow(
		context.Background(),
		query,
		writeModel.PlaidAccountId,
		writeModel.BankConnectionID,
		writeModel.Name,
		writeModel.Mask,
		writeModel.AccountType,
	).Scan(
		&id,
		&plaidAccountID,
		&bankConnectionID,
		&name,
		&mask,
		&accountTypeStr,
	)

	if err != nil {
		return models.BankAccount{}, fmt.Errorf("Failed to save new BankAccount: %w", err)
	}

	accountType, err := models.ParseAccountType(accountTypeStr)

	if err != nil {
		return models.BankAccount{}, fmt.Errorf("Failed to save new BankAccount: %w", err)
	}

	r.log.Debugf("Saved new BankAccount with id='%d'", id)

	return models.BankAccount{
		ID:               id,
		PlaidAccountId:   plaidAccountID,
		BankConnectionID: bankConnectionID,
		Name:             name,
		Mask:             mask,
		AccountType:      accountType,
	}, nil
}
