package repositories

import (
	"context"
	"fmt"
	"nerdmoney/pkg/accounts/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type BankConnectionRepository struct {
	pool *pgxpool.Pool
	log  echo.Logger
}

func NewBankConnectionRepository(pool *pgxpool.Pool, log echo.Logger) BankConnectionRepository {
	return BankConnectionRepository{pool, log}
}

func (r *BankConnectionRepository) Save(writeModel models.BankConnectionWriteModel) (models.BankConnection, error) {
	r.log.Debugf("Attempting to save a new BankConnection: %+v", writeModel)

	query := `
        INSERT INTO bank_connection (plaid_item_id, access_token, consent_expiration_time, login_required) 
        VALUES ($1, $2, $3, $4) 
        RETURNING id, plaid_item_id, access_token, consent_expiration_time, login_required`

	var savedConnection models.BankConnection

	err := r.pool.QueryRow(
		context.Background(),
		query,
		writeModel.PlaidItemID,
		writeModel.AccessToken,
		writeModel.ConsentExpirationTimestamp,
		writeModel.LoginRequired,
	).Scan(
		&savedConnection.ID,
		&savedConnection.PlaidItemID,
		&savedConnection.AccessToken,
		&savedConnection.ConsentExpirationTimestamp,
		&savedConnection.LoginRequired,
	)

	if err != nil {
		return models.BankConnection{}, fmt.Errorf("Failed to save a new BankConnection: %w", err)
	}

	return savedConnection, nil
}
