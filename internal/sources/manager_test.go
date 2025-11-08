package sources

import (
	"briefly/internal/core"
	"briefly/internal/logger"
	"briefly/internal/persistence"
	"context"
	"errors"
	"testing"
	"time"
)

// Mock repositories

type MockManualURLRepo struct {
	urls              []core.ManualURL
	statusCalls       []StatusCall
	failGetPending    bool
	failUpdate        bool
	failMarkProcessed bool
}

type StatusCall struct {
	ID           string
	Status       string
	ErrorMessage string
}

func NewMockManualURLRepo() *MockManualURLRepo {
	return &MockManualURLRepo{
		urls:        []core.ManualURL{},
		statusCalls: []StatusCall{},
	}
}

func (m *MockManualURLRepo) GetPending(ctx context.Context, limit int) ([]core.ManualURL, error) {
	if m.failGetPending {
		return nil, errors.New("mock get pending error")
	}

	var pending []core.ManualURL
	count := 0
	for _, url := range m.urls {
		if url.Status == core.ManualURLStatusPending {
			pending = append(pending, url)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	return pending, nil
}

func (m *MockManualURLRepo) UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error {
	if m.failUpdate {
		return errors.New("mock update status error")
	}

	m.statusCalls = append(m.statusCalls, StatusCall{
		ID:           id,
		Status:       status,
		ErrorMessage: errorMessage,
	})

	// Update the actual URL status in the mock
	for i := range m.urls {
		if m.urls[i].ID == id {
			m.urls[i].Status = status
			if errorMessage != "" {
				m.urls[i].ErrorMessage = errorMessage
			}
			return nil
		}
	}
	return nil
}

func (m *MockManualURLRepo) MarkProcessed(ctx context.Context, id string) error {
	if m.failMarkProcessed {
		return errors.New("mock mark processed error")
	}

	now := time.Now()
	for i := range m.urls {
		if m.urls[i].ID == id {
			m.urls[i].Status = core.ManualURLStatusProcessed
			m.urls[i].ProcessedAt = &now
			return nil
		}
	}
	return nil
}

func (m *MockManualURLRepo) MarkFailed(ctx context.Context, id string, errorMessage string) error {
	for i := range m.urls {
		if m.urls[i].ID == id {
			m.urls[i].Status = core.ManualURLStatusFailed
			m.urls[i].ErrorMessage = errorMessage
			return nil
		}
	}
	return nil
}

// Stub methods (not used in tests but required by interface)
func (m *MockManualURLRepo) Create(ctx context.Context, manualURL *core.ManualURL) error {
	return nil
}
func (m *MockManualURLRepo) CreateBatch(ctx context.Context, urls []string, submittedBy string) error {
	return nil
}
func (m *MockManualURLRepo) Get(ctx context.Context, id string) (*core.ManualURL, error) {
	return nil, nil
}
func (m *MockManualURLRepo) List(ctx context.Context, opts persistence.ListOptions) ([]core.ManualURL, error) {
	return nil, nil
}
func (m *MockManualURLRepo) GetByURL(ctx context.Context, url string) (*core.ManualURL, error) {
	return nil, nil
}
func (m *MockManualURLRepo) GetByStatus(ctx context.Context, status string, limit int) ([]core.ManualURL, error) {
	return nil, nil
}
func (m *MockManualURLRepo) Delete(ctx context.Context, id string) error {
	return nil
}

type MockFeedItemRepo struct {
	items           []core.FeedItem
	failCreate      bool
	createCallCount int
	failOnCallNum   int // Fail on this specific call number (0 = disabled)
}

func NewMockFeedItemRepo() *MockFeedItemRepo {
	return &MockFeedItemRepo{
		items: []core.FeedItem{},
	}
}

func (m *MockFeedItemRepo) Create(ctx context.Context, item *core.FeedItem) error {
	m.createCallCount++

	if m.failCreate {
		return errors.New("mock feed item creation error")
	}

	// Selective failure on specific call
	if m.failOnCallNum > 0 && m.createCallCount == m.failOnCallNum {
		return errors.New("simulated failure on call " + string(rune(m.failOnCallNum+'0')))
	}

	m.items = append(m.items, *item)
	return nil
}

// Stub methods
func (m *MockFeedItemRepo) CreateBatch(ctx context.Context, items []core.FeedItem) error {
	return nil
}
func (m *MockFeedItemRepo) Get(ctx context.Context, id string) (*core.FeedItem, error) {
	return nil, nil
}
func (m *MockFeedItemRepo) GetByFeedID(ctx context.Context, feedID string, limit int) ([]core.FeedItem, error) {
	return nil, nil
}
func (m *MockFeedItemRepo) GetUnprocessed(ctx context.Context, limit int) ([]core.FeedItem, error) {
	return nil, nil
}
func (m *MockFeedItemRepo) List(ctx context.Context, opts persistence.ListOptions) ([]core.FeedItem, error) {
	return nil, nil
}
func (m *MockFeedItemRepo) MarkProcessed(ctx context.Context, id string) error {
	return nil
}
func (m *MockFeedItemRepo) Delete(ctx context.Context, id string) error {
	return nil
}

type MockDatabase struct {
	manualURLs *MockManualURLRepo
	feedItems  *MockFeedItemRepo
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		manualURLs: NewMockManualURLRepo(),
		feedItems:  NewMockFeedItemRepo(),
	}
}

func (m *MockDatabase) ManualURLs() persistence.ManualURLRepository {
	return m.manualURLs
}

func (m *MockDatabase) FeedItems() persistence.FeedItemRepository {
	return m.feedItems
}

// Stub methods
func (m *MockDatabase) Articles() persistence.ArticleRepository   { return nil }
func (m *MockDatabase) Summaries() persistence.SummaryRepository  { return nil }
func (m *MockDatabase) Feeds() persistence.FeedRepository         { return nil }
func (m *MockDatabase) Digests() persistence.DigestRepository     { return nil }
func (m *MockDatabase) Themes() persistence.ThemeRepository       { return nil }
func (m *MockDatabase) Citations() persistence.CitationRepository { return nil }
func (m *MockDatabase) Close() error                              { return nil }
func (m *MockDatabase) Ping(ctx context.Context) error            { return nil }
func (m *MockDatabase) BeginTx(ctx context.Context) (persistence.Transaction, error) {
	return nil, nil
}

// Helper to create test manual URLs
func createTestManualURL(id, url, submittedBy string) core.ManualURL {
	return core.ManualURL{
		ID:          id,
		URL:         url,
		SubmittedBy: submittedBy,
		Status:      core.ManualURLStatusPending,
		CreatedAt:   time.Now(),
	}
}

// Tests

func TestAggregateManualURLs_Success(t *testing.T) {
	// Setup
	mockDB := NewMockDatabase()
	mockDB.manualURLs.urls = []core.ManualURL{
		createTestManualURL("url-1", "https://example.com/article1", "user1"),
		createTestManualURL("url-2", "https://example.com/article2", "user1"),
	}

	manager := &Manager{
		db:  mockDB,
		log: logger.Get(),
	}

	// Execute
	ctx := context.Background()
	result, err := manager.AggregateManualURLs(ctx, 10)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.URLsProcessed != 2 {
		t.Errorf("Expected 2 URLs processed, got %d", result.URLsProcessed)
	}

	if result.URLsFailed != 0 {
		t.Errorf("Expected 0 URLs failed, got %d", result.URLsFailed)
	}

	// Verify feed items were created
	if len(mockDB.feedItems.items) != 2 {
		t.Errorf("Expected 2 feed items created, got %d", len(mockDB.feedItems.items))
	}

	// Verify feed item properties
	for _, item := range mockDB.feedItems.items {
		if item.FeedID != "manual" {
			t.Errorf("Expected feedID 'manual', got %s", item.FeedID)
		}
		if item.Processed != false {
			t.Error("Expected feed item to be unprocessed initially")
		}
	}

	// Verify status calls: 2 processing updates + 2 processed updates
	if len(mockDB.manualURLs.statusCalls) < 2 {
		t.Errorf("Expected at least 2 status calls, got %d", len(mockDB.manualURLs.statusCalls))
	}

	// Verify URLs were marked as processed
	for _, url := range mockDB.manualURLs.urls {
		if url.Status != core.ManualURLStatusProcessed {
			t.Errorf("Expected URL %s to be processed, got status: %s", url.ID, url.Status)
		}
	}
}

func TestAggregateManualURLs_NoPendingURLs(t *testing.T) {
	mockDB := NewMockDatabase()
	// No pending URLs

	manager := &Manager{
		db:  mockDB,
		log: logger.Get(),
	}

	ctx := context.Background()
	result, err := manager.AggregateManualURLs(ctx, 10)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.URLsProcessed != 0 {
		t.Errorf("Expected 0 URLs processed, got %d", result.URLsProcessed)
	}

	if result.URLsFailed != 0 {
		t.Errorf("Expected 0 URLs failed, got %d", result.URLsFailed)
	}
}

func TestAggregateManualURLs_GetPendingError(t *testing.T) {
	mockDB := NewMockDatabase()
	mockDB.manualURLs.failGetPending = true

	manager := &Manager{
		db:  mockDB,
		log: logger.Get(),
	}

	ctx := context.Background()
	_, err := manager.AggregateManualURLs(ctx, 10)

	if err == nil {
		t.Error("Expected error when GetPending fails")
	}

	if err != nil && !contains(err.Error(), "failed to get pending URLs") {
		t.Errorf("Expected 'failed to get pending URLs' error, got: %v", err)
	}
}

func TestAggregateManualURLs_FeedItemCreationError(t *testing.T) {
	mockDB := NewMockDatabase()
	mockDB.manualURLs.urls = []core.ManualURL{
		createTestManualURL("url-1", "https://example.com/article1", "user1"),
		createTestManualURL("url-2", "https://example.com/article2", "user1"),
	}
	mockDB.feedItems.failCreate = true // Fail feed item creation

	manager := &Manager{
		db:  mockDB,
		log: logger.Get(),
	}

	ctx := context.Background()
	result, err := manager.AggregateManualURLs(ctx, 10)

	// Should not return error (graceful degradation)
	if err != nil {
		t.Fatalf("Expected no error (graceful degradation), got: %v", err)
	}

	// All URLs should be marked as failed
	if result.URLsFailed != 2 {
		t.Errorf("Expected 2 URLs failed, got %d", result.URLsFailed)
	}

	if result.URLsProcessed != 0 {
		t.Errorf("Expected 0 URLs processed, got %d", result.URLsProcessed)
	}

	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors recorded, got %d", len(result.Errors))
	}

	// Verify URLs were marked as failed
	for _, url := range mockDB.manualURLs.urls {
		if url.Status != core.ManualURLStatusFailed {
			t.Errorf("Expected URL %s to be failed, got status: %s", url.ID, url.Status)
		}
	}
}

