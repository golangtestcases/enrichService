package models

type PeopleListResponse struct {
	Data  []Person `json:"data"`
	Total int64    `json:"total"`
	Page  int      `json:"page"`
	Limit int      `json:"limit"`
}
