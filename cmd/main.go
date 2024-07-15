package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	plaid "github.com/plaid/plaid-go/v21/plaid"

	domainModels "htmx-with-templ/domain/models"
	"htmx-with-templ/view/components"
	viewModels "htmx-with-templ/view/models"
)

var (
	PLAID_CLIENT_ID                      = ""
	PLAID_SECRET                         = ""
	PLAID_ENV                            = ""
	PLAID_PRODUCTS                       = ""
	PLAID_COUNTRY_CODES                  = ""
	PLAID_REDIRECT_URI                   = ""
	APP_PORT                             = ""
	client              *plaid.APIClient = nil
)

var environments = map[string]plaid.Environment{
	"sandbox":     plaid.Sandbox,
	"production":  plaid.Production,
	"development": plaid.Development,
}

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

type Contacts = []domainModels.Contact

func hasContactWithEmail(contacts *Contacts, email string) bool {
	for _, contact := range *contacts {
		if contact.Email == email {
			return true
		}
	}

	return false
}

var id = 0

func validateName(fd *viewModels.FormData) *viewModels.FormData {
	name := fd.Data["name"]

	if len(name) < 1 {
		fd.AddError("name", "Name is required")
	}

	return fd
}

func validateEmail(fd *viewModels.FormData, contacts *Contacts) *viewModels.FormData {
	email := fd.Data["email"]

	if len(email) < 1 {
		fd.AddError("email", "Email is required")
	}

	if hasContactWithEmail(contacts, email) {
		fd.AddError("email", "A user with this email already exists")
	}

	return fd
}

func validateContactFormData(fd *viewModels.FormData, contacts *Contacts) *viewModels.FormData {
	validateName(fd)
	validateEmail(fd, contacts)
	return fd
}

func initPlaid() {
	err := godotenv.Load()

	if err != nil {
		fmt.Println("Error when loading environment variables from .env file %w", err)
	}

	// set constants from env
	PLAID_CLIENT_ID = os.Getenv("PLAID_CLIENT_ID")
	PLAID_SECRET = os.Getenv("PLAID_SECRET")

	if PLAID_CLIENT_ID == "" || PLAID_SECRET == "" {
		log.Fatal("Error: PLAID_SECRET or PLAID_CLIENT_ID is not set. Did you copy .env.example to .env and fill it out?")
	}

	PLAID_ENV = os.Getenv("PLAID_ENV")
	PLAID_PRODUCTS = os.Getenv("PLAID_PRODUCTS")
	PLAID_COUNTRY_CODES = os.Getenv("PLAID_COUNTRY_CODES")
	PLAID_REDIRECT_URI = os.Getenv("PLAID_REDIRECT_URI")

	// set defaults
	if PLAID_PRODUCTS == "" {
		PLAID_PRODUCTS = "transactions"
	}
	if PLAID_COUNTRY_CODES == "" {
		PLAID_COUNTRY_CODES = "US"
	}
	if PLAID_ENV == "" {
		PLAID_ENV = "sandbox"
	}
	if PLAID_CLIENT_ID == "" {
		log.Fatal("PLAID_CLIENT_ID is not set. Make sure to fill out the .env file")
	}
	if PLAID_SECRET == "" {
		log.Fatal("PLAID_SECRET is not set. Make sure to fill out the .env file")
	}

	// create Plaid client
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", PLAID_CLIENT_ID)
	configuration.AddDefaultHeader("PLAID-SECRET", PLAID_SECRET)
	configuration.UseEnvironment(environments[PLAID_ENV])
	client = plaid.NewAPIClient(configuration)
}

