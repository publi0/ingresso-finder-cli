package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ipLocationEndpoint    = "https://ipapi.co/json/"
	ipWhoIsEndpoint       = "https://ipwho.is/"
	ipInfoEndpoint        = "https://ipinfo.io/json"
	locationErrorSnippetN = 120
)

// UserLocation is the detected current location.
type UserLocation struct {
	Latitude  float64
	Longitude float64
	City      string
	Region    string
	Country   string
	Source    string
}

type locationProvider struct {
	name     string
	endpoint string
	parse    func([]byte) (UserLocation, error)
}

var defaultLocationProviders = []locationProvider{
	{name: "ipapi", endpoint: ipLocationEndpoint, parse: parseIPAPI},
	{name: "ipwhois", endpoint: ipWhoIsEndpoint, parse: parseIPWhoIs},
	{name: "ipinfo", endpoint: ipInfoEndpoint, parse: parseIPInfo},
}

var detectSystemLocationFn = detectCurrentLocationFromSystem

// DetectCurrentLocation resolves the current user location, prioritizing system APIs and falling back to IP geolocation.
func DetectCurrentLocation(ctx context.Context, httpClient *http.Client) (UserLocation, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}

	systemLocation, err := detectSystemLocationFn(ctx)
	if err == nil {
		if strings.TrimSpace(systemLocation.Source) == "" {
			systemLocation.Source = "system"
		}
		return systemLocation, nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return UserLocation{}, err
	}

	location, ipErr := detectCurrentLocationWithProviders(ctx, httpClient, defaultLocationProviders)
	if ipErr == nil {
		logSystemFallback(err, location.Source)
		return location, nil
	}
	if errors.Is(ipErr, context.Canceled) || errors.Is(ipErr, context.DeadlineExceeded) {
		return UserLocation{}, ipErr
	}

	return UserLocation{}, fmt.Errorf("system location failed (%s); ip fallback failed (%s)", err.Error(), ipErr.Error())
}

func logSystemFallback(systemErr error, fallbackSource string) {
	if systemErr == nil {
		return
	}
	if strings.TrimSpace(os.Getenv("INGRESSO_LOCATION_DEBUG")) == "" {
		return
	}
	source := strings.TrimSpace(fallbackSource)
	if source == "" {
		source = "unknown"
	}
	fmt.Fprintf(os.Stderr, "[location] system lookup failed: %s; using fallback source: %s\n", systemErr.Error(), source)
}

func detectCurrentLocationFromEndpoint(ctx context.Context, httpClient *http.Client, endpoint string) (UserLocation, error) {
	return detectCurrentLocationFromProvider(ctx, httpClient, locationProvider{
		name:     "custom",
		endpoint: endpoint,
		parse:    parseIPAPI,
	})
}

func detectCurrentLocationWithProviders(ctx context.Context, httpClient *http.Client, providers []locationProvider) (UserLocation, error) {
	if len(providers) == 0 {
		return UserLocation{}, errors.New("no location providers configured")
	}

	var providerErrors []string
	for _, provider := range providers {
		location, err := detectCurrentLocationFromProvider(ctx, httpClient, provider)
		if err == nil {
			return location, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return UserLocation{}, err
		}
		providerErrors = append(providerErrors, fmt.Sprintf("%s: %s", provider.name, err.Error()))
	}

	if len(providerErrors) == 0 {
		return UserLocation{}, errors.New("could not determine location")
	}
	return UserLocation{}, fmt.Errorf("all location providers failed (%s)", strings.Join(providerErrors, " | "))
}

func detectCurrentLocationFromProvider(ctx context.Context, httpClient *http.Client, provider locationProvider) (UserLocation, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.endpoint, nil)
	if err != nil {
		return UserLocation{}, fmt.Errorf("create location request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", defaultUserAgent)

	res, err := httpClient.Do(req)
	if err != nil {
		return UserLocation{}, fmt.Errorf("location request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		snippet, _ := io.ReadAll(io.LimitReader(res.Body, 4<<10))
		msg := compactProviderErrorSnippet(string(snippet))
		if msg == "" {
			return UserLocation{}, errors.New(res.Status)
		}
		return UserLocation{}, fmt.Errorf("%s: %s", res.Status, msg)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if err != nil {
		return UserLocation{}, fmt.Errorf("read location response: %w", err)
	}

	location, err := provider.parse(body)
	if err != nil {
		return UserLocation{}, err
	}
	if location.Latitude == 0 && location.Longitude == 0 {
		return UserLocation{}, errors.New("provider returned empty coordinates")
	}
	if strings.TrimSpace(location.Source) == "" {
		location.Source = strings.TrimSpace(provider.name)
	}
	return location, nil
}

func parseIPAPI(body []byte) (UserLocation, error) {
	var payload struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		City      string  `json:"city"`
		Region    string  `json:"region"`
		Country   string  `json:"country_name"`
		Error     bool    `json:"error"`
		Reason    string  `json:"reason"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return UserLocation{}, fmt.Errorf("decode location response: %w", err)
	}

	if payload.Error {
		if payload.Reason == "" {
			payload.Reason = "unknown error"
		}
		return UserLocation{}, errors.New(payload.Reason)
	}

	return UserLocation{
		Latitude:  payload.Latitude,
		Longitude: payload.Longitude,
		City:      payload.City,
		Region:    payload.Region,
		Country:   payload.Country,
	}, nil
}

func parseIPWhoIs(body []byte) (UserLocation, error) {
	var payload struct {
		Success   bool    `json:"success"`
		Message   string  `json:"message"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		City      string  `json:"city"`
		Region    string  `json:"region"`
		Country   string  `json:"country"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return UserLocation{}, fmt.Errorf("decode location response: %w", err)
	}
	if !payload.Success {
		if strings.TrimSpace(payload.Message) == "" {
			payload.Message = "provider returned unsuccessful response"
		}
		return UserLocation{}, errors.New(payload.Message)
	}
	return UserLocation{
		Latitude:  payload.Latitude,
		Longitude: payload.Longitude,
		City:      payload.City,
		Region:    payload.Region,
		Country:   payload.Country,
	}, nil
}

func parseIPInfo(body []byte) (UserLocation, error) {
	var payload struct {
		Loc     string `json:"loc"`
		City    string `json:"city"`
		Region  string `json:"region"`
		Country string `json:"country"`
		Bogon   bool   `json:"bogon"`
		Error   struct {
			Title   string `json:"title"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return UserLocation{}, fmt.Errorf("decode location response: %w", err)
	}
	if payload.Bogon {
		return UserLocation{}, errors.New("bogon IP")
	}
	if payload.Error.Message != "" {
		return UserLocation{}, errors.New(payload.Error.Message)
	}
	parts := strings.Split(strings.TrimSpace(payload.Loc), ",")
	if len(parts) != 2 {
		return UserLocation{}, errors.New("provider did not return valid loc")
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return UserLocation{}, fmt.Errorf("parse latitude: %w", err)
	}
	lng, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return UserLocation{}, fmt.Errorf("parse longitude: %w", err)
	}
	return UserLocation{
		Latitude:  lat,
		Longitude: lng,
		City:      payload.City,
		Region:    payload.Region,
		Country:   payload.Country,
	}, nil
}

func compactProviderErrorSnippet(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	if strings.Contains(strings.ToLower(text), "<html") || strings.Contains(strings.ToLower(text), "<!doctype") {
		return ""
	}
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > locationErrorSnippetN {
		text = text[:locationErrorSnippetN]
	}
	return text
}
