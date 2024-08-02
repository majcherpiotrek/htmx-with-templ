package banking

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	plaid "github.com/plaid/plaid-go/v21/plaid"
)

var environments = map[string]plaid.Environment{
	"sandbox":     plaid.Sandbox,
	"production":  plaid.Production,
	"development": plaid.Development,
}

type PlaidClient struct {
	client *plaid.APIClient
	config PlaidClientConfig
}

type PlaidClientConfig struct {
	ClientId     string
	Secret       string
	Env          string
	Products     string
	CountryCodes string
	RedirectUri  string
}

func NewPlaidClient(config PlaidClientConfig) (*PlaidClient, error) {
	Env, isOk := environments[config.Env]

	if !isOk {
		return nil, fmt.Errorf("Incorrect env value for Plaid environment: '%s'", config.Env)
	}

	// set defaults
	if config.Products == "" {
		config.Products = "transactions"
	}

	if config.CountryCodes == "" {
		config.CountryCodes = "US"
	}

	if config.ClientId == "" {
		return nil, fmt.Errorf("PLAID_CLIENT_ID is not set. Make sure to fill out the .env file")
	}

	if config.Secret == "" {
		return nil, fmt.Errorf("PLAID_SECRET is not set. Make sure to fill out the .env file")
	}

	// create Plaid client
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", config.ClientId)
	configuration.AddDefaultHeader("PLAID-SECRET", config.Secret)
	configuration.UseEnvironment(Env)

	plaidClient := PlaidClient{
		client: plaid.NewAPIClient(configuration),
		config: config,
	}

	return &plaidClient, nil
}

var paymentID string

// The authorizationID is only relevant for the Transfer ACH product.
// We store the authorizationID in memory - in production, store it in a secure
// persistent data store
var authorizationID string
var accountID string

type ItemAccessToken struct {
	AccessToken string
	ItemId      string
}

func (pc *PlaidClient) GetAccessToken(publicToken string) (ItemAccessToken, error) {
	ctx := context.Background()

	// exchange the public_token for an access_token
	exchangePublicTokenResp, _, err := pc.client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(
		*plaid.NewItemPublicTokenExchangeRequest(publicToken),
	).Execute()

	if err != nil {
		return ItemAccessToken{}, err
	}

	accessToken := exchangePublicTokenResp.GetAccessToken()
	itemId := exchangePublicTokenResp.GetItemId()

	return ItemAccessToken{
		AccessToken: accessToken,
		ItemId:      itemId,
	}, nil
}

// https://plaid.com/docs/api/products/auth/#authget
func (pc *PlaidClient) AuthGet(accessToken string) (plaid.AuthGetResponse, error) {
	ctx := context.Background()

	authGetResp, _, err := pc.client.PlaidApi.AuthGet(ctx).AuthGetRequest(
		*plaid.NewAuthGetRequest(accessToken),
	).Execute()

	return authGetResp, err

}

// https://plaid.com/docs/api/accounts/#accountsget
func (pc *PlaidClient) Accounts(accessToken string) (plaid.AccountsGetResponse, error) {
	ctx := context.Background()

	accountsGetResp, _, err := pc.client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(
		*plaid.NewAccountsGetRequest(accessToken),
	).Execute()

	return accountsGetResp, err
}

// https://plaid.com/docs/api/products/balance/#accountsbalanceget
func (pc *PlaidClient) Balances(accessToken string) (plaid.AccountsGetResponse, error) {
	ctx := context.Background()

	balancesGetResp, _, err := pc.client.PlaidApi.AccountsBalanceGet(ctx).AccountsBalanceGetRequest(
		*plaid.NewAccountsBalanceGetRequest(accessToken),
	).Execute()

	return balancesGetResp, err
}

type GetItemResponse struct {
	Item        plaid.ItemGetResponse
	Institution plaid.InstitutionsGetByIdResponse
}

// https://plaid.com/docs/api/items/#itemget
// https://plaid.com/docs/api/institutions/#institutionsget_by_id
func (pc *PlaidClient) Item(accessToken string) (GetItemResponse, error) {
	ctx := context.Background()

	itemGetResp, _, err := pc.client.PlaidApi.ItemGet(ctx).ItemGetRequest(
		*plaid.NewItemGetRequest(accessToken),
	).Execute()

	if err != nil {
		return GetItemResponse{}, err
	}

	institutionGetByIdResp, _, err := pc.client.PlaidApi.InstitutionsGetById(ctx).InstitutionsGetByIdRequest(
		*plaid.NewInstitutionsGetByIdRequest(
			*itemGetResp.GetItem().InstitutionId.Get(),
			convertCountryCodes(strings.Split(pc.config.CountryCodes, ",")),
		),
	).Execute()

	if err != nil {
		return GetItemResponse{}, err
	}

	return GetItemResponse{
		Item:        itemGetResp,
		Institution: institutionGetByIdResp,
	}, nil
}

type GetTransactionsRequest struct {
	Cursor *string
}

type LatestTransactionsResponse struct {
	Added    []plaid.Transaction
	Modified []plaid.Transaction
	Removed  []plaid.RemovedTransaction
}

