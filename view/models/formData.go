package models

type FormData struct {
	Data   map[string]string
	Errors map[string]string
}

func NewFormData() *FormData {
	return &FormData{
		Data:   make(map[string]string),
		Errors: make(map[string]string),
	}
}

func (fd *FormData) AddError(key, message string) {
	fd.Errors[key] = message
}

func (fd *FormData) HasErrors() bool {
	return len(fd.Errors) > 0
}

func (fd *FormData) AddValue(key, value string) {
	fd.Data[key] = value
}
