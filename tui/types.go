package tui

import (
	"time"

	"ingresso-finder-cli/model"
	"ingresso-finder-cli/service"
	"ingresso-finder-cli/store"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
)

type appState int

const (
	stateLoadingCities appState = iota
	stateSelectCity
	stateLoadingTheaters
	stateSelectTheater
	stateLoadingSessions
	stateSelectMovie
	stateShowSessions
	stateSelectDate
	stateLoadingSeatMap
	stateSelectSection
	stateShowSeatMap
	stateManageTheaters
	stateError
)

type appModel struct {
	client *service.Client

	state     appState
	lastState appState
	err       error

	width  int
	height int

	cities   []model.City
	theaters []model.Theater
	days     []model.TheaterSessionDay

	city    model.City
	theater model.Theater
	date    time.Time

	dateReturnState    appState
	dateReturnStateSet bool

	cityList    list.Model
	theaterList list.Model
	theaterPref list.Model
	movieList   list.Model
	sessionList list.Model
	sectionList list.Model
	dateList    list.Model

	seatMap         model.SeatMap
	selectedSession model.TheaterSession
	selectedSection model.SessionSection
	showSeatNumbers bool

	spinner spinner.Model

	seatCounts   map[string]seatCount
	movieRatings map[string]store.OMDbRating

	hiddenTheaters      map[string]bool
	userLocation        *service.UserLocation
	browsingAllTheaters bool

	errorSuggestNextDay bool
}

type errMsg struct {
	err            error
	returnState    appState
	returnStateSet bool
	suggestNextDay bool
}

type citiesMsg struct {
	cities []model.City
	err    error
}

type cityMsg struct {
	city model.City
	err  error
}

type theatersMsg struct {
	theaters []model.Theater
	err      error
}

type sessionsMsg struct {
	days []model.TheaterSessionDay
	err  error
}

type seatCountMsg struct {
	sessionID string
	count     seatCount
}

type sessionDetailsMsg struct {
	detail model.SessionDetail
	err    error
}

type seatMapMsg struct {
	seatMap model.SeatMap
	err     error
}

type movieCatalogMsg struct {
	movies     []movieAggregate
	err        error
	failed     int
	ignored    int
	noSessions bool
}

type locationMsg struct {
	location service.UserLocation
	err      error
}

type theaterSessionsResult struct {
	theater model.Theater
	days    []model.TheaterSessionDay
	err     error
}
