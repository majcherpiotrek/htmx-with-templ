package accounts

import (
	"context"
	"fmt"
	"nerdmoney/pkg/accounts/models"
	"nerdmoney/pkg/accounts/repositories"
	"nerdmoney/pkg/banking"
	"nerdmoney/pkg/common/layout"
	"nerdmoney/pkg/common/utils"
	"time"

	"github.com/labstack/echo/v4"
)

func RegisterAccountRoutes(e *echo.Echo, plaidClient *banking.PlaidClient, bankConnectionRepostiory repositories.BankConnectionRepository, bankAccountRepository repositories.BankAccountRepository) {

	log := e.Logger

	e.GET("/bank-accounts", func(c echo.Context) error {

		bankAccounts, err := bankAccountRepository.ListAll()

		if err != nil {
			log.Errorf("Failed to list bank accounts: %w", err)
			return c.String(500, "Something went wrong when listing bank accounts...")
		}

		return layout.RenderComponent(
			c,
			200,
			BankAccountList(bankAccounts),
		)
	})

	e.POST("/banks", func(c echo.Context) error {
		publicToken := c.FormValue("publicToken")

		if len(publicToken) < 1 {
			return c.String(400, "'publicToken' missing in the request")
		}

		itemAccessToken, err := plaidClient.GetAccessToken(publicToken)

		if err != nil {
			return c.String(500, "Failed to get item access token from Plaid")
		}

		authGetResponse, err := plaidClient.AuthGet(itemAccessToken.AccessToken)

		if err != nil {
			return c.String(500, "Failed to get account data from Plaid")
		}

		e.Logger.Info("Auth get response", authGetResponse.Item)

		bankConnectionWriteModel := models.BankConnectionWriteModel{
			PlaidItemID:                itemAccessToken.ItemId,
			AccessToken:                itemAccessToken.AccessToken,
			ConsentExpirationTimestamp: authGetResponse.Item.ConsentExpirationTime.Get(),
			LoginRequired:              false,
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)

		defer cancel()

		e.Logger.Debugf("Starting transaction...")
		tx, err := bankConnectionRepostiory.DbPool().Begin(ctx)

		if err != nil {
			e.Logger.Errorf("Failed to start transaction: %w", err)
			return c.String(500, fmt.Sprintf("Failed to start transaction: %+v", err))
		}

		bankConnection, err := bankConnectionRepostiory.Save(bankConnectionWriteModel)

		if err != nil {
			tx.Rollback(ctx)
			e.Logger.Errorf("Failed to save bank connection: %w", err)
			return c.String(500, fmt.Sprintf("Failed to save bank connection: %+v", err))
		}

		var bankAccounts []models.BankAccount

		for _, plaidAccount := range authGetResponse.Accounts {

			var currency string

			if plaidAccount.Balances.IsoCurrencyCode.IsSet() {
				currency = *plaidAccount.Balances.IsoCurrencyCode.Get()
			} else if plaidAccount.Balances.UnofficialCurrencyCode.IsSet() {
				currency = *plaidAccount.Balances.UnofficialCurrencyCode.Get()
			} else {
				// Should never happen as Plaid states in the docs that one of the two is alway defined: https://plaid.com/docs/api/products/auth/#authget
				currency = "Unknown"
			}

			accountWriteModel := models.BankAccountWriteModel{
				PlaidAccountId:   plaidAccount.AccountId,
				BankConnectionID: bankConnection.ID,
				Name:             plaidAccount.Name,
				Mask:             plaidAccount.Mask.Get(),
				AccountType:      string(plaidAccount.Type),
				CurrentBalance:   utils.NullDecimalFromFloat64(plaidAccount.Balances.Current.Get()),
				AvailableBalance: utils.NullDecimalFromFloat64(plaidAccount.Balances.Available.Get()),
				Currency:         currency,
			}

			savedBankAccount, err := bankAccountRepository.Save(accountWriteModel)

			if err != nil {
				tx.Rollback(ctx)
				e.Logger.Errorf("Failed to save Plaid Account - %+v. Error was: %w", accountWriteModel, err)
				return c.String(500, fmt.Sprintf("Failed to save Plaid Account - %+v. Error was: %+v", accountWriteModel, err))
			}

			bankAccounts = append(bankAccounts, savedBankAccount)
		}

		e.Logger.Debugf("Comitting transaction...")
		err = tx.Commit(ctx)

		if err != nil {
			tx.Rollback(ctx)
			e.Logger.Errorf("Failed to commit transaction: %w", err)
			return c.String(500, fmt.Sprintf("Failed to commit transaction: %+v", err))
		}

		e.Logger.Infof("Successfully saved new bank connection with %d accounts", len(bankAccounts))

		return layout.RenderComponent(
			c,
			200,
			AddAcounts(bankAccounts),
		)
	})
}
