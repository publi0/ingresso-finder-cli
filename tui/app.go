package tui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"ingresso-finder-cli/model"
	"ingresso-finder-cli/service"
	"ingresso-finder-cli/store"
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

	seatCounts map[string]seatCount

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

func New() tea.Model {
	client := service.NewClient(nil)
	m := appModel{
		client: client,
		state:  stateLoadingCities,
		date:   truncateDate(time.Now()),
	}

	m.cityList = newList("Select City")
	m.theaterList = newList("Theaters")
	m.theaterPref = newList("Visible Theaters")
	m.movieList = newList("Select Movie")
	m.sessionList = newList("Sessions")
	m.sectionList = newList("Select Section")
	m.dateList = newList("Select Date")

	m.showSeatNumbers = true
	m.seatCounts = make(map[string]seatCount)
	m.hiddenTheaters = make(map[string]bool)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	m.spinner = sp

	return m
}

func (m appModel) Init() tea.Cmd {
	if cityName := strings.TrimSpace(os.Getenv("INGRESSO_CITY")); cityName != "" {
		return tea.Batch(m.fetchCityByNameCmd(cityName), m.spinner.Tick)
	}
	if recent, ok := startupRecentCity(); ok {
		return tea.Batch(m.fetchRecentCityCmd(recent), m.spinner.Tick)
	}
	return tea.Batch(m.fetchCitiesCmd(), m.spinner.Tick)
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeLists()
		if m.state == stateShowSessions {
			return m, m.startSeatCountFetchForVisiblePage()
		}
		return m, nil

	case tea.KeyMsg:
		if m.handleFilterInput(msg) {
			if m.state == stateShowSessions {
				return m, m.startSeatCountFetchForVisiblePage()
			}
			return m, nil
		}
		var handled bool
		m, cmd, handled := m.handleKey(msg)
		if handled {
			return m, cmd
		}
		// fallthrough to component update
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.isLoadingState() {
			return m, cmd
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		if msg.returnStateSet {
			m.lastState = msg.returnState
		} else {
			m.lastState = recoverStateFrom(m.state)
		}
		m.errorSuggestNextDay = msg.suggestNextDay
		m.state = stateError
		return m, nil

	case citiesMsg:
		if msg.err != nil {
			return m, errCmd(msg.err)
		}
		m.cities = msg.cities
		m.cityList.SetItems(buildCityItems(msg.cities))
		m.state = stateSelectCity
		return m, nil

	case cityMsg:
		if msg.err != nil {
			m.state = stateLoadingCities
			return m, tea.Batch(m.fetchCitiesCmd(), m.spinner.Tick)
		}
		m.city = msg.city
		m.theater = model.Theater{}
		m.browsingAllTheaters = false
		_ = store.RememberCity(m.city)
		m.state = stateLoadingTheaters
		return m, tea.Batch(m.fetchTheatersCmd(m.city.Id), m.spinner.Tick)

	case theatersMsg:
		if msg.err != nil {
			return m, errCmd(msg.err)
		}
		m.theaters = msg.theaters
		hidden, err := store.LoadHiddenTheaters(m.city.Id)
		if err != nil {
			return m, errCmd(err)
		}
		m.hiddenTheaters = hidden
		m.refreshTheaterLists()
		m.theaterList.Select(0)
		m.state = stateSelectTheater
		return m, nil

	case sessionsMsg:
		if msg.err != nil {
			return m, errWithOptionsCmd(msg.err, stateSelectTheater, false)
		}
		m.days = msg.days
		if len(m.days) == 0 {
			return m, errWithOptionsCmd(
				fmt.Errorf("no sessions found for this theater on %s", m.date.Format(time.DateOnly)),
				stateSelectTheater,
				true,
			)
		}
		m.browsingAllTheaters = false
		m.movieList.Title = "Select Movie"
		m.movieList.SetItems(buildMovieItems(selectDay(m.days, m.date)))
		m.state = stateSelectMovie
		return m, nil

	case movieCatalogMsg:
		if msg.err != nil {
			return m, errWithOptionsCmd(msg.err, stateSelectTheater, msg.noSessions)
		}
		m.browsingAllTheaters = true
		m.movieList.Title = "Select Movie • All Theaters"
		m.movieList.SetItems(buildMovieItemsFromCatalog(msg.movies))
		m.state = stateSelectMovie
		return m, nil

	case locationMsg:
		if msg.err != nil {
			return m, errCmd(msg.err)
		}
		m.userLocation = &msg.location
		m.refreshTheaterLists()
		return m, nil

	case seatCountMsg:
		m.seatCounts[msg.sessionID] = msg.count
		if m.state == stateShowSessions {
			if cmd := m.updateSessionCount(msg.sessionID, msg.count); cmd != nil {
				return m, cmd
			}
		}
		return m, nil

	case sessionDetailsMsg:
		if msg.err != nil {
			return m, errCmd(msg.err)
		}
		sections := filterSeatSections(msg.detail.Sections)
		if len(sections) == 0 {
			return m, errCmd(errors.New("no seat map available for this session"))
		}
		if len(sections) == 1 {
			m.selectedSection = sections[0]
			m.state = stateLoadingSeatMap
			return m, tea.Batch(m.fetchSeatMapCmd(m.selectedSession.Id, m.selectedSection.Id), m.spinner.Tick)
		}
		m.sectionList.SetItems(buildSectionItems(sections))
		m.state = stateSelectSection
		return m, nil

	case seatMapMsg:
		if msg.err != nil {
			return m, errCmd(msg.err)
		}
		m.seatMap = msg.seatMap
		m.state = stateShowSeatMap
		return m, nil
	}

	var cmd tea.Cmd
	switch m.state {
	case stateSelectCity:
		m.cityList, cmd = m.cityList.Update(msg)
	case stateSelectTheater:
		m.theaterList, cmd = m.theaterList.Update(msg)
	case stateManageTheaters:
		m.theaterPref, cmd = m.theaterPref.Update(msg)
	case stateSelectMovie:
		m.movieList, cmd = m.movieList.Update(msg)
	case stateShowSessions:
		m.sessionList, cmd = m.sessionList.Update(msg)
		if lazyCmd := m.startSeatCountFetchForVisiblePage(); lazyCmd != nil {
			if cmd != nil {
				return m, tea.Batch(cmd, lazyCmd)
			}
			return m, lazyCmd
		}
	case stateSelectSection:
		m.sectionList, cmd = m.sectionList.Update(msg)
	case stateSelectDate:
		m.dateList, cmd = m.dateList.Update(msg)
	}
	return m, cmd
}

func (m appModel) View() string {
	header := m.headerView()
	switch m.state {
	case stateLoadingCities, stateLoadingTheaters, stateLoadingSessions, stateLoadingSeatMap:
		return header + "\n\n" + m.loadingView()
	case stateSelectCity:
		return header + "\n\n" + m.cityList.View()
	case stateSelectTheater:
		return header + "\n\n" + m.theaterList.View()
	case stateManageTheaters:
		return header + "\n\n" + m.theaterPref.View()
	case stateSelectMovie:
		return header + "\n\n" + m.movieList.View()
	case stateShowSessions:
		return header + "\n\n" + m.sessionList.View()
	case stateSelectSection:
		return header + "\n\n" + m.sectionList.View()
	case stateShowSeatMap:
		return header + "\n\n" + m.renderSeatMap()
	case stateSelectDate:
		return header + "\n\n" + m.dateList.View()
	case stateError:
		if m.errorSuggestNextDay {
			return header + "\n\n" + m.errorRecoveryView()
		}
		return header + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.err.Error()) + "\n\n" + hint("Press esc to go back or ctrl+c to quit.")
	default:
		return header
	}
}

