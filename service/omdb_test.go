package service

import (
	"testing"
)

func TestCleanTitleForSearch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty title",
			input:    "",
			expected: "",
		},
		{
			name:     "Clean title without artifacts",
			input:    "O Auto da Compadecida 2",
			expected: "O Auto da Compadecida 2",
		},
		{
			name:     "Title with hyphen and Dublado",
			input:    "Kung Fu Panda 4 - Dublado",
			expected: "Kung Fu Panda 4",
		},
		{
			name:     "Title with hyphen and Legendado",
			input:    "Duna: Parte 2 - Legendado",
			expected: "Duna: Parte 2",
		},
		{
			name:     "Title with hyphen and Nacional",
			input:    "Nosso Sonho - Nacional",
			expected: "Nosso Sonho",
		},
		{
			name:     "Title with parenthesis Dublado",
			input:    "Divertida Mente 2 (Dublado)",
			expected: "Divertida Mente 2",
		},
		{
			name:     "Title with parenthesis Legendado",
			input:    "O Oppenheimer (Legendado)",
			expected: "O Oppenheimer",
		},
		{
			name:     "Title with 3D suffix",
			input:    "Avatar: O Caminho da Água 3D",
			expected: "Avatar: O Caminho da Água",
		},
		{
			name:     "Title with O Filme suffix",
			input:    "Super Mario Bros O Filme",
			expected: "Super Mario Bros",
		},
		{
			name:     "Combined artifacts - 3D and Dublado",
			input:    "Deadpool & Wolverine 3D (Dublado)",
			expected: "Deadpool & Wolverine",
		},
		{
			name:     "Combined artifacts - O Filme and hyphen",
			input:    "Peppa Pig O Filme - Dublado",
			expected: "Peppa Pig",
		},
		{
			name:     "Trims leading and trailing whitespaces",
			input:    "   The Batman   ",
			expected: "The Batman",
		},
		{
			name:     "Preserves hyphens without spaces",
			input:    "Spider-Man",
			expected: "Spider-Man",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanTitleForSearch(tt.input)
			if result != tt.expected {
				t.Errorf("cleanTitleForSearch(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}
