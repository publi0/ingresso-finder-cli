package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"ingresso-finder-cli/model"
	"ingresso-finder-cli/service"
	"ingresso-finder-cli/store"
)

type testItem struct {
	value string
}

func (t testItem) Title() string       { return t.value }
func (t testItem) Description() string { return "" }
func (t testItem) FilterValue() string { return strings.ToLower(t.value) }

func newFilterModel(items []list.Item) *appModel {
	model := New().(appModel)
	model.state = stateSelectCity
	model.cityList = newList("Select City")
	model.cityList.SetItems(items)
	return &model
}

func setStoreIsolationEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("XDG_CACHE_HOME", root)
}

func TestHandleFilterInput_AppendsRunes(t *testing.T) {
	m := newFilterModel([]list.Item{
		testItem{value: "Barueri"},
		testItem{value: "Sao Paulo"},
	})

	if !m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}) {
		t.Fatal("expected filter input to be handled")
	}
	if got := m.cityList.FilterValue(); got != "b" {
		t.Fatalf("expected filter value to be %q, got %q", "b", got)
	}

	if !m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}) {
		t.Fatal("expected filter input to be handled")
	}
	if got := m.cityList.FilterValue(); got != "ba" {
		t.Fatalf("expected filter value to be %q, got %q", "ba", got)
	}
}

func TestHandleFilterInput_Backspace(t *testing.T) {
	m := newFilterModel([]list.Item{
		testItem{value: "Barueri"},
		testItem{value: "Sao Paulo"},
	})

	_ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	_ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if got := m.cityList.FilterValue(); got != "ba" {
		t.Fatalf("expected filter value to be %q, got %q", "ba", got)
	}

	if !m.handleFilterInput(tea.KeyMsg{Type: tea.KeyBackspace}) {
		t.Fatal("expected backspace to be handled")
	}
	if got := m.cityList.FilterValue(); got != "b" {
		t.Fatalf("expected filter value to be %q, got %q", "b", got)
	}
}

func TestHandleFilterInput_Space(t *testing.T) {
	m := newFilterModel([]list.Item{
		testItem{value: "Rio de Janeiro"},
	})

	_ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	_ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	_ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})

	if !m.handleFilterInput(tea.KeyMsg{Type: tea.KeySpace}) {
		t.Fatal("expected space to be handled")
	}

	if got := m.cityList.FilterValue(); got != "rio " {
		t.Fatalf("expected filter value to be %q, got %q", "rio ", got)
	}
}

func TestBuildTheaterItems_HiddenTheatersAreExcluded(t *testing.T) {
	setStoreIsolationEnv(t)

	theaters := []model.Theater{
		{Id: "1", Name: "Cinema A"},
		{Id: "2", Name: "Cinema B"},
	}
	items := buildTheaterItems(theaters, "city-1", map[string]bool{"2": true}, nil)
	if len(items) != 1 {
		t.Fatalf("expected 1 visible theater, got %d", len(items))
	}

	theater, ok := items[0].(theaterItem)
	if !ok {
		t.Fatalf("expected theater item, got %#v", items[0])
	}
	if theater.theater.Id != "1" {
		t.Fatalf("expected visible theater id %q, got %q", "1", theater.theater.Id)
	}
}

func TestBuildTheaterItems_SortsByDistanceWhenLocationExists(t *testing.T) {
	setStoreIsolationEnv(t)

	near := model.Theater{Id: "near", Name: "Near Cinema"}
	near.Geolocation.Lat = -23.5505
	near.Geolocation.Lng = -46.6333

	far := model.Theater{Id: "far", Name: "Far Cinema"}
	far.Geolocation.Lat = -22.9068
	far.Geolocation.Lng = -43.1729

	location := &service.UserLocation{Latitude: -23.55, Longitude: -46.63}
	items := buildTheaterItems([]model.Theater{far, near}, "city-1", map[string]bool{}, location)
	if len(items) != 2 {
		t.Fatalf("expected 2 theaters, got %d", len(items))
	}

	firstTheater := items[0].(theaterItem)
	secondTheater := items[1].(theaterItem)
	if firstTheater.theater.Id != "near" || secondTheater.theater.Id != "far" {
		t.Fatalf("expected near theater first, got %q then %q", firstTheater.theater.Id, secondTheater.theater.Id)
	}
}

func TestLocationLabel_IncludesSystemSource(t *testing.T) {
	label := locationLabel(&service.UserLocation{
		City:   "Sao Paulo",
		Region: "SP",
		Source: "system",
	})
	if label != "Sao Paulo, SP (via sistema)" {
		t.Fatalf("unexpected label: %q", label)
	}
}