func main() {
	e := echo.New()
	initPlaid()

	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.INFO)
	e.Static("/assets", "assets")

	contacts := Contacts{}

	e.GET("/", func(c echo.Context) error {
		contactReadModels := viewModels.MapContacts(&contacts)

		linkToken, err := linkTokenCreate(nil)
		if err != nil {
			return c.String(500, "Something went wrong")
		}

		return renderComponent(c, 200, components.Index(components.ContactPage(viewModels.NewFormData(), contactReadModels, linkToken)))
	})

	e.POST("/validate", func(c echo.Context) error {
		name := c.FormValue("name")
		email := c.FormValue("email")

		formData := viewModels.NewFormData()
		formData.AddValue("name", name)
		formData.AddValue("email", email)

		validateContactFormData(formData, &contacts)

		if formData.HasErrors() {
			return renderComponent(c, 422, components.ContactForm(formData))
		}

		return renderComponent(c, 200, components.ContactForm(formData))
	})

	e.POST("/validate/:field", func(c echo.Context) error {
		field := c.Param("field")

		name := c.FormValue("name")
		email := c.FormValue("email")

		formData := viewModels.NewFormData()
		formData.AddValue("name", name)
		formData.AddValue("email", email)

		if field == "name" {
			validateName(formData)
			return renderComponent(c, 200, components.NameInput(name, formData.Errors["name"]))
		}

		if field == "email" {
			validateEmail(formData, &contacts)
			return renderComponent(c, 200, components.EmailInput(email, formData.Errors["email"]))
		}

		return c.String(400, "Invalid form field")
	})

	e.POST("/contacts", func(c echo.Context) error {
		name := c.FormValue("name")
		email := c.FormValue("email")

		formData := viewModels.NewFormData()
		formData.AddValue("name", name)
		formData.AddValue("email", email)

		validateContactFormData(formData, &contacts)

		if formData.HasErrors() {
			return renderComponent(c, 422, components.ContactForm(formData))
		}

		contactToAdd := domainModels.Contact{Id: id, Name: name, Email: email}

		contacts = append(contacts, contactToAdd)
		id++

		addedContactReadModel := viewModels.FromContactDomainModel(contactToAdd)

		renderComponent(c, 200, components.ContactList(&[]viewModels.ContactReadModel{*addedContactReadModel}, true))

		return renderComponent(c, 200, components.ContactForm(viewModels.NewFormData()))
	})

	e.Logger.Fatal(e.Start(":42069"))
}

// We store the access_token in memory - in production, store it in a secure
// persistent data store.
var accessToken string
var itemID string

var paymentID string

// The authorizationID is only relevant for the Transfer ACH product.
// We store the authorizationID in memory - in production, store it in a secure
// persistent data store
var authorizationID string
var accountID string

func renderError(c echo.Context, originalErr error) {
	if plaidError, err := plaid.ToPlaidError(originalErr); err == nil {
		// Return 200 and allow the front end to render the error.
		c.JSON(http.StatusOK, map[string]interface{}{"error": plaidError})
		return
	}

	c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": originalErr.Error()})
}

func getAccessToken(c echo.Context) {
	publicToken := c.FormValue("public_token")
	ctx := context.Background()

	// exchange the public_token for an access_token
	exchangePublicTokenResp, _, err := client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(
		*plaid.NewItemPublicTokenExchangeRequest(publicToken),
	).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	accessToken = exchangePublicTokenResp.GetAccessToken()
	itemID = exchangePublicTokenResp.GetItemId()

	fmt.Println("public token: " + publicToken)
	fmt.Println("access token: " + accessToken)
	fmt.Println("item ID: " + itemID)

	c.JSON(http.StatusOK, map[string]interface{}{
		"access_token": accessToken,
		"item_id":      itemID,
	})
}

