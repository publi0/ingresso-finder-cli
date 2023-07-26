package cmd

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"ingresso-finder-cli/service"
	"ingresso-finder-cli/utils"
	"os"
	"strings"
	"time"
)

func GetEventsByCity() {
	cityName := service.PromptGetCity()
	if cityName == "" {
		panic("Invalid City")
	}

	city := service.GetCityInfoByName(cityName)
	events := service.GetEventsByCity(city.Id)

	rowConfigAutoMerge := table.RowConfig{AutoMerge: true}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Movie", "Theater", "Time", "Type"}, rowConfigAutoMerge)
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: true, WidthMax: 20},
		{Number: 2, AutoMerge: true},
		{Number: 4, AutoMerge: true},
	})
	t.Style().Options.SeparateRows = true

	for _, event := range events[0].Events {
		var items []table.Row
		sessions := service.GetSessionsByCityAndEvent(city.Id, event.Id, time.Now())
		for _, session := range sessions {
			for _, theater := range session.Theaters {
				for _, sessionType := range theater.SessionTypes {
					for _, session := range sessionType.Sessions {
						items = append(items, table.Row{
							event.Title,
							strings.ReplaceAll(theater.Name, city.Name, ""),
							session.Date.LocalDate.Format(time.TimeOnly),
							strings.Join(utils.Remove(session.Type, "Normal"), ", "),
						})
					}
				}
			}
		}
		t.AppendRows(items, rowConfigAutoMerge)
		t.AppendSeparator()
	}
	t.Render()
}