func (m appModel) headerView() string {
	title := lipgloss.NewStyle().Bold(true).Render("Ingresso TUI")
	sub := []string{}
	if m.city.Name != "" {
		sub = append(sub, fmt.Sprintf("City: %s", m.city.Name))
	}
	if m.theater.Name != "" {
		sub = append(sub, fmt.Sprintf("Theater: %s", m.theater.Name))
	}
	if m.browsingAllTheaters {
		sub = append(sub, "Mode: movie across theaters")
	}
	if m.userLocation != nil {
		label := locationLabel(m.userLocation)
		if label != "" {
			sub = append(sub, "Location: "+label)
		}
	}
	if !m.date.IsZero() && (m.state == stateSelectCity || m.state == stateSelectTheater || m.state == stateManageTheaters || m.state == stateSelectMovie || m.state == stateShowSessions || m.state == stateSelectDate || m.state == stateShowSeatMap) {
		sub = append(sub, fmt.Sprintf("Date: %s", m.date.Format(time.DateOnly)))
	}
	if m.state == stateShowSeatMap || m.state == stateSelectSection {
		if !m.selectedSession.Date.LocalDate.IsZero() {
			sub = append(sub, fmt.Sprintf("Session: %s", m.selectedSession.Date.LocalDate.Format("15:04")))
		}
		if m.selectedSection.Name != "" {
			sub = append(sub, fmt.Sprintf("Section: %s", m.selectedSection.Name))
		}
	}
	meta := strings.Join(sub, " • ")
	if meta != "" {
		meta = "\n" + lipgloss.NewStyle().Faint(true).Render(meta)
	}
	hints := "ctrl+c quit • esc back • type to filter • ctrl+d pick date"
	if m.state == stateSelectTheater {
		hints = "ctrl+c quit • esc back • type to filter • ctrl+d pick date • enter select • ctrl+f movie across theaters • ctrl+t manage theaters • ctrl+l detect location"
	}
	if m.state == stateShowSessions {
		hints = "ctrl+c quit • esc back • type to filter • ctrl+d pick date • enter open checkout • tab seat map"
	}
	if m.state == stateManageTheaters {
		hints = "ctrl+c quit • esc back • type to filter • enter/x toggle theater visibility"
	}
	if m.state == stateSelectDate {
		hints = "ctrl+c quit • esc back • enter select date"
	}
	if m.state == stateShowSeatMap {
		hints = "ctrl+c quit • esc back • n toggle numbers"
	}
	filterLine := ""
	if listPtr := m.activeList(); listPtr != nil {
		if filter := listPtr.FilterValue(); filter != "" {
			filterLine = "\n" + hint(fmt.Sprintf("Filter: %s", filter))
		}
	}
	return title + meta + filterLine + "\n" + hint(hints)
}

func (m appModel) errorRecoveryView() string {
	nextDate := truncateDate(m.date.AddDate(0, 0, 1))
	headerChip := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("63")).
		Padding(0, 2)
	actionChip := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("63")).
		Width(8).
		Align(lipgloss.Center).
		Padding(0, 1)
	actionText := lipgloss.NewStyle().Bold(true)

	title := headerChip.Render("Sem Sessoes")
	message := lipgloss.NewStyle().
		Foreground(lipgloss.Color("203")).
		Bold(true).
		Render(fmt.Sprintf("Nenhuma sessao foi encontrada para %s.", m.date.Format(time.DateOnly)))
	sub := hint("Pressione ENTER para tentar no proximo dia (amanha), ou CTRL+D para escolher outra data.")

	enterAction := lipgloss.JoinHorizontal(
		lipgloss.Top,
		actionChip.Render("ENTER"),
		"  ",
		actionText.Render(fmt.Sprintf("Tentar %s (amanha)", nextDate.Format(time.DateOnly))),
	)
	dateAction := lipgloss.JoinHorizontal(
		lipgloss.Top,
		actionChip.Render("CTRL+D"),
		"  ",
		"Escolher qualquer outra data (ex.: amanha)",
	)
	footer := hint("ESC voltar • CTRL+C sair")

	content := strings.Join([]string{
		title,
		"",
		message,
		"",
		sub,
		"",
		enterAction,
		"",
		dateAction,
		"",
		footer,
	}, "\n")

	panelStyle := lipgloss.NewStyle().
		Padding(1, 3).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63")).
		MarginTop(1)
	if m.width > 56 {
		cardWidth := m.width - 8
		if cardWidth > 84 {
			cardWidth = 84
		}
		panelStyle = panelStyle.Width(cardWidth)
	}
	panel := panelStyle.Render(content)
	if m.width > 0 {
		panel = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, panel)
	}

	return lipgloss.NewStyle().
		Padding(0, 1).
		Render(panel)
}

func (m appModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit, true
	case "esc":
		if listPtr := m.activeList(); listPtr != nil {
			if listPtr.SettingFilter() || listPtr.IsFiltered() {
				listPtr.ResetFilter()
				if m.state == stateShowSessions {
					return m, m.startSeatCountFetchForVisiblePage(), true
				}
				return m, nil, true
			}
		}
		model, cmd := m.goBack()
		return model, cmd, true
	case "n":
		if m.state == stateShowSeatMap {
			m.showSeatNumbers = !m.showSeatNumbers
			return m, nil, true
		}
	case "tab":
		if m.state == stateShowSessions {
			return m.openSeatMapFromSelection()
		}
	case "ctrl+l":
		if m.state == stateSelectTheater || m.state == stateManageTheaters {
			return m, m.detectLocationCmd(), true
		}
	case "x":
		if m.state == stateManageTheaters {
			return m.toggleTheaterVisibility()
		}
	case "ctrl+f":
		if m.state == stateSelectTheater {
			return m.openMovieAcrossTheaters()
		}
	}

	if msg.String() == "ctrl+t" && m.state == stateSelectTheater {
		m.state = stateManageTheaters
		m.refreshTheaterLists()
		return m, nil, true
	}

	if msg.String() == "ctrl+d" && (m.state == stateSelectCity || m.state == stateSelectTheater || m.state == stateSelectMovie || m.state == stateShowSessions) {
		m.openDatePicker(m.state)
		return m, nil, true
	}
	if msg.String() == "ctrl+d" && m.state == stateError && m.errorSuggestNextDay {
		m.openDatePicker(stateShowSessions)
		return m, nil, true
	}

	if msg.Type == tea.KeyEnter {
		if m.state == stateError && m.errorSuggestNextDay {
			return m.advanceToNextDayFromError()
		}
		switch m.state {
		case stateSelectCity:
			item, ok := m.cityList.SelectedItem().(cityItem)
			if !ok {
				return m, nil, true
			}
			m.city = item.city
			_ = store.RememberCity(m.city)
			m.state = stateLoadingTheaters
			return m, tea.Batch(m.fetchTheatersCmd(m.city.Id), m.spinner.Tick), true
		case stateSelectTheater:
			item, ok := m.theaterList.SelectedItem().(theaterItem)
			if !ok {
				return m, nil, true
			}
			m.theater = item.theater
			m.browsingAllTheaters = false
			_ = store.RememberTheater(m.city.Id, m.theater)
			m.state = stateLoadingSessions
			return m, tea.Batch(m.fetchSessionsCmd(m.city.Id, m.theater.Id, m.date), m.spinner.Tick), true
		case stateSelectMovie:
			item, ok := m.movieList.SelectedItem().(movieItem)
			if !ok {
				return m, nil, true
			}
			m.sessionList.Title = fmt.Sprintf("Sessions • %s", item.movie.Title)
			var items []list.Item
			if len(item.globalSessions) > 0 {
				items, _ = buildGlobalSessionItems(item.globalSessions, m.seatCounts)
			} else {
				items, _ = buildSessionItems(item.movie, m.seatCounts)
			}
			m.sessionList.SetItems(items)
			m.state = stateShowSessions
			if cmd := m.startSeatCountFetchForVisiblePage(); cmd != nil {
				return m, cmd, true
			}
			return m, nil, true
		case stateManageTheaters:
			return m.toggleTheaterVisibility()
		case stateShowSessions:
			item, ok := m.sessionList.SelectedItem().(sessionItem)
			if !ok {
				return m, nil, true
			}
			url := fmt.Sprintf("https://checkout.ingresso.com/assentos?sessionId=%s&partnership=home", item.session.Id)
			return m, openURLCmd(url), true
		case stateSelectSection:
			item, ok := m.sectionList.SelectedItem().(sectionItem)
			if !ok {
				return m, nil, true
			}
			m.selectedSection = item.section
			m.state = stateLoadingSeatMap
			return m, tea.Batch(m.fetchSeatMapCmd(m.selectedSession.Id, m.selectedSection.Id), m.spinner.Tick), true
		case stateSelectDate:
			item, ok := m.dateList.SelectedItem().(dateItem)
			if !ok {
				return m, nil, true
			}
			m.date = item.date
			if m.dateReturnStateSet {
				switch m.dateReturnState {
				case stateSelectCity, stateSelectTheater:
					m.state = m.dateReturnState
					m.dateReturnStateSet = false
					return m, nil, true
				case stateSelectMovie, stateShowSessions:
					if m.browsingAllTheaters {
						visible := m.visibleTheaters()
						if len(visible) == 0 {
							return m, errCmd(errors.New("no visible theaters selected")), true
						}
						m.state = stateLoadingSessions
						m.dateReturnStateSet = false
						return m, tea.Batch(m.fetchMovieCatalogCmd(m.city.Id, visible, m.date), m.spinner.Tick), true
					}
				}
			}
			m.state = stateLoadingSessions
			m.dateReturnStateSet = false
			return m, tea.Batch(m.fetchSessionsCmd(m.city.Id, m.theater.Id, m.date), m.spinner.Tick), true
		}
	}
	return m, nil, false
}

