package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDetectCurrentLocation_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"latitude":-23.55,"longitude":-46.63,"city":"Sao Paulo","region":"Sao Paulo","country_name":"Brazil"}`))
	}))
	defer server.Close()

	loc, err := detectCurrentLocationFromEndpoint(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if loc.City != "Sao Paulo" {
		t.Fatalf("unexpected city: %s", loc.City)
	}
	if loc.Latitude == 0 || loc.Longitude == 0 {
		t.Fatalf("unexpected coordinates: %+v", loc)
	}
	if loc.Source != "custom" {
		t.Fatalf("expected source custom, got %q", loc.Source)
	}
}

func TestDetectCurrentLocation_ProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	if _, err := detectCurrentLocationFromEndpoint(context.Background(), server.Client(), server.URL); err == nil {
		t.Fatal("expected error")
	}
}

func TestDetectCurrentLocationWithProviders_Fallback(t *testing.T) {
	blocked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>blocked</body></html>`))
	}))
	defer blocked.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"latitude":-23.55,"longitude":-46.63,"city":"Sao Paulo","region":"SP","country":"Brazil"}`))
	}))
	defer fallback.Close()

	loc, err := detectCurrentLocationWithProviders(context.Background(), blocked.Client(), []locationProvider{
		{name: "blocked", endpoint: blocked.URL, parse: parseIPAPI},
		{name: "fallback", endpoint: fallback.URL, parse: parseIPWhoIs},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if loc.City != "Sao Paulo" {
		t.Fatalf("unexpected city: %s", loc.City)
	}
	if loc.Source != "fallback" {
		t.Fatalf("expected source fallback, got %q", loc.Source)
	}
}

func TestDetectCurrentLocationFromProvider_HtmlErrorIsCompact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>blocked</body></html>`))
	}))
	defer server.Close()

	_, err := detectCurrentLocationFromProvider(context.Background(), server.Client(), locationProvider{
		name:     "blocked",
		endpoint: server.URL,
		parse:    parseIPAPI,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(strings.ToLower(err.Error()), "<html") {
		t.Fatalf("expected compact error without html, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected status code in error, got %q", err.Error())
	}
}

func TestDetectCurrentLocation_PrefersSystemProvider(t *testing.T) {
	previousDetect := detectSystemLocationFn
	detectSystemLocationFn = func(context.Context) (UserLocation, error) {
		return UserLocation{
			Latitude:  40.7128,
			Longitude: -74.0060,
			City:      "New York",
			Region:    "NY",
			Country:   "United States",
		}, nil
	}
	defer func() {
		detectSystemLocationFn = previousDetect
	}()

	previousProviders := defaultLocationProviders
	defaultLocationProviders = nil
	defer func() {
		defaultLocationProviders = previousProviders
	}()

	loc, err := DetectCurrentLocation(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if loc.City != "New York" {
		t.Fatalf("unexpected city: %s", loc.City)
	}
	if loc.Source != "system" {
		t.Fatalf("expected source system, got %q", loc.Source)
	}
}

func TestDetectCurrentLocation_FallsBackToIPWhenSystemFails(t *testing.T) {
	previousDetect := detectSystemLocationFn
	detectSystemLocationFn = func(context.Context) (UserLocation, error) {
		return UserLocation{}, errors.New("system unavailable")
	}
	defer func() {
		detectSystemLocationFn = previousDetect
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"latitude":-23.55,"longitude":-46.63,"city":"Sao Paulo","region":"Sao Paulo","country_name":"Brazil"}`))
	}))
	defer server.Close()

	previousProviders := defaultLocationProviders
	defaultLocationProviders = []locationProvider{
		{name: "ipapi", endpoint: server.URL, parse: parseIPAPI},
	}
	defer func() {
		defaultLocationProviders = previousProviders
	}()

	loc, err := DetectCurrentLocation(context.Background(), server.Client())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if loc.City != "Sao Paulo" {
		t.Fatalf("unexpected city: %s", loc.City)
	}
	if loc.Source != "ipapi" {
		t.Fatalf("expected source ipapi, got %q", loc.Source)
	}
}

func TestDetectCurrentLocation_CanceledContextStopsEarly(t *testing.T) {
	previousDetect := detectSystemLocationFn
	detectSystemLocationFn = func(context.Context) (UserLocation, error) {
		return UserLocation{}, context.Canceled
	}
	defer func() {
		detectSystemLocationFn = previousDetect
	}()

	_, err := DetectCurrentLocation(context.Background(), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}