// https://plaid.com/docs/api/products/transactions/#transactionssync
func (pc *PlaidClient) Transactions(request GetTransactionsRequest, accessToken string) (LatestTransactionsResponse, error) {
	ctx := context.Background()

	// New transaction updates since "cursor"
	var added []plaid.Transaction
	var modified []plaid.Transaction
	var removed []plaid.RemovedTransaction // Removed transaction ids
	hasMore := true
	// Iterate through each page of new transaction updates for item
	for hasMore {
		request := plaid.NewTransactionsSyncRequest(accessToken)
		if request.Cursor != nil {
			request.SetCursor(*request.Cursor)
		}
		resp, _, err := pc.client.PlaidApi.TransactionsSync(
			ctx,
		).TransactionsSyncRequest(*request).Execute()
		if err != nil {
			return LatestTransactionsResponse{}, err
		}

		// Update cursor to the next cursor
		nextCursor := resp.GetNextCursor()

		// If no transactions are available yet, wait and poll the endpoint.
		// Normally, we would listen for a webhook, but the Quickstart doesn't
		// support webhooks. For a webhook example, see
		// https://github.com/plaid/tutorial-resources or
		// https://github.com/plaid/pattern

		if nextCursor == "" {
			time.Sleep(2 * time.Second)
			continue
		}

		// Add this page of results
		added = append(added, resp.GetAdded()...)
		modified = append(modified, resp.GetModified()...)
		removed = append(removed, resp.GetRemoved()...)
		hasMore = resp.GetHasMore()
	}

	sort.Slice(added, func(i, j int) bool {
		return added[i].GetDate() < added[j].GetDate()
	})

	return LatestTransactionsResponse{
		Added:    added,
		Modified: modified,
		Removed:  removed,
	}, nil
}

func (pc *PlaidClient) CreatePublicToken(accessToken string) (plaid.ItemPublicTokenCreateResponse, error) {
	ctx := context.Background()

	// Create a one-time use public_token for the Item.
	// This public_token can be used to initialize Link in update mode for a user
	publicTokenCreateResp, _, err := pc.client.PlaidApi.ItemCreatePublicToken(ctx).ItemPublicTokenCreateRequest(
		*plaid.NewItemPublicTokenCreateRequest(accessToken),
	).Execute()

	return publicTokenCreateResp, err
}

type LinkTokenResponse struct {
	LinkToken string
}

func (pc *PlaidClient) CreateLinkToken() (LinkTokenResponse, error) {
	linkToken, err := pc.linkTokenCreate()
	if err != nil {
		return LinkTokenResponse{}, err
	}
	return LinkTokenResponse{LinkToken: linkToken}, nil
}

func convertCountryCodes(countryCodeStrs []string) []plaid.CountryCode {
	countryCodes := []plaid.CountryCode{}

	for _, countryCodeStr := range countryCodeStrs {
		countryCodes = append(countryCodes, plaid.CountryCode(countryCodeStr))
	}

	return countryCodes
}

func convertProducts(productStrs []string) []plaid.Products {
	products := []plaid.Products{}

	for _, productStr := range productStrs {
		products = append(products, plaid.Products(productStr))
	}

	return products
}

func containsProduct(products []plaid.Products, product plaid.Products) bool {
	for _, p := range products {
		if p == product {
			return true
		}
	}
	return false
}

// linkTokenCreate creates a link token using the specified parameters
func (pc *PlaidClient) linkTokenCreate() (string, error) {
	ctx := context.Background()

	// Institutions from all listed countries will be shown.
	countryCodes := convertCountryCodes(strings.Split(pc.config.CountryCodes, ","))
	redirectURI := pc.config.RedirectUri

	// This should correspond to a unique id for the current user.
	// Typically, this will be a user ID number from your application.
	// Personally identifiable information, such as an email address or phone number, should not be used here.
	user := plaid.LinkTokenCreateRequestUser{
		ClientUserId: time.Now().String(),
	}

	request := plaid.NewLinkTokenCreateRequest(
		"Plaid Quickstart",
		"en",
		countryCodes,
		user,
	)

	products := convertProducts(strings.Split(pc.config.Products, ","))
	request.SetProducts(products)

	if containsProduct(products, plaid.PRODUCTS_STATEMENTS) {
		statementConfig := plaid.NewLinkTokenCreateRequestStatements()
		statementConfig.SetStartDate(time.Now().Local().Add(-30 * 24 * time.Hour).Format("2006-01-02"))
		statementConfig.SetEndDate(time.Now().Local().Format("2006-01-02"))
		request.SetStatements(*statementConfig)
	}

	if redirectURI != "" {
		request.SetRedirectUri(redirectURI)
	}

	linkTokenCreateResp, _, err := pc.client.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*request).Execute()

	if err != nil {
		return "", err
	}

	return linkTokenCreateResp.GetLinkToken(), nil
}

// https://plaid.com/docs/api/products/statements/#statementslist
func (pc *PlaidClient) Statements(accessToken string) (plaid.StatementsListResponse, error) {
	ctx := context.Background()
	statementsListResp, _, err := pc.client.PlaidApi.StatementsList(ctx).StatementsListRequest(
		*plaid.NewStatementsListRequest(accessToken),
	).Execute()

	return statementsListResp, err
}
