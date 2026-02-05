package model

type State struct {
	Name   string `json:"name"`
	Uf     string `json:"uf"`
	Cities []City `json:"cities"`
}
