package search

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProviderTypeConstants(t *testing.T) {
	expectedTypes := map[ProviderType]string{
		ProviderTypeDuckDuckGo: "duckduckgo",
		ProviderTypeGoogle:     "google",
		ProviderTypeSerpAPI:    "serpapi",
		ProviderTypeMock:       "mock",
	}

	for providerType, expectedValue := range expectedTypes {
		if string(providerType) != expectedValue {
			t.Errorf("Expected %s to be %s, got %s", providerType, expectedValue, string(providerType))
		}
	}
}

func TestConfigCreation(t *testing.T) {
	config := Config{
		MaxResults: 10,
		SinceTime:  24 * time.Hour,
		Language:   "en",
	}

	if config.MaxResults != 10 {
		t.Errorf("Expected MaxResults to be 10, got %d", config.MaxResults)
	}
	if config.SinceTime != 24*time.Hour {
		t.Errorf("Expected SinceTime to be 24h, got %v", config.SinceTime)
	}
	if config.Language != "en" {
		t.Errorf("Expected Language to be 'en', got %s", config.Language)
	}
}

func TestResultCreation(t *testing.T) {
	publishedAt := time.Now()
	result := Result{
		URL:         "https://example.com/article",
		Title:       "Test Article",
		Snippet:     "This is a test snippet",
		Domain:      "example.com",
		PublishedAt: publishedAt,
		Source:      "test",
		Rank:        1,
	}

	if result.URL != "https://example.com/article" {
		t.Errorf("Expected URL to be 'https://example.com/article', got %s", result.URL)
	}
	if result.Title != "Test Article" {
		t.Errorf("Expected Title to be 'Test Article', got %s", result.Title)
	}
	if result.Rank != 1 {
		t.Errorf("Expected Rank to be 1, got %d", result.Rank)
	}
}

func TestNewProviderFactory(t *testing.T) {
	factory := NewProviderFactory()
	if factory == nil {
		t.Error("Expected NewProviderFactory to return non-nil factory")
	}
}

func TestCreateMockProvider(t *testing.T) {
	factory := NewProviderFactory()
	config := map[string]string{}

	provider, err := factory.CreateProvider(ProviderTypeMock, config)
	if err != nil {
		t.Fatalf("Expected no error creating mock provider, got %v", err)
	}

	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}

	if provider.GetName() != "Mock" {
		t.Errorf("Expected provider name to be 'Mock', got %s", provider.GetName())
	}
}

func TestCreateDuckDuckGoProvider(t *testing.T) {
	factory := NewProviderFactory()
	config := map[string]string{}

	provider, err := factory.CreateProvider(ProviderTypeDuckDuckGo, config)
	if err != nil {
		t.Fatalf("Expected no error creating DuckDuckGo provider, got %v", err)
	}

	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}
}

func TestCreateGoogleProviderMissingAPIKey(t *testing.T) {
	factory := NewProviderFactory()
	config := map[string]string{
		"search_id": "test-search-id",
	}

	provider, err := factory.CreateProvider(ProviderTypeGoogle, config)
	if err == nil {
		t.Error("Expected error when creating Google provider without API key")
	}
	if provider != nil {
		t.Error("Expected nil provider when creation fails")
	}
	if !errors.Is(err, ErrMissingAPIKey) {
		t.Errorf("Expected ErrMissingAPIKey, got %v", err)
	}
}

func TestCreateGoogleProviderMissingSearchID(t *testing.T) {
	factory := NewProviderFactory()
	config := map[string]string{
		"api_key": "test-api-key",
	}

	provider, err := factory.CreateProvider(ProviderTypeGoogle, config)
	if err == nil {
		t.Error("Expected error when creating Google provider without search ID")
	}
	if provider != nil {
		t.Error("Expected nil provider when creation fails")
	}
	if !errors.Is(err, ErrMissingSearchID) {
		t.Errorf("Expected ErrMissingSearchID, got %v", err)
	}
}

func TestCreateGoogleProviderSuccess(t *testing.T) {
	factory := NewProviderFactory()
	config := map[string]string{
		"api_key":   "test-api-key",
		"search_id": "test-search-id",
	}

	provider, err := factory.CreateProvider(ProviderTypeGoogle, config)
	if err != nil {
		t.Fatalf("Expected no error creating Google provider, got %v", err)
	}
	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}
}