// This functionality is only relevant for the UK/EU Payment Initiation product.
// Creates a link token configured for payment initiation. The payment
// information will be associated with the link token, and will not have to be
// passed in again when we initialize Plaid Link.
// See:
// - https://plaid.com/docs/payment-initiation/
// - https://plaid.com/docs/#payment-initiation-create-link-token-request
func createLinkTokenForPayment(c echo.Context) {
	ctx := context.Background()

	// Create payment recipient
	paymentRecipientRequest := plaid.NewPaymentInitiationRecipientCreateRequest("Harry Potter")
	paymentRecipientRequest.SetIban("GB33BUKB20201555555555")
	paymentRecipientRequest.SetAddress(*plaid.NewPaymentInitiationAddress(
		[]string{"4 Privet Drive"},
		"Little Whinging",
		"11111",
		"GB",
	))
	paymentRecipientCreateResp, _, err := client.PlaidApi.PaymentInitiationRecipientCreate(ctx).PaymentInitiationRecipientCreateRequest(*paymentRecipientRequest).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	// Create payment
	paymentCreateRequest := plaid.NewPaymentInitiationPaymentCreateRequest(
		paymentRecipientCreateResp.GetRecipientId(),
		"paymentRef",
		*plaid.NewPaymentAmount("GBP", 1.34),
	)
	paymentCreateResp, _, err := client.PlaidApi.PaymentInitiationPaymentCreate(ctx).PaymentInitiationPaymentCreateRequest(*paymentCreateRequest).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	// We store the payment_id in memory for demo purposes - in production, store it in a secure
	// persistent data store along with the Payment metadata, such as userId.
	paymentID = paymentCreateResp.GetPaymentId()
	fmt.Println("payment id: " + paymentID)

	// Create the link_token
	linkTokenCreateReqPaymentInitiation := plaid.NewLinkTokenCreateRequestPaymentInitiation()
	linkTokenCreateReqPaymentInitiation.SetPaymentId(paymentID)
	linkToken, err := linkTokenCreate(linkTokenCreateReqPaymentInitiation)
	if err != nil {
		renderError(c, err)
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{
		"link_token": linkToken,
	})
}

func auth(c echo.Context) {
	ctx := context.Background()

	authGetResp, _, err := client.PlaidApi.AuthGet(ctx).AuthGetRequest(
		*plaid.NewAuthGetRequest(accessToken),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"accounts": authGetResp.GetAccounts(),
		"numbers":  authGetResp.GetNumbers(),
	})
}

func accounts(c echo.Context) {
	ctx := context.Background()

	accountsGetResp, _, err := client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(
		*plaid.NewAccountsGetRequest(accessToken),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"accounts": accountsGetResp.GetAccounts(),
	})
}

func balance(c echo.Context) {
	ctx := context.Background()

	balancesGetResp, _, err := client.PlaidApi.AccountsBalanceGet(ctx).AccountsBalanceGetRequest(
		*plaid.NewAccountsBalanceGetRequest(accessToken),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"accounts": balancesGetResp.GetAccounts(),
	})
}

func item(c echo.Context) {
	ctx := context.Background()

	itemGetResp, _, err := client.PlaidApi.ItemGet(ctx).ItemGetRequest(
		*plaid.NewItemGetRequest(accessToken),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	institutionGetByIdResp, _, err := client.PlaidApi.InstitutionsGetById(ctx).InstitutionsGetByIdRequest(
		*plaid.NewInstitutionsGetByIdRequest(
			*itemGetResp.GetItem().InstitutionId.Get(),
			convertCountryCodes(strings.Split(PLAID_COUNTRY_CODES, ",")),
		),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"item":        itemGetResp.GetItem(),
		"institution": institutionGetByIdResp.GetInstitution(),
	})
}

func identity(c echo.Context) {
	ctx := context.Background()

	identityGetResp, _, err := client.PlaidApi.IdentityGet(ctx).IdentityGetRequest(
		*plaid.NewIdentityGetRequest(accessToken),
	).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"identity": identityGetResp.GetAccounts(),
	})
}

