package model

type City struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Uf       string `json:"uf"`
	State    string `json:"state"`
	UrlKey   string `json:"urlKey"`
	TimeZone string `json:"timeZone"`
}
