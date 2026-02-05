package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type testItem struct {
	value string
}

func (t testItem) Title() string       { return t.value }
func (t testItem) Description() string { return "" }
func (t testItem) FilterValue() string { return strings.ToLower(t.value) }

func newFilterModel(items []list.Item) *appModel {
	model := New().(appModel)
	model.state = stateSelectCity
	model.cityList = newList("Select City")
	model.cityList.SetItems(items)
	return &model
}

func TestHandleFilterInput_AppendsRunes(t *testing.T) {
	m := newFilterModel([]list.Item{
		testItem{value: "Barueri"},
		testItem{value: "Sao Paulo"},
	})

	if !m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}) {
		t.Fatal("expected filter input to be handled")
	}
	if got := m.cityList.FilterValue(); got != "b" {
		t.Fatalf("expected filter value to be %q, got %q", "b", got)
	}

	if !m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}) {
		t.Fatal("expected filter input to be handled")
	}
	if got := m.cityList.FilterValue(); got != "ba" {
		t.Fatalf("expected filter value to be %q, got %q", "ba", got)
	}
}

func TestHandleFilterInput_Backspace(t *testing.T) {
	m := newFilterModel([]list.Item{
		testItem{value: "Barueri"},
		testItem{value: "Sao Paulo"},
	})

	_ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	_ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if got := m.cityList.FilterValue(); got != "ba" {
		t.Fatalf("expected filter value to be %q, got %q", "ba", got)
	}

	if !m.handleFilterInput(tea.KeyMsg{Type: tea.KeyBackspace}) {
		t.Fatal("expected backspace to be handled")
	}
	if got := m.cityList.FilterValue(); got != "b" {
		t.Fatalf("expected filter value to be %q, got %q", "b", got)
	}
}

func TestHandleFilterInput_Space(t *testing.T) {
	m := newFilterModel([]list.Item{
		testItem{value: "Rio de Janeiro"},
	})

	_ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	_ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	_ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})

	if !m.handleFilterInput(tea.KeyMsg{Type: tea.KeySpace}) {
		t.Fatal("expected space to be handled")
	}

	if got := m.cityList.FilterValue(); got != "rio " {
		t.Fatalf("expected filter value to be %q, got %q", "rio ", got)
	}
}