func TestAggregateManualURLs_MaxURLsLimit(t *testing.T) {
	mockDB := NewMockDatabase()
	mockDB.manualURLs.urls = []core.ManualURL{
		createTestManualURL("url-1", "https://example.com/article1", "user1"),
		createTestManualURL("url-2", "https://example.com/article2", "user1"),
		createTestManualURL("url-3", "https://example.com/article3", "user1"),
		createTestManualURL("url-4", "https://example.com/article4", "user1"),
		createTestManualURL("url-5", "https://example.com/article5", "user1"),
	}

	manager := &Manager{
		db:  mockDB,
		log: logger.Get(),
	}

	ctx := context.Background()
	result, err := manager.AggregateManualURLs(ctx, 3) // Limit to 3

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only process 3 URLs
	if result.URLsProcessed != 3 {
		t.Errorf("Expected 3 URLs processed (limit), got %d", result.URLsProcessed)
	}

	// Verify only 3 feed items created
	if len(mockDB.feedItems.items) != 3 {
		t.Errorf("Expected 3 feed items, got %d", len(mockDB.feedItems.items))
	}
}

func TestAggregateManualURLs_ContextCancellation(t *testing.T) {
	mockDB := NewMockDatabase()
	mockDB.manualURLs.urls = []core.ManualURL{
		createTestManualURL("url-1", "https://example.com/article1", "user1"),
		createTestManualURL("url-2", "https://example.com/article2", "user1"),
	}

	manager := &Manager{
		db:  mockDB,
		log: logger.Get(),
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := manager.AggregateManualURLs(ctx, 10)

	if err == nil {
		t.Error("Expected error when context is cancelled")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestAggregateManualURLs_StatusTransitions(t *testing.T) {
	mockDB := NewMockDatabase()
	mockDB.manualURLs.urls = []core.ManualURL{
		createTestManualURL("url-1", "https://example.com/article1", "user1"),
	}

	manager := &Manager{
		db:  mockDB,
		log: logger.Get(),
	}

	ctx := context.Background()
	_, err := manager.AggregateManualURLs(ctx, 10)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify status progression: pending -> processing -> processed
	statusCalls := mockDB.manualURLs.statusCalls

	if len(statusCalls) < 1 {
		t.Fatal("Expected at least 1 status call (processing)")
	}

	// First call should be to mark as processing
	if statusCalls[0].Status != string(core.ManualURLStatusProcessing) {
		t.Errorf("Expected first status to be 'processing', got: %s", statusCalls[0].Status)
	}

	// Final status should be processed
	finalURL := mockDB.manualURLs.urls[0]
	if finalURL.Status != core.ManualURLStatusProcessed {
		t.Errorf("Expected final status to be 'processed', got: %s", finalURL.Status)
	}

	if finalURL.ProcessedAt == nil {
		t.Error("Expected ProcessedAt to be set")
	}
}

func TestAggregateManualURLs_PartialSuccess(t *testing.T) {
	mockDB := NewMockDatabase()
	mockDB.manualURLs.urls = []core.ManualURL{
		createTestManualURL("url-1", "https://example.com/article1", "user1"),
		createTestManualURL("url-2", "https://example.com/article2", "user1"),
		createTestManualURL("url-3", "https://example.com/article3", "user1"),
	}

	// Make feed item creation fail for second URL only
	mockDB.feedItems.failOnCallNum = 2 // Fail on second call

	manager := &Manager{
		db:  mockDB,
		log: logger.Get(),
	}

	ctx := context.Background()
	result, err := manager.AggregateManualURLs(ctx, 10)

	if err != nil {
		t.Fatalf("Expected no error (partial success), got: %v", err)
	}

	// Should have 2 successes and 1 failure
	if result.URLsProcessed != 2 {
		t.Errorf("Expected 2 URLs processed, got %d", result.URLsProcessed)
	}

	if result.URLsFailed != 1 {
		t.Errorf("Expected 1 URL failed, got %d", result.URLsFailed)
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error recorded, got %d", len(result.Errors))
	}
}

func TestAggregateManualURLs_FeedItemProperties(t *testing.T) {
	mockDB := NewMockDatabase()
	testURL := createTestManualURL("url-1", "https://example.com/test-article", "alice")
	mockDB.manualURLs.urls = []core.ManualURL{testURL}

	manager := &Manager{
		db:  mockDB,
		log: logger.Get(),
	}

	ctx := context.Background()
	_, err := manager.AggregateManualURLs(ctx, 10)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify feed item was created with correct properties
	if len(mockDB.feedItems.items) != 1 {
		t.Fatalf("Expected 1 feed item, got %d", len(mockDB.feedItems.items))
	}

	item := mockDB.feedItems.items[0]

	if item.ID != "url-1" {
		t.Errorf("Expected ID 'url-1', got %s", item.ID)
	}

	if item.FeedID != "manual" {
		t.Errorf("Expected FeedID 'manual', got %s", item.FeedID)
	}

	if item.Link != "https://example.com/test-article" {
		t.Errorf("Expected link to match URL, got %s", item.Link)
	}

	if item.GUID != "https://example.com/test-article" {
		t.Errorf("Expected GUID to match URL, got %s", item.GUID)
	}

	if !contains(item.Description, "alice") {
		t.Errorf("Expected description to mention submitter 'alice', got: %s", item.Description)
	}

	if item.Processed != false {
		t.Error("Expected feed item to be unprocessed")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
