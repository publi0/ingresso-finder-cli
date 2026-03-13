package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	omdbBaseURL = "http://www.omdbapi.com/"
	omdbTimeout = 5 * time.Second
)

// OMDbRating represents a single rating from an OMDb source.
type OMDbRating struct {
	Source string `json:"Source"`
	Value  string `json:"Value"`
}

// OMDbResponse represents the JSON payload returned by the OMDb API.
type OMDbResponse struct {
	Title      string       `json:"Title"`
	Year       string       `json:"Year"`
	Rated      string       `json:"Rated"`
	Runtime    string       `json:"Runtime"`
	Genre      string       `json:"Genre"`
	Director   string       `json:"Director"`
	Plot       string       `json:"Plot"`
	Language   string       `json:"Language"`
	Poster     string       `json:"Poster"`
	Ratings    []OMDbRating `json:"Ratings"`
	Metascore  string       `json:"Metascore"`
	ImdbRating string       `json:"imdbRating"`
	ImdbID     string       `json:"imdbID"`
	Response   string       `json:"Response"` // "True" or "False"
	Error      string       `json:"Error"`
}

// FetchMovieData queries the OMDb API for movie details by title.
func FetchMovieData(title, originalTitle string) (*OMDbResponse, error) {
	apiKey := os.Getenv("OMDB_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OMDB_API_KEY environment variable is not set")
	}

	titlesToTry := []string{}

	cleanOrig := cleanTitleForSearch(originalTitle)
	cleanPt := cleanTitleForSearch(title)

	if cleanOrig != "" {
		titlesToTry = append(titlesToTry, cleanOrig)
	}
	if cleanPt != "" && cleanPt != cleanOrig {
		titlesToTry = append(titlesToTry, cleanPt)
	}
	if len(titlesToTry) == 0 {
		return nil, fmt.Errorf("no valid title provided for search")
	}

	var lastErr error
	for _, t := range titlesToTry {
		reqURL, err := url.Parse(omdbBaseURL)
		if err != nil {
			return nil, err
		}

		q := reqURL.Query()
		q.Set("apikey", apiKey)
		q.Set("t", t)
		reqURL.RawQuery = q.Encode()

		client := &http.Client{Timeout: omdbTimeout}
		resp, err := client.Get(reqURL.String())
		if err != nil {
			lastErr = fmt.Errorf("failed to reach omdb api: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("omdb api returned status code %d", resp.StatusCode)
			continue
		}

		var data OMDbResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to decode omdb response: %w", err)
			continue
		}

		if data.Response == "True" {
			return &data, nil
		} else {
			lastErr = fmt.Errorf("omdb api error: %s", data.Error)
		}
	}

	return nil, lastErr
}

// cleanTitleForSearch removes common artifacts from localized movie titles
// to improve the hit rate on OMDb (which is primarily english/original titles).
func cleanTitleForSearch(title string) string {
	if title == "" {
		return ""
	}

	// Simple heuristics: remove " - Dublado", " - Legendado", etc.
	parts := strings.Split(title, " - ")
	if len(parts) > 1 {
		title = parts[0]
	}

	// Remove common PT-BR promotional suffixes
	suffixes := []string{
		" (Dublado)", " (Legendado)", " (Nacional)", " 3D", " O Filme",
	}
	for _, suffix := range suffixes {
		title = strings.ReplaceAll(title, suffix, "")
	}

	return strings.TrimSpace(title)
}
