package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ingresso-finder-cli/model"
)

const (
	apiBaseURL         = "https://api-content.ingresso.com/v0"
	checkoutBaseURL    = "https://api.ingresso.com/v1"
	defaultUserAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5.2 Safari/605.1.15"
	defaultMaxAttempts = 3
	defaultRetryBase   = 200 * time.Millisecond
	defaultRetryCap    = 1200 * time.Millisecond
)

// Client wraps HTTP access to the Ingresso content API.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	checkoutURL string
	userAgent   string
	maxAttempts int
	retryBase   time.Duration
	retryCap    time.Duration
}

// APIError is returned when the Ingresso API responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Status     string
	Endpoint   string
	Body       string
}

func (e *APIError) Error() string {
	if e == nil {
		return "ingresso api error"
	}
	return fmt.Sprintf("ingresso api error: %s: %s", e.Status, e.Body)
}

// IsNotFound reports whether the error represents a 404 from the API.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// NewClient creates a new API client. If httpClient is nil, a default client is used.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 12 * time.Second}
	}
	return &Client{
		httpClient:  httpClient,
		baseURL:     apiBaseURL,
		checkoutURL: checkoutBaseURL,
		userAgent:   defaultUserAgent,
		maxAttempts: defaultMaxAttempts,
		retryBase:   defaultRetryBase,
		retryCap:    defaultRetryCap,
	}
}

// GetCityInfoByName fetches city info by its name.
func (c *Client) GetCityInfoByName(ctx context.Context, cityName string) (model.City, error) {
	name := strings.TrimSpace(cityName)
	if name == "" {
		return model.City{}, errors.New("city name is required")
	}
	endpoint := fmt.Sprintf("%s/states/city/name/%s", c.baseURL, url.PathEscape(name))

	var city model.City
	if err := c.getJSON(ctx, endpoint, &city); err != nil {
		return model.City{}, err
	}
	if city.Id == "" {
		return model.City{}, errors.New("city not found")
	}
	return city, nil
}

// GetCities returns the full list of cities available in the API.
func (c *Client) GetCities(ctx context.Context) ([]model.City, error) {
	endpoint := fmt.Sprintf("%s/states", c.baseURL)

	var states []model.State
	if err := c.getJSON(ctx, endpoint, &states); err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, errors.New("no states found")
	}

	var cities []model.City
	for _, state := range states {
		cities = append(cities, state.Cities...)
	}
	if len(cities) == 0 {
		return nil, errors.New("no cities found")
	}
	return cities, nil
}

// GetSessionDetails fetches session detail including sections for seat maps.
func (c *Client) GetSessionDetails(ctx context.Context, sessionID string) (model.SessionDetail, error) {
	if sessionID == "" {
		return model.SessionDetail{}, errors.New("session id is required")
	}
	endpoint := fmt.Sprintf("%s/sessions/%s", c.checkoutURL, sessionID)
	var detail model.SessionDetail
	if err := c.getJSON(ctx, endpoint, &detail); err != nil {
		return model.SessionDetail{}, err
	}
	return detail, nil
}

// GetSeatMap fetches seat map for a given session and section.
func (c *Client) GetSeatMap(ctx context.Context, sessionID string, sectionID string) (model.SeatMap, error) {
	if sessionID == "" || sectionID == "" {
		return model.SeatMap{}, errors.New("session id and section id are required")
	}
	endpoint := fmt.Sprintf("%s/sessions/%s/sections/%s/seats", c.checkoutURL, sessionID, sectionID)
	var seats model.SeatMap
	if err := c.getJSON(ctx, endpoint, &seats); err != nil {
		return model.SeatMap{}, err
	}
	return seats, nil
}

// GetTheatersByCity fetches theaters for a given city.
func (c *Client) GetTheatersByCity(ctx context.Context, cityID string) ([]model.Theater, error) {
	if cityID == "" {
		return nil, errors.New("city id is required")
	}
	endpoint := fmt.Sprintf("%s/theaters/city/%s", c.baseURL, cityID)

	var theaters []model.Theater
	if err := c.getJSON(ctx, endpoint, &theaters); err != nil {
		return nil, err
	}
	return theaters, nil
}

// GetSessionsByCityAndTheater fetches sessions for a city/theater pair.
func (c *Client) GetSessionsByCityAndTheater(ctx context.Context, cityID string, theaterID string, date *time.Time) ([]model.TheaterSessionDay, error) {
	if cityID == "" || theaterID == "" {
		return nil, errors.New("city id and theater id are required")
	}

	endpoint := fmt.Sprintf("%s/sessions/city/%s/theater/%s", c.baseURL, cityID, theaterID)
	if date != nil {
		endpoint = endpoint + "?date=" + date.Format(time.DateOnly)
	}

	var days []model.TheaterSessionDay
	if err := c.getJSON(ctx, endpoint, &days); err != nil {
		return nil, err
	}
	return days, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, out any) error {
	maxAttempts := c.maxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "application/json")

		res, err := c.httpClient.Do(req)
		if err != nil {
			if c.shouldRetryNetworkError(err) && attempt < maxAttempts {
				if waitErr := c.waitRetry(ctx, attempt); waitErr != nil {
					return waitErr
				}
				continue
			}
			return fmt.Errorf("request failed: %w", err)
		}

		if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
			snippet, _ := io.ReadAll(io.LimitReader(res.Body, 8<<10))
			_ = res.Body.Close()

			apiErr := &APIError{
				StatusCode: res.StatusCode,
				Status:     res.Status,
				Endpoint:   endpoint,
				Body:       strings.TrimSpace(string(snippet)),
			}
			if c.shouldRetryStatus(res.StatusCode) && attempt < maxAttempts {
				if waitErr := c.waitRetry(ctx, attempt); waitErr != nil {
					return waitErr
				}
				continue
			}
			return apiErr
		}

		dec := json.NewDecoder(res.Body)
		err = dec.Decode(out)
		_ = res.Body.Close()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("decode response from %s: %w", endpoint, err)
		}
		return nil
	}

	return errors.New("request failed after retries")
}

func (c *Client) shouldRetryStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= http.StatusInternalServerError
}

func (c *Client) shouldRetryNetworkError(err error) bool {
	if err == nil {
		return false
	}
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

func (c *Client) waitRetry(ctx context.Context, attempt int) error {
	delay := c.retryDelay(attempt)
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	base := c.retryBase
	if base <= 0 {
		base = defaultRetryBase
	}
	cap := c.retryCap
	if cap <= 0 {
		cap = defaultRetryCap
	}

	delay := base
	for i := 1; i < attempt; i++ {
		if delay >= cap/2 {
			return cap
		}
		delay *= 2
	}
	if delay > cap {
		return cap
	}
	return delay
}
