package tui

import (
	"strings"
	"testing"

	"ingresso-finder-cli/model"

	"ingresso-finder-cli/store"

	"github.com/charmbracelet/bubbles/list"
)

func TestRenderMovieDetail_Empty(t *testing.T) {
	m := appModel{}
	m.movieList = list.New(nil, list.NewDefaultDelegate(), 10, 10)

	result := m.renderMovieDetail(60)
	if result != "" {
		t.Errorf("expected empty string when no movie is selected, got: %s", result)
	}
}

func TestRenderMovieDetail_WithMovie(t *testing.T) {
	// Setup mock movie
	movie := model.TheaterMovie{
		Title:         "O Auto da Compadecida",
		OriginalTitle: "O Auto da Compadecida", // Same as title, shouldn't render "Original:"
		ContentRating: "12",
		Duration:      "104",
		Rooms: []model.TheaterRoom{
			{
				Sessions: []model.TheaterSession{
					{Type: []string{"Nacional", "Normal"}},
				},
			},
			{
				Sessions: []model.TheaterSession{
					{Type: []string{"VIP", "Nacional"}}, // "Nacional" is duplicated, should be deduplicated
				},
			},
		},
	}

	item := movieItem{
		movie: movie,
	}

	m := appModel{}
	m.movieList = list.New([]list.Item{item}, list.NewDefaultDelegate(), 10, 10)
	m.movieList.Select(0)

	result := m.renderMovieDetail(60)

	// Assertions for Title and Icon
	if !strings.Contains(result, "🎬 O Auto da Compadecida") {
		t.Errorf("expected title with icon, got:\n%s", result)
	}

	// Assertions for Rating (should not add redundant "anos" if it's already just the number or handled)
	if !strings.Contains(result, "12") {
		t.Errorf("expected content rating '12', got:\n%s", result)
	}

	// Assertions for Duration
	if !strings.Contains(result, "⏱️ 104 min") {
		t.Errorf("expected formatted duration, got:\n%s", result)
	}

	// Assertions for Tags (Deduplicated and sorted)
	if !strings.Contains(result, "Nacional") || !strings.Contains(result, "Normal") || !strings.Contains(result, "VIP") {
		t.Errorf("expected deduplicated format tags, got:\n%s", result)
	}

	// Original title should not be rendered if it's identical
	if strings.Contains(result, "Original:") {
		t.Errorf("did not expect Original title to be rendered when it matches Title, got:\n%s", result)
	}
}

func TestRenderMovieDetail_OriginalTitleDifferent(t *testing.T) {
	movie := model.TheaterMovie{
		Title:         "Vingadores",
		OriginalTitle: "The Avengers",
		ContentRating: "14",
		Duration:      "143",
	}

	item := movieItem{
		movie: movie,
	}

	m := appModel{}
	m.movieList = list.New([]list.Item{item}, list.NewDefaultDelegate(), 10, 10)
	m.movieList.Select(0)

	result := m.renderMovieDetail(60)

	if !strings.Contains(result, "Original:") || !strings.Contains(result, "The Avengers") {
		t.Errorf("expected Original title to be rendered, got:\n%s", result)
	}
}

func TestRenderMovieDetail_GlobalSessions(t *testing.T) {
	movie := model.TheaterMovie{
		Title:         "Duna: Parte 2",
		ContentRating: "14",
	}

	item := movieItem{
		movie: movie,
		globalSessions: []sessionWithTheater{
			{theater: model.Theater{Name: "Cinema 1"}},
			{theater: model.Theater{Name: "Cinema 2"}},
			{theater: model.Theater{Name: "Cinema 1"}}, // Deduplication check
		},
	}

	m := appModel{}
	m.movieList = list.New([]list.Item{item}, list.NewDefaultDelegate(), 10, 10)
	m.movieList.Select(0)

	result := m.renderMovieDetail(60)

	if !strings.Contains(result, "Cinemas:") || !strings.Contains(result, "2 locais") {
		t.Errorf("expected global sessions to count unique theaters (2), got:\n%s", result)
	}
}

func TestRenderMovieDetail_WithRatingsAndPlot(t *testing.T) {
	movie := model.TheaterMovie{
		Title: "The Matrix",
	}

	item := movieItem{
		movie: movie,
	}

	m := appModel{
		movieRatings: map[string]store.OMDbRating{
			"The Matrix": {
				ImdbRating: "8.7",
				Rotten:     "83%",
				Genre:      "Action, Sci-Fi",
				Director:   "Lana Wachowski, Lilly Wachowski",
				Plot:       "A computer hacker learns from mysterious rebels about the true nature of his reality.",
			},
		},
	}

	m.movieList = list.New([]list.Item{item}, list.NewDefaultDelegate(), 10, 10)
	m.movieList.Select(0)

	result := m.renderMovieDetail(60)

	// Regressão: Verifica se as notas combinadas (IMDb + Rotten) aparecem na mesma linha
	if !strings.Contains(result, "Notas:") || !strings.Contains(result, "8.7/10") || !strings.Contains(result, "83%") {
		t.Errorf("expected combined ratings (IMDb and Rotten), got:\n%s", result)
	}

	// Regressão: Verifica se Metadados Extras aparecem
	if !strings.Contains(result, "Gênero:") || !strings.Contains(result, "Action, Sci-Fi") {
		t.Errorf("expected genre to be rendered, got:\n%s", result)
	}
	if !strings.Contains(result, "Diretor:") || !strings.Contains(result, "Lana Wachowski") {
		t.Errorf("expected director to be rendered, got:\n%s", result)
	}

	// Regressão: Verifica se a Sinopse foi anexada
	if !strings.Contains(result, "A computer hacker learns") {
		t.Errorf("expected plot to be rendered, got:\n%s", result)
	}
}

func TestRenderMovieDetail_NotFoundInOMDb(t *testing.T) {
	movie := model.TheaterMovie{
		Title: "Filme Obscuro Independente",
	}

	item := movieItem{
		movie: movie,
	}

	m := appModel{
		movieRatings: map[string]store.OMDbRating{
			"Filme Obscuro Independente": {
				NotFound: true,
			},
		},
	}

	m.movieList = list.New([]list.Item{item}, list.NewDefaultDelegate(), 10, 10)
	m.movieList.Select(0)

	result := m.renderMovieDetail(60)

	if !strings.Contains(result, "Filme não encontrado no IMDb") {
		t.Errorf("expected 'Not Found' message for uncataloged movies, got:\n%s", result)
	}
}