func (m appModel) openSeatMapFromSelection() (tea.Model, tea.Cmd, bool) {
	item, ok := m.sessionList.SelectedItem().(sessionItem)
	if !ok {
		return m, nil, true
	}
	if !item.session.HasSeatSelection {
		return m, errCmd(errors.New("this session does not support seat selection")), true
	}
	m.selectedSession = item.session
	m.state = stateLoadingSeatMap
	return m, tea.Batch(m.fetchSessionDetailsCmd(item.session.Id), m.spinner.Tick), true
}

func (m appModel) goBack() (tea.Model, tea.Cmd) {
	switch m.state {
	case stateSelectTheater:
		if len(m.cityList.Items()) == 0 {
			m.state = stateLoadingCities
			return m, tea.Batch(m.fetchCitiesCmd(), m.spinner.Tick)
		}
		m.state = stateSelectCity
	case stateSelectMovie:
		m.state = stateSelectTheater
	case stateShowSessions:
		m.state = stateSelectMovie
	case stateManageTheaters:
		m.state = stateSelectTheater
	case stateSelectSection:
		m.state = stateShowSessions
	case stateShowSeatMap:
		m.state = stateShowSessions
	case stateSelectDate:
		if m.dateReturnStateSet {
			m.state = m.dateReturnState
			m.dateReturnStateSet = false
		} else {
			m.state = stateShowSessions
		}
	case stateError:
		m.state = m.lastState
		m.errorSuggestNextDay = false
	default:
		return m, nil
	}
	return m, nil
}

func (m *appModel) handleFilterInput(msg tea.KeyMsg) bool {
	listPtr := m.activeList()
	if listPtr == nil {
		return false
	}
	if !listPtr.FilteringEnabled() {
		return false
	}
	switch msg.Type {
	case tea.KeyRunes:
		if len(msg.Runes) == 0 {
			return false
		}
		m.appendFilter(listPtr, string(msg.Runes))
		return true
	case tea.KeySpace:
		m.appendFilter(listPtr, " ")
		return true
	case tea.KeyBackspace, tea.KeyDelete:
		if listPtr.FilterValue() == "" {
			return false
		}
		m.popFilter(listPtr)
		return true
	default:
		return false
	}
}

func (m *appModel) appendFilter(listPtr *list.Model, value string) {
	if value == "" {
		return
	}
	current := listPtr.FilterValue()
	listPtr.SetFilterText(current + value)
}

func (m *appModel) popFilter(listPtr *list.Model) {
	value := listPtr.FilterValue()
	if value == "" {
		return
	}
	value = trimLastRune(value)
	if value == "" {
		listPtr.ResetFilter()
		return
	}
	listPtr.SetFilterText(value)
}

func (m *appModel) openDatePicker(returnState appState) {
	m.dateReturnState = returnState
	m.dateReturnStateSet = true
	m.state = stateSelectDate
	m.dateList.SetItems(buildDateItems(m.date))
}

func trimLastRune(value string) string {
	runes := []rune(value)
	if len(runes) <= 1 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func (m *appModel) activeList() *list.Model {
	switch m.state {
	case stateSelectCity:
		return &m.cityList
	case stateSelectTheater:
		return &m.theaterList
	case stateManageTheaters:
		return &m.theaterPref
	case stateSelectMovie:
		return &m.movieList
	case stateShowSessions:
		return &m.sessionList
	case stateSelectSection:
		return &m.sectionList
	default:
		return nil
	}
}

func (m appModel) isLoadingState() bool {
	return m.state == stateLoadingCities ||
		m.state == stateLoadingTheaters ||
		m.state == stateLoadingSessions ||
		m.state == stateLoadingSeatMap
}

func (m appModel) loadingView() string {
	title := "Loading"
	switch m.state {
	case stateLoadingCities:
		title = "Loading cities"
	case stateLoadingTheaters:
		title = "Loading theaters"
	case stateLoadingSessions:
		title = "Loading sessions"
	case stateLoadingSeatMap:
		title = "Loading seat map"
	}

	return fmt.Sprintf("%s %s\n\n%s", m.spinner.View(), title, hint("Fetching data..."))
}

func (m *appModel) resizeLists() {
	if m.width == 0 || m.height == 0 {
		return
	}
	h := m.height - 6
	if h < 6 {
		h = 6
	}
	m.cityList.SetSize(m.width, h)
	m.theaterList.SetSize(m.width, h)
	m.theaterPref.SetSize(m.width, h)
	m.movieList.SetSize(m.width, h)
	m.sessionList.SetSize(m.width, h)
	m.sectionList.SetSize(m.width, h)
	m.dateList.SetSize(m.width, h)
}

func newList(title string) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = title
	l.Filter = caseInsensitiveFilter
	l.SetFilteringEnabled(true)
	l.SetShowFilter(true)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	return l
}

func hint(text string) string {
	return lipgloss.NewStyle().Faint(true).Render(text)
}

func errCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return errMsg{
			err:            err,
			returnState:    0,
			returnStateSet: false,
			suggestNextDay: false,
		}
	}
}

func errWithOptionsCmd(err error, returnState appState, suggestNextDay bool) tea.Cmd {
	return func() tea.Msg {
		return errMsg{
			err:            err,
			returnState:    returnState,
			returnStateSet: true,
			suggestNextDay: suggestNextDay,
		}
	}
}

func recoverStateFrom(state appState) appState {
	switch state {
	case stateLoadingCities:
		return stateSelectCity
	case stateLoadingTheaters:
		return stateSelectCity
	case stateLoadingSessions:
		return stateSelectTheater
	case stateLoadingSeatMap:
		return stateShowSessions
	case stateError:
		return stateSelectTheater
	default:
		return state
	}
}

func truncateDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func caseInsensitiveFilter(term string, targets []string) []list.Rank {
	term = strings.ToLower(term)
	lower := make([]string, len(targets))
	for i, t := range targets {
		lower[i] = strings.ToLower(t)
	}
	return list.DefaultFilter(term, lower)
}

func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := openURL(url); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func openURL(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported OS for opening browser: %s", runtime.GOOS)
	}
}

func (m appModel) fetchCityByNameCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		city, err := m.client.GetCityInfoByName(ctx, name)
		if err != nil {
			return cityMsg{err: err}
		}
		return cityMsg{city: city, err: nil}
	}
}

