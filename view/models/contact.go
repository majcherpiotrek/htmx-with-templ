package models

import domainModels "htmx-with-templ/domain/models"

type ContactWriteModel struct {
	name  string
	email string
}

type ContactReadModel struct {
	Id    int
	Name  string
	Email string
}

func FromContactDomainModel(dm domainModels.Contact) *ContactReadModel {
	return &ContactReadModel{
		Id:    dm.Id,
		Name:  dm.Name,
		Email: dm.Email,
	}
}

func MapContacts(contacts *[]domainModels.Contact) *[]ContactReadModel {
	readModels := make([]ContactReadModel, len(*contacts))

	for i, contact := range *contacts {
		readModels[i] = *FromContactDomainModel(contact)
	}

	return &readModels
}
