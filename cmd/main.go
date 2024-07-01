package main

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"htmx-with-templ/components"
)

func render(ctx echo.Context, status int, t templ.Component) error {
	ctx.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	ctx.Response().Writer.WriteHeader(status)

	err := t.Render(ctx.Request().Context(), ctx.Response().Writer)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "failed to render response template")
	}

	return nil
}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.INFO)

	e.GET("/", func(c echo.Context) error {

		return render(c, 200, components.Index(components.Hello("Joe")))
	})

	e.Logger.Fatal(e.Start(":42069"))
}
