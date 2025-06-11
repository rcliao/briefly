package visual

import (
	"briefly/internal/core"
	"context"
	"strings"
	"testing"
)

func TestAnalyzeThemesSimple(t *testing.T) {
	service := &Service{
		llmService: nil, // Test without LLM service
		outputDir:  "/tmp",
	}

	tests := []struct {
		name     string
		digest   *core.Digest
		expected int // Expected number of themes
	}{
		{
			name: "AI content should detect AI theme",
			digest: &core.Digest{
				Title:         "AI Developments",
				Content:       "This digest covers artificial intelligence and machine learning advances.",
				DigestSummary: "AI and ML topics",
			},
			expected: 1,
		},
		{
			name: "Development content should detect dev theme",
			digest: &core.Digest{
				Title:         "Programming Updates",
				Content:       "Software development and programming best practices.",
				DigestSummary: "Code and development",
			},
			expected: 1,
		},
		{
			name: "Security content should detect security theme",
			digest: &core.Digest{
				Title:         "Security Updates",
				Content:       "Cybersecurity vulnerabilities and privacy concerns.",
				DigestSummary: "Security and privacy",
			},
			expected: 1,
		},
		{
			name: "Mixed content should detect multiple themes",
			digest: &core.Digest{
				Title:         "Tech Digest",
				Content:       "AI development, software programming, and security updates.",
				DigestSummary: "AI, development, and security",
			},
			expected: 3,
		},
		{
			name: "Generic content should get default theme",
			digest: &core.Digest{
				Title:         "News",
				Content:       "Various news and updates.",
				DigestSummary: "General updates",
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			themes, err := service.AnalyzeContentThemes(context.Background(), tt.digest)
			if err != nil {
				t.Fatalf("AnalyzeContentThemes() error = %v", err)
			}

			if len(themes) != tt.expected {
				t.Errorf("AnalyzeContentThemes() got %d themes, expected %d", len(themes), tt.expected)
			}

			// Verify each theme has required fields
			for _, theme := range themes {
				if theme.Theme == "" {
					t.Error("Theme name should not be empty")
				}
				if theme.Confidence <= 0 {
					t.Error("Theme confidence should be positive")
				}
				if theme.Category == "" {
					t.Error("Theme category should not be empty")
				}
				if theme.Description == "" {
					t.Error("Theme description should not be empty")
				}
			}
		})
	}
}

func TestGenerateBannerPrompt(t *testing.T) {
	service := &Service{
		llmService: nil,
		outputDir:  "/tmp",
	}

	themes := []core.ContentTheme{
		{
			Theme:       "AI & Machine Learning",
			Keywords:    []string{"ai", "machine learning"},
			Confidence:  0.8,
			Category:    "💡 Innovation",
			Description: "Artificial intelligence developments",
		},
	}

	tests := []struct {
		name     string
		themes   []core.ContentTheme
		style    string
		contains []string // Strings that should be in the prompt
	}{
		{
			name:     "Minimalist style",
			themes:   themes,
			style:    "minimalist",
			contains: []string{"minimalist", "artificial intelligence"},
		},
		{
			name:     "Tech style",
			themes:   themes,
			style:    "tech",
			contains: []string{"tech", "artificial intelligence"},
		},
		{
			name:     "Professional style",
			themes:   themes,
			style:    "professional",
			contains: []string{"professional", "artificial intelligence"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := service.GenerateBannerPrompt(context.Background(), tt.themes, tt.style)
			if err != nil {
				t.Fatalf("GenerateBannerPrompt() error = %v", err)
			}

			if prompt == "" {
				t.Error("GenerateBannerPrompt() should not return empty prompt")
			}

			for _, substr := range tt.contains {
				if !contains(prompt, substr) {
					t.Errorf("GenerateBannerPrompt() prompt should contain '%s', got: %s", substr, prompt)
				}
			}
		})
	}
}

func TestGenerateAltText(t *testing.T) {
	service := &Service{
		llmService: nil,
		outputDir:  "/tmp",
	}

	themes := []core.ContentTheme{
		{
			Theme:      "AI & Machine Learning",
			Confidence: 0.8,
		},
		{
			Theme:      "Software Development",
			Confidence: 0.7,
		},
	}

	altText, err := service.GenerateAltText(context.Background(), themes)
	if err != nil {
		t.Fatalf("GenerateAltText() error = %v", err)
	}

	if altText == "" {
		t.Error("GenerateAltText() should not return empty string")
	}

	// Should contain theme names in lowercase
	if !contains(altText, "ai & machine learning") && !contains(altText, "software development") {
		t.Errorf("GenerateAltText() should contain theme names, got: %s", altText)
	}
}

func TestGenerateAltTextWithNoThemes(t *testing.T) {
	service := &Service{
		llmService: nil,
		outputDir:  "/tmp",
	}

	altText, err := service.GenerateAltText(context.Background(), []core.ContentTheme{})
	if err != nil {
		t.Fatalf("GenerateAltText() error = %v", err)
	}

	expected := "AI-generated banner image for technology digest"
	if altText != expected {
		t.Errorf("GenerateAltText() with no themes = %s, expected %s", altText, expected)
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}