func (m appModel) fetchRecentCityCmd(recent store.RecentCity) tea.Cmd {
	return func() tea.Msg {
		if city, ok := cityFromRecentCache(recent); ok {
			return cityMsg{city: city, err: nil}
		}
		if strings.TrimSpace(recent.Name) == "" {
			return cityMsg{err: errors.New("recent city not found")}
		}
		ctx := context.Background()
		city, err := m.client.GetCityInfoByName(ctx, recent.Name)
		if err != nil {
			return cityMsg{err: err}
		}
		return cityMsg{city: city, err: nil}
	}
}

func startupRecentCity() (store.RecentCity, bool) {
	recents, err := store.LoadRecentCities()
	if err != nil || len(recents) == 0 {
		return store.RecentCity{}, false
	}
	recent := recents[0]
	if strings.TrimSpace(recent.ID) == "" && strings.TrimSpace(recent.Name) == "" {
		return store.RecentCity{}, false
	}
	return recent, true
}

func cityFromRecentCache(recent store.RecentCity) (model.City, bool) {
	cached, _, err := store.LoadCityCache()
	if err != nil || len(cached) == 0 {
		return model.City{}, false
	}
	recentID := strings.TrimSpace(recent.ID)
	recentName := strings.TrimSpace(recent.Name)
	recentUF := strings.TrimSpace(recent.UF)
	for _, city := range cached {
		if recentID != "" && city.Id == recentID {
			return city, true
		}
		if recentName != "" && strings.EqualFold(city.Name, recentName) {
			if recentUF == "" || strings.EqualFold(city.Uf, recentUF) {
				return city, true
			}
		}
	}
	return model.City{}, false
}

func (m appModel) fetchCitiesCmd() tea.Cmd {
	return func() tea.Msg {
		if cached, fresh, err := store.LoadCityCache(); err == nil && fresh && len(cached) > 0 {
			return citiesMsg{cities: cached}
		}
		ctx := context.Background()
		cities, err := m.client.GetCities(ctx)
		if err == nil && len(cities) > 0 {
			_ = store.SaveCityCache(cities)
		}
		return citiesMsg{cities: cities, err: err}
	}
}

func (m appModel) fetchTheatersCmd(cityID string) tea.Cmd {
	return func() tea.Msg {
		if cached, fresh, err := store.LoadTheaterCache(cityID); err == nil && fresh && len(cached) > 0 {
			return theatersMsg{theaters: cached}
		}
		ctx := context.Background()
		theaters, err := m.client.GetTheatersByCity(ctx, cityID)
		if err == nil && len(theaters) > 0 {
			_ = store.SaveTheaterCache(cityID, theaters)
		}
		return theatersMsg{theaters: theaters, err: err}
	}
}

func (m appModel) fetchSessionsCmd(cityID string, theaterID string, date time.Time) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		days, err := m.loadSessionsByTheater(ctx, cityID, theaterID, date)
		if err != nil {
			if service.IsNotFound(err) {
				return sessionsMsg{days: nil, err: nil}
			}
			return sessionsMsg{days: nil, err: err}
		}
		return sessionsMsg{days: days, err: err}
	}
}

func (m appModel) loadSessionsByTheater(ctx context.Context, cityID string, theaterID string, date time.Time) ([]model.TheaterSessionDay, error) {
	dateKey := date.Format(time.DateOnly)
	if cached, fresh, err := store.LoadSessionCache(cityID, theaterID, dateKey); err == nil && fresh && len(cached) > 0 {
		return cached, nil
	}
	days, err := m.client.GetSessionsByCityAndTheater(ctx, cityID, theaterID, &date)
	if err != nil {
		return nil, err
	}
	if len(days) > 0 {
		_ = store.SaveSessionCache(cityID, theaterID, dateKey, days)
	}
	return days, nil
}

func (m appModel) fetchMovieCatalogCmd(cityID string, theaters []model.Theater, date time.Time) tea.Cmd {
	return func() tea.Msg {
		if len(theaters) == 0 {
			return movieCatalogMsg{err: errors.New("no theaters available")}
		}

		ctx := context.Background()
		out := make(chan theaterSessionsResult, len(theaters))
		sem := make(chan struct{}, 6)
		var wg sync.WaitGroup

		for _, theater := range theaters {
			wg.Add(1)
			go func(theater model.Theater) {
				defer wg.Done()
				sem <- struct{}{}
				days, err := m.loadSessionsByTheater(ctx, cityID, theater.Id, date)
				<-sem
				out <- theaterSessionsResult{theater: theater, days: days, err: err}
			}(theater)
		}

		wg.Wait()
		close(out)

		movies, failed, ignored := aggregateMovieCatalog(out, date, m.userLocation)
		if len(movies) == 0 {
			return movieCatalogMsg{
				err:        fmt.Errorf("no sessions found in visible theaters on %s", date.Format(time.DateOnly)),
				failed:     failed,
				ignored:    ignored,
				noSessions: true,
			}
		}
		return movieCatalogMsg{movies: movies, failed: failed, ignored: ignored}
	}
}

func (m appModel) detectLocationCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		location, err := service.DetectCurrentLocation(ctx, nil)
		if err != nil {
			return locationMsg{err: fmt.Errorf("failed to detect current location: %w", err)}
		}
		return locationMsg{location: location}
	}
}

func (m appModel) fetchSessionDetailsCmd(sessionID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		detail, err := m.client.GetSessionDetails(ctx, sessionID)
		return sessionDetailsMsg{detail: detail, err: err}
	}
}

func (m appModel) fetchSeatMapCmd(sessionID string, sectionID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		seatMap, err := m.client.GetSeatMap(ctx, sessionID, sectionID)
		return seatMapMsg{seatMap: seatMap, err: err}
	}
}

type dateItem struct {
	date time.Time
}

func (d dateItem) Title() string {
	if isSameDay(d.date, time.Now()) {
		return fmt.Sprintf("%s • %s (Today)", d.date.Format("Mon"), d.date.Format("02/01"))
	}
	return fmt.Sprintf("%s • %s", d.date.Format("Mon"), d.date.Format("02/01"))
}

func (d dateItem) Description() string {
	return d.date.Format(time.DateOnly)
}

func (d dateItem) FilterValue() string {
	return d.Title()
}

func buildDateItems(base time.Time) []list.Item {
	start := truncateDate(base)
	items := make([]list.Item, 0, 5)
	for i := 0; i < 5; i++ {
		items = append(items, dateItem{date: start.AddDate(0, 0, i)})
	}
	return items
}

func isSameDay(a time.Time, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func (m *appModel) startSeatCountFetchForVisiblePage() tea.Cmd {
	ids := m.pendingSeatCountSessionIDsOnCurrentPage()
	if len(ids) == 0 {
		return nil
	}

	cmds := make([]tea.Cmd, 0, len(ids))
	for _, id := range ids {
		m.seatCounts[id] = seatCount{}
		cmds = append(cmds, m.fetchSeatCountCmd(id))
	}
	return tea.Batch(cmds...)
}

func (m appModel) pendingSeatCountSessionIDsOnCurrentPage() []string {
	if m.state != stateShowSessions {
		return nil
	}

	items := m.sessionList.VisibleItems()
	if len(items) == 0 {
		return nil
	}
	start, end := m.sessionList.Paginator.GetSliceBounds(len(items))
	if start < 0 || end > len(items) || start >= end {
		return nil
	}

	ids := make([]string, 0, end-start)
	for _, item := range items[start:end] {
		si, ok := item.(sessionItem)
		if !ok || !si.session.HasSeatSelection || si.session.Id == "" {
			continue
		}
		if _, alreadyRequested := m.seatCounts[si.session.Id]; alreadyRequested {
			continue
		}
		ids = append(ids, si.session.Id)
	}
	return ids
}

func (m appModel) fetchSeatCountCmd(sessionID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		detail, err := m.client.GetSessionDetails(ctx, sessionID)
		if err != nil {
			return seatCountMsg{sessionID: sessionID, count: seatCount{loaded: true, err: err}}
		}
		sections := filterSeatSections(detail.Sections)
		if len(sections) == 0 {
			return seatCountMsg{sessionID: sessionID, count: seatCount{loaded: true}}
		}
		var total seatCount
		total.loaded = true
		for _, section := range sections {
			seatMap, err := m.client.GetSeatMap(ctx, sessionID, section.Id)
			if err != nil {
				total.err = err
				continue
			}
			part := computeSeatCounts(seatMap)
			total.available += part.available
			total.occupied += part.occupied
			total.blocked += part.blocked
			total.total += part.total
			total.nonIdealAvailable += part.nonIdealAvailable
			total.idealAvailable += part.idealAvailable
			total.pairAvailable += part.pairAvailable
		}
		return seatCountMsg{sessionID: sessionID, count: total}
	}
}

