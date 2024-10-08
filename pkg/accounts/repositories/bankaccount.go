package repositories

import (
	"context"
	"fmt"
	"nerdmoney/pkg/accounts/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

type BankAccountRepository interface {
	ListAll() ([]models.BankAccount, error)
	Save(models.BankAccountWriteModel) (models.BankAccount, error)
	DbPool() *pgxpool.Pool
}

type bankAccountRepositoryImpl struct {
	pool *pgxpool.Pool
	log  echo.Logger
}

func NewBankAccountRepository(pool *pgxpool.Pool, log echo.Logger) BankAccountRepository {
	return &bankAccountRepositoryImpl{pool, log}
}

func (r *bankAccountRepositoryImpl) DbPool() *pgxpool.Pool {
	return r.pool
}

func (r *bankAccountRepositoryImpl) ListAll() ([]models.BankAccount, error) {
	r.log.Debugf("Attempting to list all bank accounts")

	query := `SELECT * FROM bank_account`

	rows, err := r.pool.Query(context.Background(), query)

	if err != nil {
		return []models.BankAccount{}, fmt.Errorf("Failed to list all bank accounts: %w", err)
	}

	var allAccounts []models.BankAccount

	for rows.Next() {
		var bankAccount = models.BankAccount{}
		var accountTypeStr string

		rows.Scan(
			&bankAccount.ID,
			&bankAccount.PlaidAccountId,
			&bankAccount.BankConnectionID,
			&bankAccount.Name,
			&bankAccount.Mask,
			&accountTypeStr,
			&bankAccount.CurrentBalance,
			&bankAccount.AvailableBalance,
			&bankAccount.Currency,
		)

		accountType, err := models.ParseAccountType(accountTypeStr)

		if err != nil {
			return []models.BankAccount{}, fmt.Errorf("Failed to parse bank account type for bank account with ID='%d': %w", bankAccount.ID, err)
		}

		bankAccount.AccountType = accountType
		allAccounts = append(allAccounts, bankAccount)
	}

	if err := rows.Err(); err != nil {
		return []models.BankAccount{}, fmt.Errorf("Failed to read rows when trying to list all bank accounts: %w", err)
	}

	return allAccounts, nil
}

func (r *bankAccountRepositoryImpl) Save(writeModel models.BankAccountWriteModel) (models.BankAccount, error) {
	r.log.Debugf("Attempting to save a new BankAccount: %+v", writeModel)

	query := `
	INSERT INTO bank_account (plaid_account_id, bank_connection_id, name, mask, account_type, current_balance, available_balance, currency) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
	RETURNING id, plaid_account_id, bank_connection_id, name, mask, account_type, current_balance, available_balance, currency`

	var id int
	var plaidAccountID string
	var bankConnectionID int
	var name string
	var mask *string
	var accountTypeStr string
	var currentBalance decimal.NullDecimal
	var availableBalance decimal.NullDecimal
	var currency string

	err := r.pool.QueryRow(
		context.Background(),
		query,
		writeModel.PlaidAccountId,
		writeModel.BankConnectionID,
		writeModel.Name,
		writeModel.Mask,
		writeModel.AccountType,
		writeModel.CurrentBalance,
		writeModel.AvailableBalance,
		writeModel.Currency,
	).Scan(
		&id,
		&plaidAccountID,
		&bankConnectionID,
		&name,
		&mask,
		&accountTypeStr,
		&currentBalance,
		&availableBalance,
		&currency,
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
