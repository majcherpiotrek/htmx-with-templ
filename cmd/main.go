package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"nerdmoney/pkg/accounts"
	"nerdmoney/pkg/accounts/repositories"
	"nerdmoney/pkg/banking"
	"nerdmoney/pkg/home"

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

	dbPool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		e.Logger.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	err = runMigrations(dbPool, e.Logger)
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

	// Instantiate repositories
	bankConnectionRepository := repositories.NewBankConnectionRepository(dbPool, e.Logger)
	bankAccountRepository := repositories.NewBankAccountRepository(dbPool, e.Logger)

	// Register routes
	home.RegisterHomeRoutes(e, plaidClient)
	accounts.RegisterAccountRoutes(e, plaidClient, bankConnectionRepository, bankAccountRepository)

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
