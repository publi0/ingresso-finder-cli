package model

import "time"

type SessionDetail struct {
	Id       string           `json:"id"`
	Date     time.Time        `json:"date"`
	Sections []SessionSection `json:"sections"`
}

type SessionSection struct {
	Id               string  `json:"id"`
	Name             string  `json:"name"`
	Capacity         int     `json:"capacity"`
	HasSeatSelection bool    `json:"hasSeatSelection"`
	Layout           string  `json:"layout"`
	HighestPrice     float64 `json:"highestPrice"`
	LowestPrice      float64 `json:"lowestPrice"`
}

type SeatMap struct {
	Id     string     `json:"id"`
	Bounds SeatBounds `json:"bounds"`
	Lines  []SeatLine `json:"lines"`
}

type SeatBounds struct {
	Lines   int `json:"lines"`
	Columns int `json:"columns"`
}

type SeatLine struct {
	Line  int    `json:"line"`
	Seats []Seat `json:"seats"`
}

type Seat struct {
	Id          string `json:"id"`
	Label       string `json:"label"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	RowIndex    int    `json:"rowIndex"`
	ColumnIndex int    `json:"columnIndex"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
}
