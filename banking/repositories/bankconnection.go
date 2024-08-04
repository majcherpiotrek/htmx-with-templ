package repositories

import (
	"context"
	"htmx-with-templ/banking/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BankConnectionRepository struct {
	pool *pgxpool.Pool
}

func NewBankConnectionRepository(pool *pgxpool.Pool) BankConnectionRepository {
	return BankConnectionRepository{pool}
}

func (r *BankConnectionRepository) Save(plaidItemID string, accessToken string) (models.BankConnection, error) {
	query := `
        INSERT INTO bank_connection (plaid_item_id, access_token) 
        VALUES ($1, $2) 
        RETURNING id`

	var savedConnection models.BankConnection

	err := r.pool.QueryRow(
		context.Background(),
		query,
		plaidItemID,
		accessToken,
	).Scan(
		&savedConnection.ID,
		&savedConnection.PlaidItemID,
		&savedConnection.AccessToken,
	)

	if err != nil {
		return models.BankConnection{}, nil
	}

	return savedConnection, nil
}