func transactions(c echo.Context) {
	ctx := context.Background()

	// Set cursor to empty to receive all historical updates
	var cursor *string

	// New transaction updates since "cursor"
	var added []plaid.Transaction
	var modified []plaid.Transaction
	var removed []plaid.RemovedTransaction // Removed transaction ids
	hasMore := true
	// Iterate through each page of new transaction updates for item
	for hasMore {
		request := plaid.NewTransactionsSyncRequest(accessToken)
		if cursor != nil {
			request.SetCursor(*cursor)
		}
		resp, _, err := client.PlaidApi.TransactionsSync(
			ctx,
		).TransactionsSyncRequest(*request).Execute()
		if err != nil {
			renderError(c, err)
			return
		}

		// Update cursor to the next cursor
		nextCursor := resp.GetNextCursor()
		cursor = &nextCursor

		// If no transactions are available yet, wait and poll the endpoint.
		// Normally, we would listen for a webhook, but the Quickstart doesn't
		// support webhooks. For a webhook example, see
		// https://github.com/plaid/tutorial-resources or
		// https://github.com/plaid/pattern

		if *cursor == "" {
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
	latestTransactions := added[len(added)-9:]

	c.JSON(http.StatusOK, map[string]interface{}{
		"latest_transactions": latestTransactions,
	})
}

// This functionality is only relevant for the UK Payment Initiation product.
// Retrieve Payment for a specified Payment ID
func payment(c echo.Context) {
	ctx := context.Background()

	paymentGetResp, _, err := client.PlaidApi.PaymentInitiationPaymentGet(ctx).PaymentInitiationPaymentGetRequest(
		*plaid.NewPaymentInitiationPaymentGetRequest(paymentID),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"payment": paymentGetResp,
	})
}

func signalEvaluate(c echo.Context) {
	ctx := context.Background()
	accountsGetResp, _, err := client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(
		*plaid.NewAccountsGetRequest(accessToken),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	accountID = accountsGetResp.GetAccounts()[0].AccountId

	signalEvaluateRequest := plaid.NewSignalEvaluateRequest(
		accessToken,
		accountID,
		"txn1234",
		100.00)

	signalEvaluateResp, _, err := client.PlaidApi.SignalEvaluate(ctx).SignalEvaluateRequest(*signalEvaluateRequest).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, signalEvaluateResp)
}

func investmentTransactions(c echo.Context) {
	ctx := context.Background()

	endDate := time.Now().Local().Format("2006-01-02")
	startDate := time.Now().Local().Add(-30 * 24 * time.Hour).Format("2006-01-02")

	request := plaid.NewInvestmentsTransactionsGetRequest(accessToken, startDate, endDate)
	invTxResp, _, err := client.PlaidApi.InvestmentsTransactionsGet(ctx).InvestmentsTransactionsGetRequest(*request).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"investments_transactions": invTxResp,
	})
}

func holdings(c echo.Context) {
	ctx := context.Background()

	holdingsGetResp, _, err := client.PlaidApi.InvestmentsHoldingsGet(ctx).InvestmentsHoldingsGetRequest(
		*plaid.NewInvestmentsHoldingsGetRequest(accessToken),
	).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"holdings": holdingsGetResp,
	})
}

func info(context echo.Context) {
	context.JSON(http.StatusOK, map[string]interface{}{
		"item_id":      itemID,
		"access_token": accessToken,
		"products":     strings.Split(PLAID_PRODUCTS, ","),
	})
}

func createPublicToken(c echo.Context) {
	ctx := context.Background()

	// Create a one-time use public_token for the Item.
	// This public_token can be used to initialize Link in update mode for a user
	publicTokenCreateResp, _, err := client.PlaidApi.ItemCreatePublicToken(ctx).ItemPublicTokenCreateRequest(
		*plaid.NewItemPublicTokenCreateRequest(accessToken),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"public_token": publicTokenCreateResp.GetPublicToken(),
	})
}

