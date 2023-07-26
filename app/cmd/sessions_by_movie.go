package cmd

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/manifoldco/promptui"
	"golang.org/x/exp/maps"
	"ingresso-finder-cli/model"
	"ingresso-finder-cli/service"
	"ingresso-finder-cli/utils"
	"os"
	"strings"
	"time"
)

func GetSessionsByMovie() {
	cityName := service.PromptGetCity()
	city := service.GetCityInfoByName(cityName)
	events := service.GetEventsByCity(city.Id)

	rowConfigAutoMerge := table.RowConfig{AutoMerge: true}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Theater", "Time", "Type"}, rowConfigAutoMerge)
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: true, WidthMax: 20},
		{Number: 2, AutoMerge: true},
		{Number: 4, AutoMerge: true},
	})
	t.Style().Options.SeparateRows = true

	eventId := promptSelectEvent(events)

	sessions := service.GetSessionsByCityAndEvent(city.Id, eventId)
	sessionIndex := promptSelectEventDay(sessions)
	for _, theater := range sessions[sessionIndex].Theaters {
		var items []table.Row
		for _, sessionType := range theater.SessionTypes {
			for _, session := range sessionType.Sessions {
				items = append(items, table.Row{
					strings.ReplaceAll(theater.Name, city.Name, ""),
					session.Date.LocalDate.Format(time.TimeOnly),
					strings.Join(utils.Remove(session.Type, "Normal"), ", "),
				})
			}
		}
		t.AppendRows(items, rowConfigAutoMerge)
		t.AppendSeparator()
	}

	t.Render()
}

func promptSelectEvent(events []model.Event) (eventId string) {
	eventIdByName := make(map[string]string)
	for _, event := range events[0].Events {
		eventIdByName[event.Title] = event.Id
	}

	searcher := func(input string, index int) bool {
		return true
	}

	selectEvent := promptui.Select{
		Label:    "Select Event",
		Items:    maps.Keys(eventIdByName),
		Size:     10,
		Searcher: searcher,
	}

	_, eventName, err := selectEvent.Run()
	eventId, ok := eventIdByName[eventName]
	if !ok || err != nil {
		panic("invalid event")
	}
	return
}

func promptSelectEventDay(sessions []model.Session) (indexSession int) {
	sessionIndexByDate := make(map[string]int)
	for i, session := range sessions {
		sessionIndexByDate[session.DateFormatted] = i
	}

	selectEventDay := promptui.Select{
		Label: "Select Date",
		Items: maps.Keys(sessionIndexByDate),
		Size:  10,
	}
	_, selectedDay, err := selectEventDay.Run()
	indexSession, ok := sessionIndexByDate[selectedDay]
	if !ok || err != nil {
		panic("invalid date")
	}
	return
}
