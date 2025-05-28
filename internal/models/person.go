package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"people-service/internal/enrich"

	"gorm.io/gorm"
)

type Person struct {
	gorm.Model
	Name        string  `json:"name" validate:"required,min=2,max=100,alphaunicode"`
	Surname     string  `json:"surname" validate:"required,min=2,max=100,alphaunicode"`
	Patronymic  *string `json:"patronymic,omitempty" validate:"omitempty,min=2,max=100,alphaunicode"`
	Age         int     `json:"age" validate:"min=0,max=120"`
	Gender      string  `json:"gender" validate:"omitempty,oneof=male female other"`
	Nationality string  `json:"nationality" validate:"omitempty,len=2"`
}

var (
	ErrNameRequired    = errors.New("имя обязательно")
	ErrSurnameRequired = errors.New("фамилия обязательна")
	ErrInvalidName     = errors.New("имя может содержать только буквы")
	ErrInvalidAge      = errors.New("возраст должен быть от 0 до 120")
	ErrInvalidGender   = errors.New("пол должен быть 'male', 'female' или 'other'")
	nameRegex          = regexp.MustCompile(`^[a-zA-Zа-яА-Я\-]+$`)
)

// GetPeople возвращает список людей с фильтрацией и пагинацией
func GetPeople(db *gorm.DB, filter PersonFilter) ([]Person, int64, error) {
	var people []Person
	query := db.Model(&Person{})

	// Применяем фильтры
	if filter.Name != "" {
		query = query.Where("name ILIKE ?", "%"+filter.Name+"%")
	}
	if filter.Surname != "" {
		query = query.Where("surname ILIKE ?", "%"+filter.Surname+"%")
	}
	if filter.Age > 0 {
		query = query.Where("age = ?", filter.Age)
	}
	if filter.Gender != "" {
		query = query.Where("gender = ?", strings.ToLower(filter.Gender))
	}
	if filter.Nationality != "" {
		query = query.Where("nationality = ?", strings.ToUpper(filter.Nationality))
	}

	// Получаем общее количество записей (для пагинации)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("ошибка получения общего количества: %w", err)
	}

	// Применяем пагинацию
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	// Получаем данные
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&people).Error; err != nil {
		return nil, 0, fmt.Errorf("ошибка получения списка: %w", err)
	}

	return people, total, nil
}

func (p *Person) Validate() error {
	var errs []error

	if strings.TrimSpace(p.Name) == "" {
		errs = append(errs, ErrNameRequired)
	} else if !nameRegex.MatchString(p.Name) {
		errs = append(errs, ErrInvalidName)
	}

	if strings.TrimSpace(p.Surname) == "" {
		errs = append(errs, ErrSurnameRequired)
	} else if !nameRegex.MatchString(p.Surname) {
		errs = append(errs, ErrInvalidName)
	}

	if p.Patronymic != nil && *p.Patronymic != "" && !nameRegex.MatchString(*p.Patronymic) {
		errs = append(errs, ErrInvalidName)
	}

	if p.Age < 0 || p.Age > 120 {
		errs = append(errs, ErrInvalidAge)
	}

	if p.Gender != "" && !contains([]string{"male", "female", "other"}, strings.ToLower(p.Gender)) {
		errs = append(errs, ErrInvalidGender)
	}

	if len(errs) > 0 {
		return joinErrors(errs)
	}
	return nil
}

func (p *Person) Enrich() error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("нельзя обогатить невалидные данные: %w", err)
	}

	age, err := enrich.GetAge(p.Name)
	if err != nil {
		return fmt.Errorf("ошибка получения возраста: %w", err)
	}
	p.Age = age

	gender, err := enrich.GetGender(p.Name)
	if err != nil {
		return fmt.Errorf("ошибка определения пола: %w", err)
	}
	p.Gender = gender

	nationality, err := enrich.GetNationality(p.Name)
	if err != nil {
		return fmt.Errorf("ошибка определения национальности: %w", err)
	}
	p.Nationality = nationality

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func joinErrors(errs []error) error {
	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return errors.New(strings.Join(messages, "; "))
}