func (m *appModel) updateSessionCount(sessionID string, count seatCount) tea.Cmd {
	items := m.sessionList.Items()
	for i, item := range items {
		si, ok := item.(sessionItem)
		if !ok {
			continue
		}
		if si.session.Id == sessionID {
			si.count = count
			return m.sessionList.SetItem(i, si)
		}
	}
	return nil
}

func computeSeatCounts(seatMap model.SeatMap) seatCount {
	var result seatCount
	front := frontLineSet(seatMap, 3)
	availableCols := map[int][]int{}
	for _, line := range seatMap.Lines {
		for _, seat := range line.Seats {
			result.total++
			status := strings.ToLower(seat.Status)
			switch status {
			case "available":
				result.available++
				if front[seat.Line] {
					result.nonIdealAvailable++
				}
				if seat.Column > 0 {
					availableCols[seat.Line] = append(availableCols[seat.Line], seat.Column)
				}
			case "occupied":
				result.occupied++
			case "blocked", "unavailable":
				result.blocked++
			}
		}
	}
	result.idealAvailable = max(0, result.available-result.nonIdealAvailable)
	result.pairAvailable = countAdjacentPairsFromCols(availableCols)
	return result
}

type sessionWithTheater struct {
	session     model.TheaterSession
	theater     model.Theater
	hasDistance bool
	distanceKM  float64
}

type movieAggregate struct {
	movie    model.TheaterMovie
	sessions []sessionWithTheater
}

type cityItem struct {
	city   model.City
	recent bool
}

func (c cityItem) Title() string {
	if c.city.Uf != "" {
		return fmt.Sprintf("%s (%s)", c.city.Name, c.city.Uf)
	}
	return c.city.Name
}

func (c cityItem) Description() string {
	if c.recent {
		return "Recent"
	}
	if c.city.State != "" {
		return c.city.State
	}
	return ""
}

func (c cityItem) FilterValue() string {
	return strings.ToLower(strings.Join([]string{c.city.Name, c.city.Uf, c.city.State, c.city.UrlKey}, " "))
}

type theaterItem struct {
	theater     model.Theater
	recent      bool
	hasDistance bool
	distanceKM  float64
}

func (t theaterItem) Title() string {
	return t.theater.Name
}

func (t theaterItem) Description() string {
	parts := []string{}
	if t.recent {
		parts = append(parts, "Recent")
	}
	if t.theater.Neighborhood != "" {
		parts = append(parts, t.theater.Neighborhood)
	} else if t.theater.Address != "" {
		parts = append(parts, t.theater.Address)
	}
	if t.hasDistance {
		parts = append(parts, fmt.Sprintf("%.1f km", t.distanceKM))
	}
	return strings.Join(parts, " • ")
}

func (t theaterItem) FilterValue() string {
	return strings.ToLower(strings.Join([]string{t.theater.Name, t.theater.Neighborhood, t.theater.Address}, " "))
}

type movieItem struct {
	movie          model.TheaterMovie
	count          int
	globalSessions []sessionWithTheater
}

func (m movieItem) Title() string {
	return m.movie.Title
}

func (m movieItem) Description() string {
	if m.count > 0 {
		return fmt.Sprintf("%d sessions", m.count)
	}
	return ""
}

func (m movieItem) FilterValue() string {
	return strings.ToLower(strings.Join([]string{m.movie.Title, m.movie.OriginalTitle, m.movie.ContentRating}, " "))
}

type sessionItem struct {
	session     model.TheaterSession
	theaterName string
	hasDistance bool
	distanceKM  float64
	count       seatCount
}

func (s sessionItem) Title() string {
	timeLabel := s.session.Date.LocalDate.Format("15:04")
	room := strings.TrimSpace(s.session.Room)
	if room == "" {
		room = "Sala"
	}
	if s.theaterName != "" {
		return fmt.Sprintf("%s • %s • %s", timeLabel, s.theaterName, room)
	}
	return fmt.Sprintf("%s • %s", timeLabel, room)
}

func (s sessionItem) Description() string {
	types := formatSessionTypes(s.session.Type)
	full := formatPrice(s.session.Price)
	half := formatPrice(halfPrice(s.session.Price))
	prefix := ""
	if s.hasDistance {
		prefix = fmt.Sprintf("%.1f km • ", s.distanceKM)
	}
	seatHint := ""
	if s.session.HasSeatSelection {
		seatHint = " • seats ..."
		if s.count.loaded && s.count.err == nil {
			if s.count.nonIdealAvailable > 0 {
				seatHint = fmt.Sprintf(" • seats %d (ideal %d • front %d • pairs %d)", s.count.available, s.count.idealAvailable, s.count.nonIdealAvailable, s.count.pairAvailable)
			} else {
				seatHint = fmt.Sprintf(" • seats %d (ideal %d • pairs %d)", s.count.available, s.count.idealAvailable, s.count.pairAvailable)
			}
		} else if s.count.err != nil {
			seatHint = " • seats n/a"
		}
	}
	return fmt.Sprintf("%s%s • Full %s • Half %s%s", prefix, types, full, half, seatHint)
}

func (s sessionItem) FilterValue() string {
	parts := append(s.session.Type, s.session.Room, s.theaterName)
	return strings.ToLower(strings.Join(parts, " "))
}

type theaterVisibilityItem struct {
	theater     model.Theater
	hidden      bool
	hasDistance bool
	distanceKM  float64
}

func (t theaterVisibilityItem) Title() string {
	if t.hidden {
		return fmt.Sprintf("[ ] %s", t.theater.Name)
	}
	return fmt.Sprintf("[x] %s", t.theater.Name)
}

func (t theaterVisibilityItem) Description() string {
	parts := []string{}
	if t.theater.Neighborhood != "" {
		parts = append(parts, t.theater.Neighborhood)
	}
	if t.hasDistance {
		parts = append(parts, fmt.Sprintf("%.1f km", t.distanceKM))
	}
	if t.hidden {
		parts = append(parts, "hidden")
	} else {
		parts = append(parts, "visible")
	}
	return strings.Join(parts, " • ")
}

func (t theaterVisibilityItem) FilterValue() string {
	return strings.ToLower(strings.Join([]string{
		t.theater.Name,
		t.theater.Neighborhood,
		t.theater.Address,
	}, " "))
}

func buildCityItems(cities []model.City) []list.Item {
	recents, _ := store.LoadRecentCities()
	byID := map[string]model.City{}
	byName := map[string]model.City{}
	for _, city := range cities {
		byID[city.Id] = city
		byName[strings.ToLower(city.Name)] = city
	}

	var items []list.Item
	used := map[string]bool{}
	for _, recent := range recents {
		if recent.ID != "" {
			if city, ok := byID[recent.ID]; ok {
				items = append(items, cityItem{city: city, recent: true})
				used[city.Id] = true
				continue
			}
		}
		if recent.Name != "" {
			if city, ok := byName[strings.ToLower(recent.Name)]; ok && !used[city.Id] {
				items = append(items, cityItem{city: city, recent: true})
				used[city.Id] = true
			}
		}
	}

	remaining := make([]model.City, 0, len(cities))
	for _, city := range cities {
		if !used[city.Id] {
			remaining = append(remaining, city)
		}
	}

	sort.Slice(remaining, func(i, j int) bool {
		return strings.ToLower(remaining[i].Name) < strings.ToLower(remaining[j].Name)
	})

	for _, city := range remaining {
		items = append(items, cityItem{city: city})
	}
	return items
}

