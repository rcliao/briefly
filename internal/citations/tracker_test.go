package citations

import (
	"briefly/internal/core"
	"briefly/internal/persistence"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// MockCitationRepo implements persistence.CitationRepository for testing
type MockCitationRepo struct {
	citations      map[string]*core.Citation
	shouldFail     bool
	shouldFailOnce bool
	getByIDFail    bool
}

func NewMockCitationRepo() *MockCitationRepo {
	return &MockCitationRepo{
		citations: make(map[string]*core.Citation),
	}
}

func (m *MockCitationRepo) Create(ctx context.Context, citation *core.Citation) error {
	if m.shouldFail || m.shouldFailOnce {
		if m.shouldFailOnce {
			m.shouldFailOnce = false
		}
		return errors.New("mock create failed")
	}
	m.citations[citation.ID] = citation
	return nil
}

func (m *MockCitationRepo) Get(ctx context.Context, id string) (*core.Citation, error) {
	if m.getByIDFail {
		return nil, errors.New("mock get failed")
	}
	citation, exists := m.citations[id]
	if !exists {
		return nil, errors.New("citation not found")
	}
	return citation, nil
}

func (m *MockCitationRepo) GetByArticleID(ctx context.Context, articleID string) (*core.Citation, error) {
	if m.shouldFail {
		return nil, errors.New("mock get by article ID failed")
	}
	for _, citation := range m.citations {
		if citation.ArticleID == articleID {
			return citation, nil
		}
	}
	return nil, errors.New("citation not found")
}

func (m *MockCitationRepo) GetByArticleIDs(ctx context.Context, articleIDs []string) (map[string]*core.Citation, error) {
	result := make(map[string]*core.Citation)
	for _, articleID := range articleIDs {
		for _, citation := range m.citations {
			if citation.ArticleID == articleID {
				result[articleID] = citation
			}
		}
	}
	return result, nil
}

func (m *MockCitationRepo) List(ctx context.Context, opts persistence.ListOptions) ([]core.Citation, error) {
	result := make([]core.Citation, 0, len(m.citations))
	for _, citation := range m.citations {
		result = append(result, *citation)
	}
	return result, nil
}

func (m *MockCitationRepo) Update(ctx context.Context, citation *core.Citation) error {
	if m.shouldFail {
		return errors.New("mock update failed")
	}
	m.citations[citation.ID] = citation
	return nil
}

func (m *MockCitationRepo) Delete(ctx context.Context, id string) error {
	delete(m.citations, id)
	return nil
}

func (m *MockCitationRepo) DeleteByArticleID(ctx context.Context, articleID string) error {
	for id, citation := range m.citations {
		if citation.ArticleID == articleID {
			delete(m.citations, id)
		}
	}
	return nil
}

// v2.0 methods
func (m *MockCitationRepo) CreateBatch(ctx context.Context, citations []core.Citation) error {
	for i := range citations {
		if err := m.Create(ctx, &citations[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockCitationRepo) GetByDigestID(ctx context.Context, digestID string) ([]core.Citation, error) {
	result := make([]core.Citation, 0)
	for _, citation := range m.citations {
		if citation.DigestID != nil && *citation.DigestID == digestID {
			result = append(result, *citation)
		}
	}
	return result, nil
}

func (m *MockCitationRepo) DeleteByDigestID(ctx context.Context, digestID string) error {
	for id, citation := range m.citations {
		if citation.DigestID != nil && *citation.DigestID == digestID {
			delete(m.citations, id)
		}
	}
	return nil
}

// MockDatabase implements persistence.Database for testing
type MockDatabase struct {
	citationRepo *MockCitationRepo
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		citationRepo: NewMockCitationRepo(),
	}
}

func (m *MockDatabase) Citations() persistence.CitationRepository {
	return m.citationRepo
}

func (m *MockDatabase) Articles() persistence.ArticleRepository                   { return nil }
func (m *MockDatabase) Summaries() persistence.SummaryRepository                   { return nil }
func (m *MockDatabase) Feeds() persistence.FeedRepository                          { return nil }
func (m *MockDatabase) FeedItems() persistence.FeedItemRepository                  { return nil }
func (m *MockDatabase) Digests() persistence.DigestRepository                      { return nil }
func (m *MockDatabase) Themes() persistence.ThemeRepository                        { return nil }
func (m *MockDatabase) ManualURLs() persistence.ManualURLRepository                { return nil }
func (m *MockDatabase) Tags() persistence.TagRepository                            { return nil }
func (m *MockDatabase) ClusterCoherence() persistence.ClusterCoherenceRepository   { return nil }
func (m *MockDatabase) Close() error                                               { return nil }
func (m *MockDatabase) Ping(ctx context.Context) error                             { return nil }
func (m *MockDatabase) BeginTx(ctx context.Context) (persistence.Transaction, error) {
	return nil, nil
}

// Test TrackArticle - basic functionality
func TestTrackArticle_Success(t *testing.T) {
	db := NewMockDatabase()
	tracker := NewTracker(db)
	ctx := context.Background()

	article := &core.Article{
		ID:          uuid.NewString(),
		URL:         "https://example.com/article",
		Title:       "Test Article",
		DateFetched: time.Now().UTC(),
	}

	citation, err := tracker.TrackArticle(ctx, article)
	if err != nil {
		t.Fatalf("TrackArticle failed: %v", err)
	}

	if citation == nil {
		t.Fatal("Expected citation, got nil")
	}

	if citation.ArticleID != article.ID {
		t.Errorf("Expected ArticleID %s, got %s", article.ID, citation.ArticleID)
	}

	if citation.URL != article.URL {
		t.Errorf("Expected URL %s, got %s", article.URL, citation.URL)
	}

	if citation.Title != article.Title {
		t.Errorf("Expected Title %s, got %s", article.Title, citation.Title)
	}

	if citation.Publisher != "example.com" {
		t.Errorf("Expected Publisher 'example.com', got '%s'", citation.Publisher)
	}
}

// Test TrackArticle - idempotency (existing citation)
func TestTrackArticle_Idempotency(t *testing.T) {
	db := NewMockDatabase()
	tracker := NewTracker(db)
	ctx := context.Background()

	article := &core.Article{
		ID:          uuid.NewString(),
		URL:         "https://example.com/article",
		Title:       "Test Article",
		DateFetched: time.Now().UTC(),
	}

	// Track first time
	citation1, err := tracker.TrackArticle(ctx, article)
	if err != nil {
		t.Fatalf("First TrackArticle failed: %v", err)
	}

	// Track second time - should return existing citation
	citation2, err := tracker.TrackArticle(ctx, article)
	if err != nil {
		t.Fatalf("Second TrackArticle failed: %v", err)
	}

	if citation1.ID != citation2.ID {
		t.Errorf("Expected same citation ID, got %s and %s", citation1.ID, citation2.ID)
	}

	// Verify only one citation exists in the database
	citations, _ := db.Citations().List(ctx, persistence.ListOptions{})
	if len(citations) != 1 {
		t.Errorf("Expected 1 citation in database, got %d", len(citations))
	}
}

// Test TrackArticle - nil article
func TestTrackArticle_NilArticle(t *testing.T) {
	db := NewMockDatabase()
	tracker := NewTracker(db)
	ctx := context.Background()

	_, err := tracker.TrackArticle(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil article, got nil")
	}
}

// Test TrackArticle - database error
func TestTrackArticle_DatabaseError(t *testing.T) {
	db := NewMockDatabase()
	db.citationRepo.shouldFail = true
	tracker := NewTracker(db)
	ctx := context.Background()

	article := &core.Article{
		ID:          uuid.NewString(),
		URL:         "https://example.com/article",
		Title:       "Test Article",
		DateFetched: time.Now().UTC(),
	}

	// First call to GetByArticleID will fail, then Create will fail
	_, err := tracker.TrackArticle(ctx, article)
	if err == nil {
		t.Fatal("Expected database error, got nil")
	}
}

// Test TrackBatch - successful batch
func TestTrackBatch_Success(t *testing.T) {
	db := NewMockDatabase()
	tracker := NewTracker(db)
	ctx := context.Background()

	articles := []core.Article{
		{
			ID:          uuid.NewString(),
			URL:         "https://example.com/article1",
			Title:       "Article 1",
			DateFetched: time.Now().UTC(),
		},
		{
			ID:          uuid.NewString(),
			URL:         "https://example.com/article2",
			Title:       "Article 2",
			DateFetched: time.Now().UTC(),
		},
		{
			ID:          uuid.NewString(),
			URL:         "https://example.com/article3",
			Title:       "Article 3",
			DateFetched: time.Now().UTC(),
		},
	}

	citations, err := tracker.TrackBatch(ctx, articles)
	if err != nil {
		t.Fatalf("TrackBatch failed: %v", err)
	}

	if len(citations) != 3 {
		t.Errorf("Expected 3 citations, got %d", len(citations))
	}

	// Verify all articles have citations
	for _, article := range articles {
		if _, exists := citations[article.ID]; !exists {
			t.Errorf("Missing citation for article %s", article.ID)
		}
	}
}

// Test TrackBatch - partial failure
func TestTrackBatch_PartialFailure(t *testing.T) {
	db := NewMockDatabase()
	db.citationRepo.shouldFailOnce = true // Fail on first article
	tracker := NewTracker(db)
	ctx := context.Background()

	articles := []core.Article{
		{
			ID:          uuid.NewString(),
			URL:         "https://example.com/article1",
			Title:       "Article 1",
			DateFetched: time.Now().UTC(),
		},
		{
			ID:          uuid.NewString(),
			URL:         "https://example.com/article2",
			Title:       "Article 2",
			DateFetched: time.Now().UTC(),
		},
	}

	citations, err := tracker.TrackBatch(ctx, articles)

	// Should return error because not all articles were tracked
	if err == nil {
		t.Fatal("Expected error for partial failure, got nil")
	}

	// But should still return partial results
	if len(citations) != 1 {
		t.Errorf("Expected 1 successful citation, got %d", len(citations))
	}
}

// Test GetCitation - retrieval
func TestGetCitation_Success(t *testing.T) {
	db := NewMockDatabase()
	tracker := NewTracker(db)
	ctx := context.Background()

	article := &core.Article{
		ID:          uuid.NewString(),
		URL:         "https://example.com/article",
		Title:       "Test Article",
		DateFetched: time.Now().UTC(),
	}

	// Create citation
	created, err := tracker.TrackArticle(ctx, article)
	if err != nil {
		t.Fatalf("TrackArticle failed: %v", err)
	}

	// Retrieve citation
	retrieved, err := tracker.GetCitation(ctx, article.ID)
	if err != nil {
		t.Fatalf("GetCitation failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, retrieved.ID)
	}
}

// Test extractPublisher - various URL formats
func TestExtractPublisher(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Simple domain",
			url:      "https://example.com/article",
			expected: "example.com",
		},
		{
			name:     "With www prefix",
			url:      "https://www.example.com/article",
			expected: "example.com",
		},
		{
			name:     "Subdomain",
			url:      "https://blog.example.com/article",
			expected: "example.com",
		},
		{
			name:     "Complex subdomain",
			url:      "https://api.staging.example.com/article",
			expected: "example.com",
		},
		{
			name:     "Path and query",
			url:      "https://example.com/path/to/article?utm_source=test",
			expected: "example.com",
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPublisher(tt.url)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Test FormatCitation - Simple format
func TestFormatCitation_Simple(t *testing.T) {
	publishedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	citation := &core.Citation{
		ID:            uuid.NewString(),
		ArticleID:     uuid.NewString(),
		URL:           "https://example.com/article",
		Title:         "Test Article",
		Publisher:     "example.com",
		Author:        "John Doe",
		PublishedDate: &publishedDate,
		AccessedDate:  time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	formatted := FormatCitation(citation, "simple")

	// Should contain all key elements
	if !contains(formatted, "John Doe") {
		t.Error("Simple citation should contain author")
	}
	if !contains(formatted, "Test Article") {
		t.Error("Simple citation should contain title")
	}
	if !contains(formatted, "example.com") {
		t.Error("Simple citation should contain publisher")
	}
	if !contains(formatted, "https://example.com/article") {
		t.Error("Simple citation should contain URL")
	}
	if !contains(formatted, "2024-01-15") {
		t.Error("Simple citation should contain published date")
	}
	if !contains(formatted, "accessed 2024-02-01") {
		t.Error("Simple citation should contain accessed date")
	}
}

// Test FormatCitation - APA format
func TestFormatCitation_APA(t *testing.T) {
	publishedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	citation := &core.Citation{
		ID:            uuid.NewString(),
		ArticleID:     uuid.NewString(),
		URL:           "https://example.com/article",
		Title:         "Test Article",
		Publisher:     "example.com",
		Author:        "Doe, J.",
		PublishedDate: &publishedDate,
	}

	formatted := FormatCitation(citation, "apa")

	// APA format: Author (Year). Title. Publisher. URL
	if !contains(formatted, "Doe, J. (2024)") {
		t.Errorf("APA citation should contain 'Doe, J. (2024)', got: %s", formatted)
	}
	if !contains(formatted, "Test Article") {
		t.Error("APA citation should contain title")
	}
}

// Test FormatCitation - MLA format
func TestFormatCitation_MLA(t *testing.T) {
	publishedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	citation := &core.Citation{
		ID:            uuid.NewString(),
		ArticleID:     uuid.NewString(),
		URL:           "https://example.com/article",
		Title:         "Test Article",
		Publisher:     "Example Publisher",
		Author:        "Doe, John",
		PublishedDate: &publishedDate,
	}

	formatted := FormatCitation(citation, "mla")

	// MLA format: Author. "Title." Publisher, Date. URL.
	if !contains(formatted, "Doe, John") {
		t.Error("MLA citation should contain author")
	}
	if !contains(formatted, "\"Test Article.\"") {
		t.Errorf("MLA citation should contain quoted title, got: %s", formatted)
	}
}

// Test FormatCitation - Chicago format
func TestFormatCitation_Chicago(t *testing.T) {
	publishedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	citation := &core.Citation{
		ID:            uuid.NewString(),
		ArticleID:     uuid.NewString(),
		URL:           "https://example.com/article",
		Title:         "Test Article",
		Publisher:     "Example Publisher",
		Author:        "Doe, John",
		PublishedDate: &publishedDate,
	}

	formatted := FormatCitation(citation, "chicago")

	// Chicago format: Author. "Title." Publisher. Date. URL.
	if !contains(formatted, "Doe, John") {
		t.Error("Chicago citation should contain author")
	}
	if !contains(formatted, "\"Test Article.\"") {
		t.Error("Chicago citation should contain quoted title")
	}
	if !contains(formatted, "January 15, 2024") {
		t.Errorf("Chicago citation should contain formatted date, got: %s", formatted)
	}
}

// Test EnrichWithMetadata
func TestEnrichWithMetadata(t *testing.T) {
	db := NewMockDatabase()
	tracker := NewTracker(db)

	citation := &core.Citation{
		ID:        uuid.NewString(),
		ArticleID: uuid.NewString(),
		Metadata:  make(map[string]interface{}),
	}

	metadata := map[string]interface{}{
		"doi":      "10.1234/test",
		"keywords": []string{"test", "citation"},
		"abstract": "Test abstract",
	}

	tracker.EnrichWithMetadata(citation, metadata)

	if citation.Metadata["doi"] != "10.1234/test" {
		t.Error("Metadata should contain DOI")
	}

	if citation.Metadata["abstract"] != "Test abstract" {
		t.Error("Metadata should contain abstract")
	}

	if len(citation.Metadata) != 3 {
		t.Errorf("Expected 3 metadata fields, got %d", len(citation.Metadata))
	}
}

// Test EnrichWithMetadata - nil metadata
func TestEnrichWithMetadata_NilMetadata(t *testing.T) {
	db := NewMockDatabase()
	tracker := NewTracker(db)

	citation := &core.Citation{
		ID:        uuid.NewString(),
		ArticleID: uuid.NewString(),
		Metadata:  nil, // Nil metadata
	}

	metadata := map[string]interface{}{
		"test": "value",
	}

	// Should initialize metadata if nil
	tracker.EnrichWithMetadata(citation, metadata)

	if citation.Metadata == nil {
		t.Fatal("Metadata should be initialized")
	}

	if citation.Metadata["test"] != "value" {
		t.Error("Metadata should contain test value")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)+1 && anySubstring(s[1:len(s)-1], substr)))
}

func anySubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
