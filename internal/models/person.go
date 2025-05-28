package models

import (
	"people-service/internal/enrich"

	"gorm.io/gorm"
)

type Person struct {
	gorm.Model
	Name        string  `json:"name"`
	Surname     string  `json:"surname"`
	Patronymic  *string `json:"patronymic,omitempty"`
	Age         int     `json:"age"`
	Gender      string  `json:"gender"`
	Nationality string  `json:"nationality"`
}

func (p *Person) Enrich() error {
	age, err := enrich.GetAge(p.Name)
	if err != nil {
		return err
	}
	p.Age = age

	gender, err := enrich.GetGender(p.Name)
	if err != nil {
		return err
	}
	p.Gender = gender

	nationality, err := enrich.GetNationality(p.Name)
	if err != nil {
		return err
	}
	p.Nationality = nationality

	return nil
}