func buildTheaterItems(theaters []model.Theater, cityID string, hidden map[string]bool, userLocation *service.UserLocation) []list.Item {
	recents, _ := store.LoadRecentTheaters()

	visible := make([]model.Theater, 0, len(theaters))
	for _, theater := range theaters {
		if hidden[theater.Id] {
			continue
		}
		visible = append(visible, theater)
	}

	items := make([]list.Item, 0, len(visible))

	if userLocation != nil {
		sortTheatersByDistance(visible, userLocation)
		for _, theater := range visible {
			distance, hasDistance := theaterDistanceKM(theater, userLocation)
			items = append(items, theaterItem{
				theater:     theater,
				recent:      isRecentTheater(theater, cityID, recents),
				hasDistance: hasDistance,
				distanceKM:  distance,
			})
		}
		return items
	}

	byID := map[string]model.Theater{}
	byName := map[string]model.Theater{}
	for _, theater := range visible {
		byID[theater.Id] = theater
		byName[strings.ToLower(theater.Name)] = theater
	}

	used := map[string]bool{}
	for _, recent := range recents {
		if recent.CityID != "" && recent.CityID != cityID {
			continue
		}
		if recent.TheaterID != "" {
			if theater, ok := byID[recent.TheaterID]; ok {
				items = append(items, theaterItem{theater: theater, recent: true})
				used[theater.Id] = true
				continue
			}
		}
		if recent.Name != "" {
			if theater, ok := byName[strings.ToLower(recent.Name)]; ok && !used[theater.Id] {
				items = append(items, theaterItem{theater: theater, recent: true})
				used[theater.Id] = true
			}
		}
	}

	remaining := make([]model.Theater, 0, len(visible))
	for _, theater := range visible {
		if !used[theater.Id] {
			remaining = append(remaining, theater)
		}
	}

	sort.Slice(remaining, func(i, j int) bool {
		return strings.ToLower(remaining[i].Name) < strings.ToLower(remaining[j].Name)
	})

	for _, theater := range remaining {
		items = append(items, theaterItem{theater: theater})
	}
	return items
}

func buildTheaterVisibilityItems(theaters []model.Theater, hidden map[string]bool, userLocation *service.UserLocation) []list.Item {
	sorted := append([]model.Theater{}, theaters...)
	if userLocation != nil {
		sortTheatersByDistance(sorted, userLocation)
	} else {
		sort.Slice(sorted, func(i, j int) bool {
			return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
		})
	}

	items := make([]list.Item, 0, len(sorted))
	for _, theater := range sorted {
		distance, hasDistance := theaterDistanceKM(theater, userLocation)
		items = append(items, theaterVisibilityItem{
			theater:     theater,
			hidden:      hidden[theater.Id],
			hasDistance: hasDistance,
			distanceKM:  distance,
		})
	}
	return items
}

func buildMovieItems(day model.TheaterSessionDay) []list.Item {
	var items []list.Item
	for _, movie := range day.Movies {
		count := 0
		for _, room := range movie.Rooms {
			count += len(room.Sessions)
		}
		items = append(items, movieItem{movie: movie, count: count})
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].(movieItem).movie.Title) < strings.ToLower(items[j].(movieItem).movie.Title)
	})
	return items
}

func buildMovieItemsFromCatalog(movies []movieAggregate) []list.Item {
	items := make([]list.Item, 0, len(movies))
	for _, movie := range movies {
		items = append(items, movieItem{
			movie:          movie.movie,
			count:          len(movie.sessions),
			globalSessions: movie.sessions,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		left := items[i].(movieItem).movie.Title
		right := items[j].(movieItem).movie.Title
		return strings.ToLower(left) < strings.ToLower(right)
	})
	return items
}

func buildSessionItems(movie model.TheaterMovie, counts map[string]seatCount) ([]list.Item, []model.TheaterSession) {
	var sessions []model.TheaterSession
	for _, room := range movie.Rooms {
		for _, session := range room.Sessions {
			if strings.TrimSpace(session.Room) == "" {
				session.Room = room.Name
			}
			sessions = append(sessions, session)
		}
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Date.LocalDate.Before(sessions[j].Date.LocalDate)
	})

	items := make([]list.Item, 0, len(sessions))
	for _, session := range sessions {
		count := counts[session.Id]
		items = append(items, sessionItem{session: session, count: count})
	}
	return items, sessions
}

func buildGlobalSessionItems(sessions []sessionWithTheater, counts map[string]seatCount) ([]list.Item, []model.TheaterSession) {
	sorted := append([]sessionWithTheater{}, sessions...)
	sort.Slice(sorted, func(i, j int) bool {
		a := sorted[i]
		b := sorted[j]

		if a.hasDistance && b.hasDistance && math.Abs(a.distanceKM-b.distanceKM) > 1e-6 {
			return a.distanceKM < b.distanceKM
		}
		if a.hasDistance != b.hasDistance {
			return a.hasDistance
		}
		if !a.session.Date.LocalDate.Equal(b.session.Date.LocalDate) {
			return a.session.Date.LocalDate.Before(b.session.Date.LocalDate)
		}
		return strings.ToLower(a.theater.Name) < strings.ToLower(b.theater.Name)
	})

	items := make([]list.Item, 0, len(sorted))
	plain := make([]model.TheaterSession, 0, len(sorted))
	for _, entry := range sorted {
		count := counts[entry.session.Id]
		items = append(items, sessionItem{
			session:     entry.session,
			theaterName: entry.theater.Name,
			hasDistance: entry.hasDistance,
			distanceKM:  entry.distanceKM,
			count:       count,
		})
		plain = append(plain, entry.session)
	}
	return items, plain
}

func (m *appModel) refreshTheaterLists() {
	m.theaterList.SetItems(buildTheaterItems(m.theaters, m.city.Id, m.hiddenTheaters, m.userLocation))
	m.theaterPref.SetItems(buildTheaterVisibilityItems(m.theaters, m.hiddenTheaters, m.userLocation))
}

func (m appModel) visibleTheaters() []model.Theater {
	visible := make([]model.Theater, 0, len(m.theaters))
	for _, theater := range m.theaters {
		if m.hiddenTheaters[theater.Id] {
			continue
		}
		visible = append(visible, theater)
	}
	return visible
}

func (m appModel) openMovieAcrossTheaters() (tea.Model, tea.Cmd, bool) {
	visible := m.visibleTheaters()
	if len(visible) == 0 {
		return m, errCmd(errors.New("no visible theaters selected")), true
	}
	m.browsingAllTheaters = true
	m.theater = model.Theater{}
	m.state = stateLoadingSessions
	return m, tea.Batch(m.fetchMovieCatalogCmd(m.city.Id, visible, m.date), m.spinner.Tick), true
}

func (m appModel) advanceToNextDayFromError() (tea.Model, tea.Cmd, bool) {
	m.date = truncateDate(m.date.AddDate(0, 0, 1))
	m.state = stateLoadingSessions
	m.errorSuggestNextDay = false

	if m.browsingAllTheaters {
		visible := m.visibleTheaters()
		if len(visible) == 0 {
			return m, errWithOptionsCmd(errors.New("no visible theaters selected"), stateSelectTheater, false), true
		}
		return m, tea.Batch(m.fetchMovieCatalogCmd(m.city.Id, visible, m.date), m.spinner.Tick), true
	}

	if m.city.Id == "" || m.theater.Id == "" {
		return m, errWithOptionsCmd(errors.New("select a theater before trying another date"), stateSelectTheater, false), true
	}
	return m, tea.Batch(m.fetchSessionsCmd(m.city.Id, m.theater.Id, m.date), m.spinner.Tick), true
}

