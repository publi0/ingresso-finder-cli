package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ingresso-finder-cli/model"
)

const (
	cityCacheTTL     = 7 * 24 * time.Hour
	theaterCacheTTL  = 72 * time.Hour
	sessionCacheTTL  = 10 * time.Minute
	maxRecentCities  = 8
	maxRecentTheater = 8
)

type cacheEnvelope[T any] struct {
	UpdatedAt time.Time `json:"updated_at"`
	Data      T         `json:"data"`
}

type RecentCity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	UF   string `json:"uf"`
}

type cityHistory struct {
	Cities []RecentCity `json:"cities"`
}

type RecentTheater struct {
	CityID    string `json:"city_id"`
	TheaterID string `json:"theater_id"`
	Name      string `json:"name"`
}

type theaterHistory struct {
	Theaters []RecentTheater `json:"theaters"`
}

type theaterVisibility struct {
	HiddenByCity map[string][]string `json:"hidden_by_city"`
}

func LoadCityCache() ([]model.City, bool, error) {
	path, err := cachePath("cities.json")
	if err != nil {
		return nil, false, err
	}
	cache, err := loadCache[[]model.City](path)
	if err != nil {
		return nil, false, err
	}
	return cache.Data, time.Since(cache.UpdatedAt) <= cityCacheTTL, nil
}

func SaveCityCache(cities []model.City) error {
	path, err := cachePath("cities.json")
	if err != nil {
		return err
	}
	return saveCache(path, cities)
}

func LoadTheaterCache(cityID string) ([]model.Theater, bool, error) {
	path, err := cachePath(fmt.Sprintf("theaters_%s.json", cityID))
	if err != nil {
		return nil, false, err
	}
	cache, err := loadCache[[]model.Theater](path)
	if err != nil {
		return nil, false, err
	}
	return cache.Data, time.Since(cache.UpdatedAt) <= theaterCacheTTL, nil
}

func SaveTheaterCache(cityID string, theaters []model.Theater) error {
	path, err := cachePath(fmt.Sprintf("theaters_%s.json", cityID))
	if err != nil {
		return err
	}
	return saveCache(path, theaters)
}

func LoadSessionCache(cityID string, theaterID string, date string) ([]model.TheaterSessionDay, bool, error) {
	path, err := cachePath(fmt.Sprintf("sessions_%s_%s_%s.json", cityID, theaterID, date))
	if err != nil {
		return nil, false, err
	}
	cache, err := loadCache[[]model.TheaterSessionDay](path)
	if err != nil {
		return nil, false, err
	}
	return cache.Data, time.Since(cache.UpdatedAt) <= sessionCacheTTL, nil
}

func SaveSessionCache(cityID string, theaterID string, date string, days []model.TheaterSessionDay) error {
	path, err := cachePath(fmt.Sprintf("sessions_%s_%s_%s.json", cityID, theaterID, date))
	if err != nil {
		return err
	}
	return saveCache(path, days)
}

func LoadRecentCities() ([]RecentCity, error) {
	path, err := configPath("history.json")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var history cityHistory
	if err := json.Unmarshal(data, &history); err == nil {
		return history.Cities, nil
	}

	var legacy []string
	if err := json.Unmarshal(data, &legacy); err == nil {
		var cities []RecentCity
		for _, name := range legacy {
			if name != "" {
				cities = append(cities, RecentCity{Name: name})
			}
		}
		return cities, nil
	}

	return nil, errors.New("invalid city history format")
}

func RememberCity(city model.City) error {
	history, _ := LoadRecentCities()
	next := []RecentCity{{ID: city.Id, Name: city.Name, UF: city.Uf}}

	for _, existing := range history {
		if existing.ID == city.Id && existing.ID != "" {
			continue
		}
		if existing.Name != "" && stringsEqualFold(existing.Name, city.Name) && stringsEqualFold(existing.UF, city.Uf) {
			continue
		}
		next = append(next, existing)
		if len(next) >= maxRecentCities {
			break
		}
	}

	return saveRecentCities(next)
}