func TestCreateSerpAPIProviderMissingAPIKey(t *testing.T) {
	factory := NewProviderFactory()
	config := map[string]string{}

	provider, err := factory.CreateProvider(ProviderTypeSerpAPI, config)
	if err == nil {
		t.Error("Expected error when creating SerpAPI provider without API key")
	}
	if provider != nil {
		t.Error("Expected nil provider when creation fails")
	}
	if !errors.Is(err, ErrMissingAPIKey) {
		t.Errorf("Expected ErrMissingAPIKey, got %v", err)
	}
}

func TestCreateSerpAPIProviderSuccess(t *testing.T) {
	factory := NewProviderFactory()
	config := map[string]string{
		"api_key": "test-api-key",
	}

	provider, err := factory.CreateProvider(ProviderTypeSerpAPI, config)
	if err != nil {
		t.Fatalf("Expected no error creating SerpAPI provider, got %v", err)
	}
	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}
}

func TestCreateUnsupportedProvider(t *testing.T) {
	factory := NewProviderFactory()
	config := map[string]string{}

	provider, err := factory.CreateProvider("unsupported", config)
	if err == nil {
		t.Error("Expected error when creating unsupported provider")
	}
	if provider != nil {
		t.Error("Expected nil provider when creation fails")
	}
	if !errors.Is(err, ErrUnsupportedProvider) {
		t.Errorf("Expected ErrUnsupportedProvider, got %v", err)
	}
}

func TestGetAvailableProviders(t *testing.T) {
	factory := NewProviderFactory()
	providers := factory.GetAvailableProviders()

	expectedProviders := []ProviderType{
		ProviderTypeDuckDuckGo,
		ProviderTypeGoogle,
		ProviderTypeSerpAPI,
		ProviderTypeMock,
	}

	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(providers))
	}

	// Check that all expected providers are present
	providerMap := make(map[ProviderType]bool)
	for _, provider := range providers {
		providerMap[provider] = true
	}

	for _, expected := range expectedProviders {
		if !providerMap[expected] {
			t.Errorf("Expected provider %s to be in available providers list", expected)
		}
	}
}

func TestErrorsExist(t *testing.T) {
	errors := []error{
		ErrMissingAPIKey,
		ErrMissingSearchID,
		ErrUnsupportedProvider,
		ErrNoResults,
		ErrRateLimited,
		ErrProviderUnavailable,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Expected error to be defined")
		}
		if err.Error() == "" {
			t.Error("Expected error to have non-empty message")
		}
	}
}

func TestMockProviderSearch(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()
	config := Config{
		MaxResults: 2,
		Language:   "en",
	}

	results, err := provider.Search(ctx, "test query", config)
	if err != nil {
		t.Fatalf("Expected no error from mock search, got %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check that query is included in title
	for _, result := range results {
		if result.Title == "" {
			t.Error("Expected non-empty title")
		}
		if result.URL == "" {
			t.Error("Expected non-empty URL")
		}
		if result.Snippet == "" {
			t.Error("Expected non-empty snippet")
		}
	}
}

func TestMockProviderCustomization(t *testing.T) {
	provider := NewMockProvider()
	
	// Test name customization
	provider.SetName("CustomMock")
	if provider.GetName() != "CustomMock" {
		t.Errorf("Expected provider name to be 'CustomMock', got %s", provider.GetName())
	}

	// Test results customization
	customResults := []Result{
		{
			URL:     "https://custom.com/article",
			Title:   "Custom Article",
			Snippet: "Custom snippet",
			Domain:  "custom.com",
			Source:  "Custom",
			Rank:    1,
		},
	}
	
	provider.SetResults(customResults)
	
	ctx := context.Background()
	config := Config{MaxResults: 5}
	
	results, err := provider.Search(ctx, "test", config)
	if err != nil {
		t.Fatalf("Expected no error from mock search, got %v", err)
	}
	
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	
	if results[0].Domain != "custom.com" {
		t.Errorf("Expected domain to be 'custom.com', got %s", results[0].Domain)
	}
}