func (m appModel) toggleTheaterVisibility() (tea.Model, tea.Cmd, bool) {
	item, ok := m.theaterPref.SelectedItem().(theaterVisibilityItem)
	if !ok {
		return m, nil, true
	}
	hidden := !item.hidden
	if err := store.SetTheaterHidden(m.city.Id, item.theater.Id, hidden); err != nil {
		return m, errCmd(err), true
	}
	if m.hiddenTheaters == nil {
		m.hiddenTheaters = map[string]bool{}
	}
	if hidden {
		m.hiddenTheaters[item.theater.Id] = true
	} else {
		delete(m.hiddenTheaters, item.theater.Id)
	}
	if hidden && m.theater.Id == item.theater.Id {
		m.theater = model.Theater{}
	}

	index := m.theaterPref.Index()
	m.refreshTheaterLists()
	if count := len(m.theaterPref.Items()); count > 0 {
		if index >= count {
			index = count - 1
		}
		m.theaterPref.Select(index)
	}
	return m, nil, true
}

func aggregateMovieCatalog(results <-chan theaterSessionsResult, date time.Time, userLocation *service.UserLocation) ([]movieAggregate, int, int) {
	byMovie := map[string]*movieAggregate{}
	failed := 0
	ignored := 0

	for result := range results {
		if result.err != nil {
			if service.IsNotFound(result.err) {
				ignored++
				continue
			}
			failed++
			continue
		}

		day := selectDay(result.days, date)
		if len(day.Movies) == 0 {
			ignored++
			continue
		}

		for _, movie := range day.Movies {
			key := movieAggregateKey(movie)
			entry := byMovie[key]
			if entry == nil {
				copyMovie := movie
				copyMovie.Rooms = nil
				entry = &movieAggregate{movie: copyMovie}
				byMovie[key] = entry
			}
			for _, room := range movie.Rooms {
				for _, session := range room.Sessions {
					if strings.TrimSpace(session.Room) == "" {
						session.Room = room.Name
					}
					distance, hasDistance := theaterDistanceKM(result.theater, userLocation)
					entry.sessions = append(entry.sessions, sessionWithTheater{
						session:     session,
						theater:     result.theater,
						hasDistance: hasDistance,
						distanceKM:  distance,
					})
				}
			}
		}
	}

	movies := make([]movieAggregate, 0, len(byMovie))
	for _, movie := range byMovie {
		movies = append(movies, *movie)
	}
	sort.Slice(movies, func(i, j int) bool {
		return strings.ToLower(movies[i].movie.Title) < strings.ToLower(movies[j].movie.Title)
	})
	return movies, failed, ignored
}

func movieAggregateKey(movie model.TheaterMovie) string {
	if strings.TrimSpace(movie.Id) != "" {
		return "id:" + movie.Id
	}
	title := strings.ToLower(strings.TrimSpace(movie.Title))
	orig := strings.ToLower(strings.TrimSpace(movie.OriginalTitle))
	return title + "|" + orig
}

func isRecentTheater(theater model.Theater, cityID string, recents []store.RecentTheater) bool {
	for _, recent := range recents {
		if recent.CityID != "" && cityID != "" && recent.CityID != cityID {
			continue
		}
		if recent.TheaterID != "" && recent.TheaterID == theater.Id {
			return true
		}
		if recent.Name != "" && strings.EqualFold(recent.Name, theater.Name) {
			return true
		}
	}
	return false
}

func sortTheatersByDistance(theaters []model.Theater, location *service.UserLocation) {
	sort.Slice(theaters, func(i, j int) bool {
		leftDist, leftOK := theaterDistanceKM(theaters[i], location)
		rightDist, rightOK := theaterDistanceKM(theaters[j], location)
		if leftOK && rightOK && math.Abs(leftDist-rightDist) > 1e-6 {
			return leftDist < rightDist
		}
		if leftOK != rightOK {
			return leftOK
		}
		return strings.ToLower(theaters[i].Name) < strings.ToLower(theaters[j].Name)
	})
}

func theaterDistanceKM(theater model.Theater, location *service.UserLocation) (float64, bool) {
	if location == nil {
		return 0, false
	}
	if theater.Geolocation.Lat == 0 && theater.Geolocation.Lng == 0 {
		return 0, false
	}
	distance := haversineKM(location.Latitude, location.Longitude, theater.Geolocation.Lat, theater.Geolocation.Lng)
	return distance, true
}

func locationLabel(location *service.UserLocation) string {
	if location == nil {
		return ""
	}
	place := strings.TrimSpace(strings.Join([]string{location.City, location.Region}, ", "))
	source := locationSourceLabel(location.Source)
	if place != "" && source != "" {
		return fmt.Sprintf("%s (%s)", place, source)
	}
	if place != "" {
		return place
	}
	return source
}

func locationSourceLabel(source string) string {
	raw := strings.TrimSpace(source)
	if raw == "" {
		return ""
	}
	normalized := strings.ToLower(raw)
	switch normalized {
	case "system":
		return "via sistema"
	case "ipapi", "ipwhois", "ipinfo":
		return fmt.Sprintf("via IP (%s)", normalized)
	default:
		if strings.Contains(normalized, "ip") {
			return fmt.Sprintf("via IP (%s)", normalized)
		}
		return fmt.Sprintf("via %s", raw)
	}
}

func haversineKM(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	const earthRadius = 6371.0
	toRad := math.Pi / 180

	lat1Rad := lat1 * toRad
	lon1Rad := lon1 * toRad
	lat2Rad := lat2 * toRad
	lon2Rad := lon2 * toRad

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

type sectionItem struct {
	section model.SessionSection
}

func (s sectionItem) Title() string {
	return s.section.Name
}

func (s sectionItem) Description() string {
	if s.section.HighestPrice > 0 || s.section.LowestPrice > 0 {
		return fmt.Sprintf("R$ %.2f - R$ %.2f", s.section.LowestPrice, s.section.HighestPrice)
	}
	return ""
}

func (s sectionItem) FilterValue() string {
	return strings.ToLower(s.section.Name)
}

func buildSectionItems(sections []model.SessionSection) []list.Item {
	items := make([]list.Item, 0, len(sections))
	for _, section := range sections {
		items = append(items, sectionItem{section: section})
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].(sectionItem).section.Name) < strings.ToLower(items[j].(sectionItem).section.Name)
	})
	return items
}

func filterSeatSections(sections []model.SessionSection) []model.SessionSection {
	var filtered []model.SessionSection
	for _, section := range sections {
		if section.HasSeatSelection {
			filtered = append(filtered, section)
		}
	}
	return filtered
}

func selectDay(days []model.TheaterSessionDay, date time.Time) model.TheaterSessionDay {
	if len(days) == 0 {
		return model.TheaterSessionDay{}
	}
	target := date.Format(time.DateOnly)
	for _, day := range days {
		if day.Date == target {
			return day
		}
	}
	return days[0]
}

