package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"ingresso-finder-cli/model"
	"io"
	"net/http"
	"time"
)

func GetSessionsByCityAndEvent(cityId string, eventId string, date ...time.Time) []model.Session {
	var url string
	if len(date) > 0 {
		url = fmt.Sprintf(
			"https://api-content.ingresso.com/v0/sessions/city/%s/event/%s/partnership/home/groupBy/sessionType?date=%s",
			cityId,
			eventId,
			date[0].Format(time.DateOnly),
		)
	} else {
		url = fmt.Sprintf(
			"https://api-content.ingresso.com/v0/sessions/city/%s/event/%s/partnership/home/groupBy/sessionType",
			cityId,
			eventId,
		)
	}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5.2 Safari/605.1.15")
	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var session []model.Session
	json.Unmarshal(body, &session)
	return session
}

func GetEventsByCity(cityId string) []model.Event {
	url := fmt.Sprintf("https://api-content.ingresso.com/v0/carousel/%s/partnership/home?carousels=em-cartaz", cityId)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5.2 Safari/605.1.15")
	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var event []model.Event

	json.Unmarshal(body, &event)
	return event
}

func GetCityInfoByName(cityName string) model.City {
	url := "https://api-content.ingresso.com/v0/states/city/name/" + cityName
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5.2 Safari/605.1.15")
	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var city model.City
	json.Unmarshal(body, &city)
	return city
}

func PromptGetCity() string {
	validate := func(input string) error {
		if input == "" {
			return errors.New("invalid number")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Your City",
		Validate: validate,
	}

	cityName, err := prompt.Run()
	if err != nil {
		panic("Invalid City")
	}
	return cityName
}
