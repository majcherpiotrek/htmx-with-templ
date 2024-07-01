package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.INFO)

	e.GET("/", func(c echo.Context) error {
		return c.HTML(200, "<h1>Hello world</h1>")
	})

	e.Logger.Fatal(e.Start(":42069"))
}