func (m appModel) renderSeatMap() string {
	if m.seatMap.Bounds.Lines == 0 || m.seatMap.Bounds.Columns == 0 {
		return "No seat map data."
	}

	rows := m.seatMap.Bounds.Lines
	cols := m.seatMap.Bounds.Columns

	grid := make([][]seatCell, rows)
	for i := range grid {
		grid[i] = make([]seatCell, cols)
		for j := range grid[i] {
			grid[i][j] = seatCell{}
		}
	}

	rowLabel := make(map[int]string)
	frontRows := frontLineSet(m.seatMap, 3)
	available := 0
	occupied := 0
	blocked := 0
	nonIdealAvailable := 0
	total := 0
	minRow, maxRow := rows-1, 0
	minCol, maxCol := cols-1, 0

	for _, line := range m.seatMap.Lines {
		for _, seat := range line.Seats {
			r := seat.Line - 1
			c := seat.Column - 1
			if r < 0 || c < 0 || r >= rows || c >= cols {
				continue
			}
			total++
			minRow = min(minRow, r)
			maxRow = max(maxRow, r)
			minCol = min(minCol, c)
			maxCol = max(maxCol, c)

			if _, ok := rowLabel[r]; !ok {
				rowLabel[r] = seatRowLabel(seat)
			}
			token, status := seatToken(seat)
			switch status {
			case "available":
				available++
			case "occupied":
				occupied++
			case "blocked":
				blocked++
			}
			cell := seatCell{
				token:  token,
				status: status,
				label:  seatNumberLabel(seat),
				front:  frontRows[seat.Line],
			}
			if cell.front && status == "available" {
				nonIdealAvailable++
			}
			grid[r][c] = cell
		}
	}

	if total == 0 {
		return "No seat map data."
	}

	rowWidth := 2
	for _, label := range rowLabel {
		if len(label) > rowWidth {
			rowWidth = len(label)
		}
	}

	maxLabelWidth := 2
	if m.showSeatNumbers {
		for r := minRow; r <= maxRow; r++ {
			for c := minCol; c <= maxCol; c++ {
				if l := len(grid[r][c].label); l > maxLabelWidth {
					maxLabelWidth = l
				}
			}
		}
	}
	cellWidth := max(2, maxLabelWidth)

	var b strings.Builder
	seatStyleAvailable := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	seatStyleOccupied := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	seatStyleBlocked := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	seatStyleAccessible := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	seatStyleFront := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)

	for r := minRow; r <= maxRow; r++ {
		label := rowLabel[r]
		if label == "" {
			label = fmt.Sprintf("%d", r+1)
		}
		b.WriteString(fmt.Sprintf("%*s ", rowWidth, label))
		for c := minCol; c <= maxCol; c++ {
			cell := grid[r][c]
			text := cell.token
			if m.showSeatNumbers && cell.label != "" {
				text = cell.label
			}
			rendered := padCell(text, cellWidth)
			switch cell.token {
			case "[]":
				if cell.front {
					rendered = seatStyleFront.Render(rendered)
				} else {
					rendered = seatStyleAvailable.Render(rendered)
				}
			case "XX":
				rendered = seatStyleOccupied.Render(rendered)
			case "##":
				rendered = seatStyleBlocked.Render(rendered)
			case "DD":
				rendered = seatStyleAccessible.Render(rendered)
			}
			b.WriteString(rendered)
			if c < maxCol {
				b.WriteString(" ")
			}
		}
		b.WriteString(fmt.Sprintf(" %*s\n", rowWidth, label))
	}

	gridWidth := (maxCol-minCol+1)*(cellWidth+1) - 1
	screenStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("214"))
	screenBorderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Background(lipgloss.Color("236"))

	screenBar := screenBarBlock(gridWidth, "SCREEN")

	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", rowWidth+1))
	b.WriteString(screenBorderStyle.Render(screenBar.top))
	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", rowWidth+1))
	b.WriteString(screenStyle.Render(screenBar.mid))
	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", rowWidth+1))
	b.WriteString(screenBorderStyle.Render(screenBar.bot))
	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", rowWidth+1))
	b.WriteString(hint("Front / Screen"))
	b.WriteString("\n\n")

	legend := "Legend: [] available • XX occupied • DD accessibility • ## blocked • front rows (not ideal)"
	if m.showSeatNumbers {
		legend = "Legend: color shows status • numbers are seat labels • front rows in yellow"
	}
	percent := float64(available) / float64(max(1, total)) * 100
	ideal := max(0, available-nonIdealAvailable)
	pairs := countAdjacentPairs(m.seatMap)
	counts := fmt.Sprintf("Available: %d • Ideal: %d • Front: %d • Pairs: %d • Occupied: %d • Blocked: %d • Total: %d • %.0f%% available", available, ideal, nonIdealAvailable, pairs, occupied, blocked, total, percent)
	return b.String() + hint(legend) + "\n" + hint(counts)
}

func seatToken(seat model.Seat) (string, string) {
	switch strings.ToLower(seat.Status) {
	case "occupied":
		return "XX", "occupied"
	case "available":
		if strings.EqualFold(seat.Type, "Disability") {
			return "DD", "available"
		}
		return "[]", "available"
	case "blocked", "unavailable":
		return "##", "blocked"
	default:
		return "  ", "unknown"
	}
}

type seatCell struct {
	token  string
	status string
	label  string
	front  bool
}

func seatRowLabel(seat model.Seat) string {
	parts := strings.Fields(strings.TrimSpace(seat.Label))
	if len(parts) > 0 {
		first := parts[0]
		if len(first) == 1 && first[0] >= 'A' && first[0] <= 'Z' {
			return first
		}
	}
	if seat.Line > 0 {
		return fmt.Sprintf("%d", seat.Line)
	}
	return ""
}

func seatNumberLabel(seat model.Seat) string {
	parts := strings.Fields(strings.TrimSpace(seat.Label))
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[len(parts)-1]
}

func frontLineSet(seatMap model.SeatMap, count int) map[int]bool {
	lines := map[int]bool{}
	for _, line := range seatMap.Lines {
		for _, seat := range line.Seats {
			if seat.Line > 0 {
				lines[seat.Line] = true
			}
		}
	}
	if len(lines) == 0 {
		return map[int]bool{}
	}
	var keys []int
	for k := range lines {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	if count > len(keys) {
		count = len(keys)
	}
	front := make(map[int]bool, count)
	for i := len(keys) - count; i < len(keys); i++ {
		front[keys[i]] = true
	}
	return front
}

func countAdjacentPairs(seatMap model.SeatMap) int {
	cols := map[int][]int{}
	for _, line := range seatMap.Lines {
		for _, seat := range line.Seats {
			if strings.ToLower(seat.Status) != "available" {
				continue
			}
			if seat.Column > 0 {
				cols[seat.Line] = append(cols[seat.Line], seat.Column)
			}
		}
	}
	return countAdjacentPairsFromCols(cols)
}

func countAdjacentPairsFromCols(cols map[int][]int) int {
	count := 0
	for _, list := range cols {
		if len(list) == 0 {
			continue
		}
		sort.Ints(list)
		for i := 0; i < len(list)-1; {
			if list[i]+1 == list[i+1] {
				count++
				i += 2
				continue
			}
			i++
		}
	}
	return count
}

func padCell(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if text == "" {
		return strings.Repeat(" ", width)
	}
	if len(text) >= width {
		return text[:width]
	}
	padding := width - len(text)
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

type seatCount struct {
	available         int
	occupied          int
	blocked           int
	total             int
	nonIdealAvailable int
	idealAvailable    int
	pairAvailable     int
	loaded            bool
	err               error
}

type screenBlock struct {
	top string
	mid string
	bot string
}

func screenBarBlock(width int, label string) screenBlock {
	if width < len(label)+4 {
		width = len(label) + 4
	}
	if width < 10 {
		width = 10
	}

	border := "╭" + strings.Repeat("─", width-2) + "╮"
	bottom := "╰" + strings.Repeat("─", width-2) + "╯"

	labelText := " " + label + " "
	padding := width - len(labelText) - 2
	left := padding / 2
	right := padding - left
	mid := "│" + strings.Repeat(" ", left) + labelText + strings.Repeat(" ", right) + "│"
	return screenBlock{top: border, mid: mid, bot: bottom}
}

func formatSessionTypes(types []string) string {
	if len(types) == 0 {
		return "Normal"
	}
	var cleaned []string
	for _, t := range types {
		trimmed := strings.TrimSpace(t)
		if strings.EqualFold(trimmed, "Normal") {
			continue
		}
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return "Normal"
	}
	return strings.Join(cleaned, ", ")
}

func formatPrice(price float64) string {
	if price <= 0 {
		return "-"
	}
	return fmt.Sprintf("R$ %.2f", price)
}

func halfPrice(price float64) float64 {
	if price <= 0 {
		return 0
	}
	return price / 2
}
