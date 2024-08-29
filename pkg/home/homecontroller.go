package home

import (
	"nerdmoney/pkg/banking"
	"nerdmoney/pkg/common/layout"

	"github.com/labstack/echo/v4"
)

func RegisterHomeRoutes(e *echo.Echo, plaidClient *banking.PlaidClient) {
	e.GET("/", func(c echo.Context) error {
		linkTokenResponse, err := plaidClient.CreateLinkToken()
		if err != nil {
			return c.String(500, "Something went wrong")
		}

		return layout.RenderPage(
			c,
			200,
			HomePage(linkTokenResponse.LinkToken),
		)
	})
}