func createLinkToken(c echo.Context) {
	linkToken, err := linkTokenCreate(nil)
	if err != nil {
		renderError(c, err)
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"link_token": linkToken})
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
func linkTokenCreate(
	paymentInitiation *plaid.LinkTokenCreateRequestPaymentInitiation,
) (string, error) {
	ctx := context.Background()

	// Institutions from all listed countries will be shown.
	countryCodes := convertCountryCodes(strings.Split(PLAID_COUNTRY_CODES, ","))
	redirectURI := PLAID_REDIRECT_URI

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

	products := convertProducts(strings.Split(PLAID_PRODUCTS, ","))
	if paymentInitiation != nil {
		request.SetPaymentInitiation(*paymentInitiation)
		// The 'payment_initiation' product has to be the only element in the 'products' list.
		request.SetProducts([]plaid.Products{plaid.PRODUCTS_PAYMENT_INITIATION})
	} else {
		request.SetProducts(products)
	}

	if containsProduct(products, plaid.PRODUCTS_STATEMENTS) {
		statementConfig := plaid.NewLinkTokenCreateRequestStatements()
		statementConfig.SetStartDate(time.Now().Local().Add(-30 * 24 * time.Hour).Format("2006-01-02"))
		statementConfig.SetEndDate(time.Now().Local().Format("2006-01-02"))
		request.SetStatements(*statementConfig)
	}

	if redirectURI != "" {
		request.SetRedirectUri(redirectURI)
	}

	linkTokenCreateResp, _, err := client.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*request).Execute()

	if err != nil {
		return "", err
	}

	return linkTokenCreateResp.GetLinkToken(), nil
}

func statements(c echo.Context) {
	ctx := context.Background()
	statementsListResp, _, err := client.PlaidApi.StatementsList(ctx).StatementsListRequest(
		*plaid.NewStatementsListRequest(accessToken),
	).Execute()
	statementId := statementsListResp.GetAccounts()[0].GetStatements()[0].StatementId

	statementsDownloadResp, _, err := client.PlaidApi.StatementsDownload(ctx).StatementsDownloadRequest(
		*plaid.NewStatementsDownloadRequest(accessToken, statementId),
	).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	reader := bufio.NewReader(statementsDownloadResp)
	content, err := io.ReadAll(reader)
	if err != nil {
		renderError(c, err)
		return
	}

	// convert pdf to base64
	encodedPdf := base64.StdEncoding.EncodeToString(content)

	c.JSON(http.StatusOK, map[string]interface{}{
		"json": statementsListResp,
		"pdf":  encodedPdf,
	})
}

func assets(c echo.Context) {
	ctx := context.Background()

	createRequest := plaid.NewAssetReportCreateRequest(10)
	createRequest.SetAccessTokens([]string{accessToken})

	// create the asset report
	assetReportCreateResp, _, err := client.PlaidApi.AssetReportCreate(ctx).AssetReportCreateRequest(
		*createRequest,
	).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	assetReportToken := assetReportCreateResp.GetAssetReportToken()

	// get the asset report
	assetReportGetResp, err := pollForAssetReport(ctx, client, assetReportToken)
	if err != nil {
		renderError(c, err)
		return
	}

	// get it as a pdf
	pdfRequest := plaid.NewAssetReportPDFGetRequest(assetReportToken)
	pdfFile, _, err := client.PlaidApi.AssetReportPdfGet(ctx).AssetReportPDFGetRequest(*pdfRequest).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	reader := bufio.NewReader(pdfFile)
	content, err := io.ReadAll(reader)
	if err != nil {
		renderError(c, err)
		return
	}

	// convert pdf to base64
	encodedPdf := base64.StdEncoding.EncodeToString(content)

	c.JSON(http.StatusOK, map[string]interface{}{
		"json": assetReportGetResp.GetReport(),
		"pdf":  encodedPdf,
	})
}

func pollForAssetReport(ctx context.Context, client *plaid.APIClient, assetReportToken string) (*plaid.AssetReportGetResponse, error) {
	numRetries := 20
	request := plaid.NewAssetReportGetRequest()
	request.SetAssetReportToken(assetReportToken)

	for i := 0; i < numRetries; i++ {
		response, _, err := client.PlaidApi.AssetReportGet(ctx).AssetReportGetRequest(*request).Execute()
		if err != nil {
			plaidErr, err := plaid.ToPlaidError(err)
			if plaidErr.ErrorCode == "PRODUCT_NOT_READY" {
				time.Sleep(1 * time.Second)
				continue
			} else {
				return nil, err
			}
		} else {
			return &response, nil
		}
	}
	return nil, errors.New("Timed out when polling for an asset report.")
}
