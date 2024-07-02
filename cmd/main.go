package main

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	domainModels "htmx-with-templ/domain/models"
	"htmx-with-templ/view/components"
	viewModels "htmx-with-templ/view/models"
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

func validateContactFormData(fd *viewModels.FormData, contacts *Contacts) *viewModels.FormData {
	name := fd.Data["name"]
	email := fd.Data["email"]

	if len(name) < 1 {
		fd.AddError("name", "Name is required")
	}

	if len(email) < 1 {
		fd.AddError("email", "Email is required")
	}

	if hasContactWithEmail(contacts, email) {
		fd.AddError("email", "A user with this email already exists")
	}

	return fd
}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.INFO)
	e.Static("/assets", "assets")

	contacts := Contacts{}

	e.GET("/", func(c echo.Context) error {
		contactReadModels := viewModels.MapContacts(&contacts)
		return renderComponent(c, 200, components.Index(components.ContactPage(viewModels.NewFormData(), contactReadModels)))
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