func LoadRecentTheaters() ([]RecentTheater, error) {
	path, err := configPath("theaters.json")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var history theaterHistory
	if err := json.Unmarshal(data, &history); err == nil {
		return history.Theaters, nil
	}
	return nil, errors.New("invalid theater history format")
}

func RememberTheater(cityID string, theater model.Theater) error {
	history, _ := LoadRecentTheaters()
	next := []RecentTheater{{
		CityID:    cityID,
		TheaterID: theater.Id,
		Name:      theater.Name,
	}}

	for _, existing := range history {
		if existing.CityID == cityID && existing.TheaterID == theater.Id && existing.TheaterID != "" {
			continue
		}
		if existing.Name != "" && stringsEqualFold(existing.Name, theater.Name) && existing.CityID == cityID {
			continue
		}
		next = append(next, existing)
		if len(next) >= maxRecentTheater {
			break
		}
	}

	return saveRecentTheaters(next)
}

func LoadHiddenTheaters(cityID string) (map[string]bool, error) {
	result := map[string]bool{}
	if strings.TrimSpace(cityID) == "" {
		return result, nil
	}

	visibility, err := loadTheaterVisibility()
	if err != nil {
		return nil, err
	}
	for _, theaterID := range visibility.HiddenByCity[cityID] {
		if theaterID != "" {
			result[theaterID] = true
		}
	}
	return result, nil
}

func SetTheaterHidden(cityID string, theaterID string, hidden bool) error {
	cityID = strings.TrimSpace(cityID)
	theaterID = strings.TrimSpace(theaterID)
	if cityID == "" || theaterID == "" {
		return errors.New("city id and theater id are required")
	}

	visibility, err := loadTheaterVisibility()
	if err != nil {
		return err
	}
	if visibility.HiddenByCity == nil {
		visibility.HiddenByCity = map[string][]string{}
	}

	current := visibility.HiddenByCity[cityID]
	index := -1
	for i, id := range current {
		if id == theaterID {
			index = i
			break
		}
	}

	if hidden {
		if index < 0 {
			current = append(current, theaterID)
		}
	} else if index >= 0 {
		current = append(current[:index], current[index+1:]...)
	}

	if len(current) == 0 {
		delete(visibility.HiddenByCity, cityID)
	} else {
		sort.Strings(current)
		visibility.HiddenByCity[cityID] = current
	}
	return saveTheaterVisibility(visibility)
}

func loadCache[T any](path string) (cacheEnvelope[T], error) {
	var cache cacheEnvelope[T]
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil
		}
		return cache, err
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, err
	}
	return cache, nil
}

func saveCache[T any](path string, data T) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	cache := cacheEnvelope[T]{
		UpdatedAt: time.Now(),
		Data:      data,
	}
	payload, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func saveRecentCities(cities []RecentCity) error {
	path, err := configPath("history.json")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	history := cityHistory{Cities: cities}
	payload, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func saveRecentTheaters(theaters []RecentTheater) error {
	path, err := configPath("theaters.json")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	history := theaterHistory{Theaters: theaters}
	payload, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func loadTheaterVisibility() (theaterVisibility, error) {
	path, err := configPath("theater_visibility.json")
	if err != nil {
		return theaterVisibility{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return theaterVisibility{HiddenByCity: map[string][]string{}}, nil
		}
		return theaterVisibility{}, err
	}

	var visibility theaterVisibility
	if err := json.Unmarshal(data, &visibility); err != nil {
		return theaterVisibility{}, errors.New("invalid theater visibility format")
	}
	if visibility.HiddenByCity == nil {
		visibility.HiddenByCity = map[string][]string{}
	}
	return visibility, nil
}

func saveTheaterVisibility(visibility theaterVisibility) error {
	path, err := configPath("theater_visibility.json")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(visibility, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func configPath(name string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ingresso-finder-cli", name), nil
}

func cachePath(name string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ingresso-finder-cli", name), nil
}

func stringsEqualFold(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(a, b)
}
