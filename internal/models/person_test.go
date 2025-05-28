package models_test

import (
	"people-service/internal/models"
	"testing"
)

func TestPersonValidation(t *testing.T) {
	tests := []struct {
		name    string
		person  models.Person
		wantErr bool
	}{
		{
			name:    "empty name",
			person:  models.Person{Surname: "Ivanov"},
			wantErr: true,
		},
		{
			name:    "invalid name chars",
			person:  models.Person{Name: "Ivan123", Surname: "Ivanov"},
			wantErr: true,
		},
		{
			name:    "valid data",
			person:  models.Person{Name: "Иван", Surname: "Иванов", Age: 30},
			wantErr: false,
		},
		{
			name:    "invalid age",
			person:  models.Person{Name: "Ivan", Surname: "Ivanov", Age: 150},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.person.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
