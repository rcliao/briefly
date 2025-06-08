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

// Test LegacyProviderAdapter functionality

type mockLegacyProvider struct {
	name    string
	results []LegacySearchResult
}

func (m *mockLegacyProvider) Search(query string, maxResults int) ([]LegacySearchResult, error) {
	if len(m.results) == 0 {
		return []LegacySearchResult{
			{
				Title:       "Legacy Result 1: " + query,
				URL:         "https://legacy.example.com/1",
				Snippet:     "Legacy snippet for " + query,
				Source:      "legacy.example.com",
				PublishedAt: "2024-01-01",
				Rank:        1,
			},
			{
				Title:       "Legacy Result 2: " + query,
				URL:         "https://legacy.example.com/2",
				Snippet:     "Another legacy snippet",
				Source:      "legacy.example.com",
				PublishedAt: "2024-01-02",
				Rank:        2,
			},
		}, nil
	}
	
	// Return limited results based on maxResults
	if maxResults < len(m.results) {
		return m.results[:maxResults], nil
	}
	return m.results, nil
}

func (m *mockLegacyProvider) GetName() string {
	return m.name
}

func TestLegacyProviderAdapter(t *testing.T) {
	adapter := NewLegacyProviderAdapter(NewMockProvider())

	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}

	// Test search functionality
	results, err := adapter.Search("test query", 2)
	if err != nil {
		t.Fatalf("Expected no error from adapter search, got %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check result format conversion
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
		if result.Source == "" {
			t.Error("Expected non-empty source")
		}
		if result.Rank <= 0 {
			t.Error("Expected positive rank")
		}
	}

	// Test name passthrough
	if adapter.GetName() != "Mock" {
		t.Errorf("Expected adapter name to be 'Mock', got %s", adapter.GetName())
	}
}

