package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"nerdmoney/banking"
	"nerdmoney/banking/models"
	bankingRepositories "nerdmoney/banking/repositories"

	"nerdmoney/view/components"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

var (
	PLAID_CLIENT_ID     = ""
	PLAID_SECRET        = ""
	PLAID_ENV           = ""
	PLAID_PRODUCTS      = ""
	PLAID_COUNTRY_CODES = ""
	PLAID_REDIRECT_URI  = ""
	DATABASE_URL        = ""
)

func renderComponent(ctx echo.Context, status int, t templ.Component) error {
	ctx.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	ctx.Response().Writer.WriteHeader(status)

	err := t.Render(ctx.Request().Context(), ctx.Response().Writer)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "failed to render response template")
	}

	return nil
}

func renderPage(ctx echo.Context, status int, pageContent templ.Component) error {
	return renderComponent(ctx, status, components.Index(pageContent))
}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)
	e.Static("/assets", "assets")

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		e.Logger.Fatalf("Error loading .env file: %v", err)
	}

	// Assign environment variables to the corresponding variables
	PLAID_CLIENT_ID = os.Getenv("PLAID_CLIENT_ID")
	PLAID_SECRET = os.Getenv("PLAID_SECRET")
	PLAID_ENV = os.Getenv("PLAID_ENV")
	PLAID_PRODUCTS = os.Getenv("PLAID_PRODUCTS")
	PLAID_COUNTRY_CODES = os.Getenv("PLAID_COUNTRY_CODES")
	PLAID_REDIRECT_URI = os.Getenv("PLAID_REDIRECT_URI")
	DATABASE_URL = os.Getenv("DATABASE_URL")

	dbpool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		e.Logger.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	err = runMigrations(dbpool, e.Logger)
	if err != nil {
		e.Logger.Fatalf("Faied to run migrations: %v\n", err)
	}

	plaidClient, err := banking.NewPlaidClient(banking.PlaidClientConfig{
		ClientId:     PLAID_CLIENT_ID,
		Secret:       PLAID_SECRET,
		Env:          PLAID_ENV,
		Products:     PLAID_PRODUCTS,
		CountryCodes: PLAID_COUNTRY_CODES,
		RedirectUri:  PLAID_REDIRECT_URI,
	})

	if err != nil {
		e.Logger.Fatalf("Error initializing PlaidClient: %v", err)
	}

	bankConnectionRepository := bankingRepositories.NewBankConnectionRepository(dbpool, e.Logger)
	bankAccountRepository := bankingRepositories.NewBankAccountRepository(dbpool, e.Logger)

	linkToken := ""

	e.GET("/", func(c echo.Context) error {
		linkTokenResponse, err := plaidClient.CreateLinkToken()
		if err != nil {
			return c.String(500, "Something went wrong")
		}

		linkToken = linkTokenResponse.LinkToken

		return renderComponent(
			c,
			200,
			components.Index(components.MainPage(linkTokenResponse.LinkToken)),
		)
	})

	e.GET("/bank-accounts", func(c echo.Context) error {

		bankAccounts, err := bankAccountRepository.ListAll()

		if err != nil {
			log.Errorf("Failed to list bank accounts: %w", err)
			return c.String(500, "Something went wrong when listing bank accounts...")
		}

		accountNames := make([]string, len(bankAccounts))

		for _, bankAccount := range bankAccounts {
			accountNames = append(accountNames, bankAccount.Name)
		}

		log.Infof("Account names: %v", accountNames)

		return renderComponent(
			c,
			200,
			components.BankAccountList(accountNames),
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
		tx, err := dbpool.Begin(ctx)

		if err != nil {
			e.Logger.Errorf("Failed to start transaction: %w", err)
			return c.String(500, fmt.Sprintf("Failed to start transaction: %+v", err))
		}

		bankConnection, err := bankConnectionRepository.Save(bankConnectionWriteModel)

		if err != nil {
			tx.Rollback(ctx)
			e.Logger.Errorf("Failed to save bank connection: %w", err)
			return c.String(500, fmt.Sprintf("Failed to save bank connection: %+v", err))
		}

		for _, plaidAccount := range authGetResponse.Accounts {
			accountWriteModel := models.BankAccountWriteModel{
				PlaidAccountId:   plaidAccount.AccountId,
				BankConnectionID: bankConnection.ID,
				Name:             plaidAccount.Name,
				Mask:             plaidAccount.Mask.Get(),
				AccountType:      string(plaidAccount.Type),
			}

			_, err := bankAccountRepository.Save(accountWriteModel)

			if err != nil {
				tx.Rollback(ctx)
				e.Logger.Errorf("Failed to save Plaid Account - %+v. Error was: %w", accountWriteModel, err)
				return c.String(500, fmt.Sprintf("Failed to save Plaid Account - %+v. Error was: %+v", accountWriteModel, err))
			}

		}

		e.Logger.Debugf("Comitting transaction...")
		err = tx.Commit(ctx)

		if err != nil {
			tx.Rollback(ctx)
			e.Logger.Errorf("Failed to commit transaction: %w", err)
			return c.String(500, fmt.Sprintf("Failed to commit transaction: %+v", err))
		}

		e.Logger.Infof("Successfully saved new bank connection with %d accounts", len(authGetResponse.Accounts))

		return renderComponent(
			c,
			200,
			components.Index(components.MainPage(linkToken)),
		)
	})

	e.Logger.Fatal(e.Start(":42069"))
}

func runMigrations(dbpool *pgxpool.Pool, log echo.Logger) error {
	log.Infof("Database migration started")

	db := stdlib.OpenDBFromPool(dbpool)
	log.Infof("DB connection acquired")

	driver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("Could not create SQL migration driver: %v", err)
	}
	log.Infof("PGX driver created")

	m, err := migrate.NewWithDatabaseInstance(
		"file://db/migrations",
		"pgx", driver)
	if err != nil {
		return fmt.Errorf("Could not create migration client: %v", err)
	}
	log.Infof("Migration client created")

	err = m.Up()

	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("An error occurred while running the migrations: %v", err)
	}

	if err == migrate.ErrNoChange {
		log.Infof("Database is up to date")
	}

	if err == nil {
		log.Infof("Migrations succesfully applied")
	}

	return nil
}
