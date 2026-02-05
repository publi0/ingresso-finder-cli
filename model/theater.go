package model

import "time"

type Theater struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	Neighborhood string `json:"neighborhood"`
	City         string `json:"city"`
	Uf           string `json:"uf"`
	UrlKey       string `json:"urlKey"`
}

type TheaterSessionDay struct {
	Date          string         `json:"date"`
	DateFormatted string         `json:"dateFormatted"`
	DayOfWeek     string         `json:"dayOfWeek"`
	IsToday       bool           `json:"isToday"`
	Movies        []TheaterMovie `json:"movies"`
}

type TheaterMovie struct {
	Id            string        `json:"id"`
	Title         string        `json:"title"`
	OriginalTitle string        `json:"originalTitle"`
	ContentRating string        `json:"contentRating"`
	Duration      string        `json:"duration"`
	Rooms         []TheaterRoom `json:"rooms"`
}

type TheaterRoom struct {
	Name     string           `json:"name"`
	Sessions []TheaterSession `json:"sessions"`
}

type TheaterSession struct {
	Id               string   `json:"id"`
	Price            float64  `json:"price"`
	Room             string   `json:"room"`
	Type             []string `json:"type"`
	HasSeatSelection bool     `json:"hasSeatSelection"`
	Date             struct {
		LocalDate time.Time `json:"localDate"`
	} `json:"date"`
}