func TestModernProviderAdapter(t *testing.T) {
	legacyProvider := &mockLegacyProvider{name: "MockLegacy"}
	adapter := NewModernProviderAdapter(legacyProvider)

	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}

	// Test search functionality
	ctx := context.Background()
	config := Config{
		MaxResults: 2,
		Language:   "en",
	}

	results, err := adapter.Search(ctx, "test query", config)
	if err != nil {
		t.Fatalf("Expected no error from adapter search, got %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check result format conversion
	for i, result := range results {
		if result.Title == "" {
			t.Error("Expected non-empty title")
		}
		if result.URL == "" {
			t.Error("Expected non-empty URL")
		}
		if result.Snippet == "" {
			t.Error("Expected non-empty snippet")
		}
		if result.Domain == "" {
			t.Error("Expected non-empty domain")
		}
		if result.Source != "MockLegacy" {
			t.Errorf("Expected source to be 'MockLegacy', got %s", result.Source)
		}
		if result.Rank != i+1 {
			t.Errorf("Expected rank %d, got %d", i+1, result.Rank)
		}
		if result.PublishedAt.IsZero() {
			t.Error("Expected non-zero published date")
		}
	}

	// Test name passthrough
	if adapter.GetName() != "MockLegacy" {
		t.Errorf("Expected adapter name to be 'MockLegacy', got %s", adapter.GetName())
	}
}

func TestModernProviderAdapter_InvalidDate(t *testing.T) {
	legacyProvider := &mockLegacyProvider{
		name: "MockLegacy",
		results: []LegacySearchResult{
			{
				Title:       "Test Result",
				URL:         "https://example.com",
				Snippet:     "Test snippet",
				Source:      "example.com",
				PublishedAt: "invalid-date",
				Rank:        1,
			},
		},
	}
	adapter := NewModernProviderAdapter(legacyProvider)

	ctx := context.Background()
	config := Config{MaxResults: 1}

	results, err := adapter.Search(ctx, "test", config)
	if err != nil {
		t.Fatalf("Expected no error from adapter search, got %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Should handle invalid date gracefully with zero time
	if !results[0].PublishedAt.IsZero() {
		t.Error("Expected zero time for invalid date")
	}
}

func TestModernProviderAdapter_EmptyDate(t *testing.T) {
	legacyProvider := &mockLegacyProvider{
		name: "MockLegacy",
		results: []LegacySearchResult{
			{
				Title:       "Test Result",
				URL:         "https://example.com",
				Snippet:     "Test snippet",
				Source:      "example.com",
				PublishedAt: "",
				Rank:        1,
			},
		},
	}
	adapter := NewModernProviderAdapter(legacyProvider)

	ctx := context.Background()
	config := Config{MaxResults: 1}

	results, err := adapter.Search(ctx, "test", config)
	if err != nil {
		t.Fatalf("Expected no error from adapter search, got %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Should handle empty date gracefully with zero time
	if !results[0].PublishedAt.IsZero() {
		t.Error("Expected zero time for empty date")
	}
}

func TestLegacyProviderAdapter_MaxResultsLimiting(t *testing.T) {
	adapter := NewLegacyProviderAdapter(NewMockProvider())

	// Test with smaller max results than provider returns
	results, err := adapter.Search("test", 1)
	if err != nil {
		t.Fatalf("Expected no error from adapter search, got %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result when maxResults=1, got %d", len(results))
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test zero-value config
	var config Config
	if config.MaxResults != 0 {
		t.Error("Expected default MaxResults to be 0")
	}
	if config.SinceTime != 0 {
		t.Error("Expected default SinceTime to be 0")
	}
	if config.Language != "" {
		t.Error("Expected default Language to be empty")
	}
}

func TestResultDefaults(t *testing.T) {
	// Test zero-value result
	var result Result
	if result.URL != "" {
		t.Error("Expected default URL to be empty")
	}
	if result.Rank != 0 {
		t.Error("Expected default Rank to be 0")
	}
	if !result.PublishedAt.IsZero() {
		t.Error("Expected default PublishedAt to be zero time")
	}
}

func TestMockProviderWithContext(t *testing.T) {
	provider := NewMockProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	config := Config{
		MaxResults: 1,
		Language:   "en",
	}

	results, err := provider.Search(ctx, "test query", config)
	if err != nil {
		t.Fatalf("Expected no error from mock search with context, got %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestMockProviderCancelledContext(t *testing.T) {
	provider := NewMockProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	config := Config{MaxResults: 1}

	// Mock provider should still work even with cancelled context
	// since it doesn't actually respect cancellation
	results, err := provider.Search(ctx, "test", config)
	if err != nil {
		t.Fatalf("Mock provider should not fail with cancelled context, got %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestProviderTypeStringValues(t *testing.T) {
	// Test that provider type constants have expected string values
	testCases := map[ProviderType]string{
		ProviderTypeDuckDuckGo: "duckduckgo",
		ProviderTypeGoogle:     "google",
		ProviderTypeSerpAPI:    "serpapi",
		ProviderTypeMock:       "mock",
	}

	for providerType, expectedString := range testCases {
		if string(providerType) != expectedString {
			t.Errorf("Expected %s to equal %s", string(providerType), expectedString)
		}
	}
}

// Test error handling edge cases

type errorProvider struct{}

func (e *errorProvider) Search(ctx context.Context, query string, config Config) ([]Result, error) {
	return nil, ErrProviderUnavailable
}

func (e *errorProvider) GetName() string {
	return "ErrorProvider"
}

func TestLegacyProviderAdapter_ErrorHandling(t *testing.T) {
	errorProvider := &errorProvider{}
	adapter := NewLegacyProviderAdapter(errorProvider)

	results, err := adapter.Search("test", 10)
	if err == nil {
		t.Error("Expected error from adapter when underlying provider fails")
	}
	if results != nil {
		t.Error("Expected nil results when error occurs")
	}
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("Expected ErrProviderUnavailable, got %v", err)
	}
}

type errorLegacyProvider struct{}

func (e *errorLegacyProvider) Search(query string, maxResults int) ([]LegacySearchResult, error) {
	return nil, ErrNoResults
}

func (e *errorLegacyProvider) GetName() string {
	return "ErrorLegacyProvider"
}

func TestModernProviderAdapter_ErrorHandling(t *testing.T) {
	errorLegacyProvider := &errorLegacyProvider{}
	adapter := NewModernProviderAdapter(errorLegacyProvider)

	ctx := context.Background()
	config := Config{MaxResults: 10}

	results, err := adapter.Search(ctx, "test", config)
	if err == nil {
		t.Error("Expected error from adapter when underlying legacy provider fails")
	}
	if results != nil {
		t.Error("Expected nil results when error occurs")
	}
	if !errors.Is(err, ErrNoResults) {
		t.Errorf("Expected ErrNoResults, got %v", err)
	}
}
