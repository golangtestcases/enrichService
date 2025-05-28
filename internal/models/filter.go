package models

type PersonFilter struct {
	Name        string `form:"name"`
	Surname     string `form:"surname"`
	Age         int    `form:"age"`
	Gender      string `form:"gender"`
	Nationality string `form:"nationality"`
	Page        int    `form:"page" default:"1"`
	Limit       int    `form:"limit" default:"10"`
}