func TestLocationLabel_IncludesIPSource(t *testing.T) {
	label := locationLabel(&service.UserLocation{
		City:   "Sao Paulo",
		Region: "SP",
		Source: "ipapi",
	})
	if label != "Sao Paulo, SP (via IP (ipapi))" {
		t.Fatalf("unexpected label: %q", label)
	}
}

func TestHandleFilterInput_TheaterScreenAcceptsNumericFilter(t *testing.T) {
	setStoreIsolationEnv(t)

	app := New().(appModel)
	app.state = stateSelectTheater
	app.theaterList = newList("Theaters")
	app.theaterList.SetItems([]list.Item{
		theaterItem{theater: model.Theater{Id: "1", Name: "Cinema A"}},
	})

	handled := app.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if !handled {
		t.Fatal("expected key '1' to be handled as filter text")
	}
	if got := app.theaterList.FilterValue(); got != "1" {
		t.Fatalf("expected filter value 1, got %q", got)
	}
}

func TestErrorFromLoadingSessions_RecoversToSelectTheater(t *testing.T) {
	app := New().(appModel)
	app.state = stateLoadingSessions

	updated, _ := app.Update(errMsg{err: errTest("boom")})
	next := updated.(appModel)

	if next.state != stateError {
		t.Fatalf("expected stateError, got %v", next.state)
	}
	if next.lastState != stateSelectTheater {
		t.Fatalf("expected recover state stateSelectTheater, got %v", next.lastState)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next = updated.(appModel)
	if next.state != stateSelectTheater {
		t.Fatalf("expected esc to return to stateSelectTheater, got %v", next.state)
	}
}

func TestErrorEnter_AdvancesToNextDay(t *testing.T) {
	app := New().(appModel)
	app.state = stateError
	app.errorSuggestNextDay = true
	app.browsingAllTheaters = true
	app.city = model.City{Id: "1", Name: "Sao Paulo"}
	app.theaters = []model.Theater{{Id: "10", Name: "Cinema A"}}
	app.hiddenTheaters = map[string]bool{}
	app.date = time.Date(2026, 2, 6, 10, 30, 0, 0, time.UTC)

	updated, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(appModel)

	if next.state != stateLoadingSessions {
		t.Fatalf("expected stateLoadingSessions, got %v", next.state)
	}
	if next.date.Format(time.DateOnly) != "2026-02-07" {
		t.Fatalf("expected date to advance to 2026-02-07, got %s", next.date.Format(time.DateOnly))
	}
	if cmd == nil {
		t.Fatal("expected reload command when advancing date")
	}
}

func TestErrorCtrlD_OpensDatePicker(t *testing.T) {
	app := New().(appModel)
	app.state = stateError
	app.errorSuggestNextDay = true
	app.date = time.Date(2026, 2, 6, 0, 0, 0, 0, time.UTC)

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	next := updated.(appModel)

	if next.state != stateSelectDate {
		t.Fatalf("expected stateSelectDate, got %v", next.state)
	}
	if !next.dateReturnStateSet {
		t.Fatal("expected dateReturnStateSet to be true")
	}
	if next.dateReturnState != stateShowSessions {
		t.Fatalf("expected dateReturnState=stateShowSessions, got %v", next.dateReturnState)
	}
}

func TestOpenMovieAcrossTheaters_EnablesGlobalMode(t *testing.T) {
	app := New().(appModel)
	app.city = model.City{Id: "1"}
	app.theaters = []model.Theater{{Id: "10", Name: "Cinema A"}}
	app.hiddenTheaters = map[string]bool{}

	updated, cmd, handled := app.openMovieAcrossTheaters()
	next := updated.(appModel)

	if !handled {
		t.Fatal("expected handled=true")
	}
	if !next.browsingAllTheaters {
		t.Fatal("expected browsingAllTheaters=true")
	}
	if next.state != stateLoadingSessions {
		t.Fatalf("expected stateLoadingSessions, got %v", next.state)
	}
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
}

func TestEnterTheaterItem_StartsTheaterFlow(t *testing.T) {
	app := New().(appModel)
	app.state = stateSelectTheater
	app.theaterList = newList("Theaters")
	app.theaterList.SetItems([]list.Item{
		theaterItem{theater: model.Theater{Id: "10", Name: "Cinema A"}},
	})

	updated, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(appModel)

	if next.state != stateLoadingSessions {
		t.Fatalf("expected stateLoadingSessions, got %v", next.state)
	}
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
}

func TestTheaterScreen_AcceptsRuneInputWithoutChangingState(t *testing.T) {
	app := New().(appModel)
	app.state = stateSelectTheater
	app.theaterList = newList("Theaters")
	app.theaterList.SetItems([]list.Item{
		theaterItem{theater: model.Theater{Id: "10", Name: "Cinema A"}},
	})

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	next := updated.(appModel)
	if next.state != stateSelectTheater {
		t.Fatalf("expected stateSelectTheater, got %v", next.state)
	}
}

func TestStartupRecentCity_UsesMostRecent(t *testing.T) {
	setStoreIsolationEnv(t)

	if err := store.RememberCity(model.City{Id: "3558", Name: "Sao Paulo", Uf: "SP"}); err != nil {
		t.Fatalf("remember city: %v", err)
	}

	recent, ok := startupRecentCity()
	if !ok {
		t.Fatal("expected a recent city")
	}
	if recent.ID != "3558" {
		t.Fatalf("expected recent id 3558, got %s", recent.ID)
	}
}

func TestCityFromRecentCache_ByID(t *testing.T) {
	setStoreIsolationEnv(t)

	cities := []model.City{
		{Id: "3558", Name: "Sao Paulo", Uf: "SP"},
		{Id: "6000", Name: "Campinas", Uf: "SP"},
	}
	if err := store.SaveCityCache(cities); err != nil {
		t.Fatalf("save city cache: %v", err)
	}

	city, ok := cityFromRecentCache(store.RecentCity{ID: "6000"})
	if !ok {
		t.Fatal("expected cache hit")
	}
	if city.Name != "Campinas" {
		t.Fatalf("expected Campinas, got %s", city.Name)
	}
}

func TestPendingSeatCountSessionIDsOnCurrentPage(t *testing.T) {
	app := New().(appModel)
	app.state = stateShowSessions
	app.seatCounts = map[string]seatCount{
		"s2": {loaded: true},
	}
	app.sessionList = newList("Sessions")
	app.sessionList.SetItems([]list.Item{
		sessionItem{session: model.TheaterSession{Id: "s1", HasSeatSelection: true}},
		sessionItem{session: model.TheaterSession{Id: "s2", HasSeatSelection: true}},
		sessionItem{session: model.TheaterSession{Id: "s3", HasSeatSelection: false}},
		sessionItem{session: model.TheaterSession{Id: "s4", HasSeatSelection: true}},
	})
	app.sessionList.Paginator.PerPage = 2
	app.sessionList.Paginator.Page = 1

	ids := app.pendingSeatCountSessionIDsOnCurrentPage()
	if len(ids) != 1 {
		t.Fatalf("expected 1 pending id, got %d (%v)", len(ids), ids)
	}
	if ids[0] != "s4" {
		t.Fatalf("expected pending id s4, got %s", ids[0])
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestAggregateMovieCatalog_MergesSameMovieAcrossTheaters(t *testing.T) {
	date := time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)
	sessionA := model.TheaterSession{Id: "s1"}
	sessionA.Date.LocalDate = date.Add(19 * time.Hour)
	sessionB := model.TheaterSession{Id: "s2"}
	sessionB.Date.LocalDate = date.Add(21 * time.Hour)

	results := make(chan theaterSessionsResult, 2)
	results <- theaterSessionsResult{
		theater: model.Theater{Id: "t1", Name: "Cinema 1"},
		days: []model.TheaterSessionDay{{
			Date: date.Format(time.DateOnly),
			Movies: []model.TheaterMovie{{
				Id:    "m1",
				Title: "Movie X",
				Rooms: []model.TheaterRoom{{Name: "Room 1", Sessions: []model.TheaterSession{sessionA}}},
			}},
		}},
	}
	results <- theaterSessionsResult{
		theater: model.Theater{Id: "t2", Name: "Cinema 2"},
		days: []model.TheaterSessionDay{{
			Date: date.Format(time.DateOnly),
			Movies: []model.TheaterMovie{{
				Id:    "m1",
				Title: "Movie X",
				Rooms: []model.TheaterRoom{{Name: "Room 2", Sessions: []model.TheaterSession{sessionB}}},
			}},
		}},
	}
	close(results)

	movies, failed, ignored := aggregateMovieCatalog(results, date, nil)
	if failed != 0 || ignored != 0 {
		t.Fatalf("expected no failures/ignored, got failed=%d ignored=%d", failed, ignored)
	}
	if len(movies) != 1 {
		t.Fatalf("expected 1 merged movie, got %d", len(movies))
	}
	if len(movies[0].sessions) != 2 {
		t.Fatalf("expected 2 sessions in merged movie, got %d", len(movies[0].sessions))
	}
}
