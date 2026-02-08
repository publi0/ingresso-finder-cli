package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetJSON_Non2xxReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.baseURL = server.URL
	client.maxAttempts = 1

	var out map[string]any
	err := client.getJSON(context.Background(), server.URL+"/fail", &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetJSON_RetriesTransientServerErrors(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		if current < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("retry later"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.baseURL = server.URL
	client.maxAttempts = 3
	client.retryBase = time.Millisecond
	client.retryCap = 2 * time.Millisecond

	var out map[string]any
	if err := client.getJSON(context.Background(), server.URL+"/retry", &out); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if ok, _ := out["ok"].(bool); !ok {
		t.Fatalf("unexpected payload: %+v", out)
	}
}

func TestGetJSON_DoesNotRetryOnClientErrors(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.baseURL = server.URL
	client.maxAttempts = 3
	client.retryBase = time.Millisecond
	client.retryCap = 2 * time.Millisecond

	var out map[string]any
	err := client.getJSON(context.Background(), server.URL+"/bad-request", &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestGetCityInfoByName_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/states/city/name/") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1","name":"Sao Paulo"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.baseURL = server.URL

	city, err := client.GetCityInfoByName(context.Background(), "sao paulo")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if city.Id != "1" {
		t.Fatalf("unexpected city id: %s", city.Id)
	}
}

func TestGetCities_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/states" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
  {
    "name": "State A",
    "uf": "AA",
    "cities": [
      {"id": "1", "name": "City One", "uf": "AA", "state": "State A", "urlKey": "city-one", "timeZone": "America/Sao_Paulo"}
    ]
  },
  {
    "name": "State B",
    "uf": "BB",
    "cities": [
      {"id": "2", "name": "City Two", "uf": "BB", "state": "State B", "urlKey": "city-two", "timeZone": "America/Sao_Paulo"}
    ]
  }
]`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.baseURL = server.URL

	cities, err := client.GetCities(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(cities) != 2 {
		t.Fatalf("expected 2 cities, got %d", len(cities))
	}
}

func TestGetTheatersByCity_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/theaters/city/1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
  {"id": "10", "name": "Theater One", "address": "Street", "neighborhood": "Center", "uf": "SP"}
]`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.baseURL = server.URL

	theaters, err := client.GetTheatersByCity(context.Background(), "1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(theaters) != 1 {
		t.Fatalf("expected 1 theater, got %d", len(theaters))
	}
}

func TestGetSessionsByCityAndTheater_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/city/1/theater/10" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.RawQuery != "date=2026-02-03" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
  {
    "date": "2026-02-03",
    "dateFormatted": "03/02",
    "dayOfWeek": "terça-feira",
    "isToday": true,
    "movies": [
      {
        "id": "99",
        "title": "Movie One",
        "rooms": [
          {
            "name": "Room 1",
            "sessions": [
              {"id": "1", "price": 40.0, "room": "Room 1", "type": ["Dublado"], "date": {"localDate": "2026-02-03T19:30:00-03:00"}}
            ]
          }
        ]
      }
    ]
  }
]`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.baseURL = server.URL

	date := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	days, err := client.GetSessionsByCityAndTheater(context.Background(), "1", "10", &date)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(days) != 1 {
		t.Fatalf("expected 1 day, got %d", len(days))
	}
	if len(days[0].Movies) != 1 {
		t.Fatalf("expected 1 movie, got %d", len(days[0].Movies))
	}
}

func TestGetSessionDetails_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "id": "123",
  "sections": [
    {"id": "456", "name": "Sala 1", "hasSeatSelection": true, "highestPrice": 40.0, "lowestPrice": 20.0}
  ]
}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.checkoutURL = server.URL

	detail, err := client.GetSessionDetails(context.Background(), "123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if detail.Id != "123" {
		t.Fatalf("unexpected session id: %s", detail.Id)
	}
	if len(detail.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(detail.Sections))
	}
}

func TestGetSeatMap_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/123/sections/456/seats" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "id": "456",
  "bounds": {"lines": 1, "columns": 1},
  "lines": [{"line": 1, "seats": [{"id": "1", "rowIndex": 0, "columnIndex": 0, "status": "Available", "type": "Normal"}]}]
}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.checkoutURL = server.URL

	seats, err := client.GetSeatMap(context.Background(), "123", "456")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if seats.Id != "456" {
		t.Fatalf("unexpected seat map id: %s", seats.Id)
	}
	if seats.Bounds.Lines != 1 || seats.Bounds.Columns != 1 {
		t.Fatalf("unexpected bounds: %+v", seats.Bounds)
	}
}
