package agent

import (
	"briefly/internal/core"
	"fmt"
	"sync"
)

// WorkingMemory is the in-memory state store for an agent session.
// All tools read from and write to this store.
type WorkingMemory struct {
	mu sync.RWMutex

	sessionID string

	// Stable article index — built once after fetch, used by all tools for citation [N]
	articleIndex []ArticleIndexEntry     // ordered: index position = citation number - 1
	idToNum      map[string]int          // article ID -> citation number (1-based)

	// Ingestion state
	articles     map[string]core.Article
	summaries    map[string]core.Summary
	triageScores map[string]TriageScore

	// Analysis state
	embeddings map[string][]float64
	clusters   []core.TopicCluster
	clusterEvals []ClusterEvaluation

	// Generation state
	narratives       map[string]core.ClusterNarrative
	executiveSummary string
	digestDraft      *core.Digest
	digestContent    map[string]string // section name -> content for revisions

	// Quality state
	reflections []ReflectionReport
	revisionLog []RevisionRecord

	// Tracking
	toolCallLog       []ToolCallRecord
	qualityTrajectory []float64
}

// NewWorkingMemory creates an empty working memory for a session.
func NewWorkingMemory(sessionID string) *WorkingMemory {
	return &WorkingMemory{
		sessionID:     sessionID,
		idToNum:       make(map[string]int),
		articles:      make(map[string]core.Article),
		summaries:     make(map[string]core.Summary),
		triageScores:  make(map[string]TriageScore),
		embeddings:    make(map[string][]float64),
		narratives:    make(map[string]core.ClusterNarrative),
		digestContent: make(map[string]string),
	}
}

// --- Articles ---

func (wm *WorkingMemory) SetArticles(articles map[string]core.Article) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.articles = articles
}

func (wm *WorkingMemory) GetArticles() map[string]core.Article {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make(map[string]core.Article, len(wm.articles))
	for k, v := range wm.articles {
		cp[k] = v
	}
	return cp
}

func (wm *WorkingMemory) AddArticle(id string, article core.Article) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.articles[id] = article
}

// --- Article Index ---

// BuildArticleIndex creates a stable, ordered citation index from the current articles.
// Must be called once after all articles are fetched. Citation numbers are 1-based.
// Articles are sorted by URL for deterministic ordering across runs.
func (wm *WorkingMemory) BuildArticleIndex() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Collect and sort by URL for deterministic order
	type kv struct {
		id      string
		article core.Article
	}
	var sorted []kv
	for id, a := range wm.articles {
		sorted = append(sorted, kv{id, a})
	}
	// Sort by URL for stable ordering
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].article.URL > sorted[j].article.URL {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	wm.articleIndex = make([]ArticleIndexEntry, 0, len(sorted))
	wm.idToNum = make(map[string]int, len(sorted))
	for i, item := range sorted {
		num := i + 1
		readMin := sanitizeReadTime(item.article.EstimatedReadMinutes)
		wm.articleIndex = append(wm.articleIndex, ArticleIndexEntry{
			CitationNum:  num,
			ArticleID:    item.id,
			Title:        item.article.Title,
			URL:          item.article.URL,
			ReadMinutes:  readMin,
		})
		wm.idToNum[item.id] = num
	}
}

// GetArticleIndex returns the stable article index.
func (wm *WorkingMemory) GetArticleIndex() []ArticleIndexEntry {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make([]ArticleIndexEntry, len(wm.articleIndex))
	copy(cp, wm.articleIndex)
	return cp
}

// GetCitationNum returns the stable citation number for an article ID. Returns 0 if not found.
func (wm *WorkingMemory) GetCitationNum(articleID string) int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.idToNum[articleID]
}

// SetReaderIntent updates the reader intent for an article in the index.
func (wm *WorkingMemory) SetReaderIntent(articleID string, intent string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	for i := range wm.articleIndex {
		if wm.articleIndex[i].ArticleID == articleID {
			wm.articleIndex[i].ReaderIntent = intent
			break
		}
	}
}

// SetArticleTopic updates the topic category for an article in the index.
func (wm *WorkingMemory) SetArticleTopic(articleID string, topicID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	for i := range wm.articleIndex {
		if wm.articleIndex[i].ArticleID == articleID {
			wm.articleIndex[i].TopicID = topicID
			break
		}
	}
}

// SetReadMinutes overrides the read time for an article with the LLM estimate.
func (wm *WorkingMemory) SetReadMinutes(articleID string, minutes int) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	for i := range wm.articleIndex {
		if wm.articleIndex[i].ArticleID == articleID {
			wm.articleIndex[i].ReadMinutes = minutes
			break
		}
	}
}

// SetEditorialSummary updates the editorial summary for an article in the index.
func (wm *WorkingMemory) SetEditorialSummary(articleID string, summary string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	for i := range wm.articleIndex {
		if wm.articleIndex[i].ArticleID == articleID {
			wm.articleIndex[i].EditorialSummary = summary
			break
		}
	}
}

// FormatArticleList returns a formatted string of all articles with their stable citation numbers.
// Use this in all prompts instead of iterating the articles map directly.
func (wm *WorkingMemory) FormatArticleList() string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	var sb string
	for _, entry := range wm.articleIndex {
		sb += fmt.Sprintf("[%d] %s (%s)\n", entry.CitationNum, entry.Title, entry.URL)
	}
	return sb
}

// --- Summaries ---

func (wm *WorkingMemory) SetSummaries(summaries map[string]core.Summary) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.summaries = summaries
}

func (wm *WorkingMemory) GetSummaries() map[string]core.Summary {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make(map[string]core.Summary, len(wm.summaries))
	for k, v := range wm.summaries {
		cp[k] = v
	}
	return cp
}

func (wm *WorkingMemory) AddSummary(articleID string, summary core.Summary) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.summaries[articleID] = summary
}

// --- Triage Scores ---

func (wm *WorkingMemory) SetTriageScores(scores map[string]TriageScore) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.triageScores = scores
}

func (wm *WorkingMemory) GetTriageScores() map[string]TriageScore {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make(map[string]TriageScore, len(wm.triageScores))
	for k, v := range wm.triageScores {
		cp[k] = v
	}
	return cp
}

// --- Embeddings ---

func (wm *WorkingMemory) SetEmbeddings(embeddings map[string][]float64) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.embeddings = embeddings
}

func (wm *WorkingMemory) GetEmbeddings() map[string][]float64 {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make(map[string][]float64, len(wm.embeddings))
	for k, v := range wm.embeddings {
		vec := make([]float64, len(v))
		copy(vec, v)
		cp[k] = vec
	}
	return cp
}

func (wm *WorkingMemory) AddEmbedding(articleID string, embedding []float64) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.embeddings[articleID] = embedding
}

// --- Clusters ---

func (wm *WorkingMemory) SetClusters(clusters []core.TopicCluster) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.clusters = clusters
}

func (wm *WorkingMemory) GetClusters() []core.TopicCluster {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make([]core.TopicCluster, len(wm.clusters))
	copy(cp, wm.clusters)
	return cp
}

// --- Cluster Evaluations ---

func (wm *WorkingMemory) SetClusterEvaluations(evals []ClusterEvaluation) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.clusterEvals = evals
}

func (wm *WorkingMemory) GetClusterEvaluations() []ClusterEvaluation {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make([]ClusterEvaluation, len(wm.clusterEvals))
	copy(cp, wm.clusterEvals)
	return cp
}

// --- Narratives ---

func (wm *WorkingMemory) SetNarrative(clusterID string, narrative core.ClusterNarrative) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.narratives[clusterID] = narrative
}

func (wm *WorkingMemory) GetNarratives() map[string]core.ClusterNarrative {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make(map[string]core.ClusterNarrative, len(wm.narratives))
	for k, v := range wm.narratives {
		cp[k] = v
	}
	return cp
}

// --- Executive Summary ---

func (wm *WorkingMemory) SetExecutiveSummary(summary string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.executiveSummary = summary
}

func (wm *WorkingMemory) GetExecutiveSummary() string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.executiveSummary
}

// --- Digest Draft ---

func (wm *WorkingMemory) SetDigestDraft(digest *core.Digest) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.digestDraft = digest
}

func (wm *WorkingMemory) GetDigestDraft() *core.Digest {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.digestDraft
}

// --- Digest Content Sections (for reflect/revise) ---

func (wm *WorkingMemory) SetDigestSection(section string, content string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.digestContent[section] = content
}

func (wm *WorkingMemory) GetDigestSection(section string) (string, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	c, ok := wm.digestContent[section]
	return c, ok
}

func (wm *WorkingMemory) GetAllDigestSections() map[string]string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make(map[string]string, len(wm.digestContent))
	for k, v := range wm.digestContent {
		cp[k] = v
	}
	return cp
}

// --- Reflections ---

func (wm *WorkingMemory) AddReflection(report ReflectionReport) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.reflections = append(wm.reflections, report)
	wm.qualityTrajectory = append(wm.qualityTrajectory, report.OverallScore)
}

func (wm *WorkingMemory) GetReflections() []ReflectionReport {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make([]ReflectionReport, len(wm.reflections))
	copy(cp, wm.reflections)
	return cp
}

func (wm *WorkingMemory) GetQualityTrajectory() []float64 {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make([]float64, len(wm.qualityTrajectory))
	copy(cp, wm.qualityTrajectory)
	return cp
}

func (wm *WorkingMemory) GetLatestReflection() *ReflectionReport {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	if len(wm.reflections) == 0 {
		return nil
	}
	r := wm.reflections[len(wm.reflections)-1]
	return &r
}

// --- Revisions ---

func (wm *WorkingMemory) AddRevision(record RevisionRecord) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.revisionLog = append(wm.revisionLog, record)
}

func (wm *WorkingMemory) GetRevisions() []RevisionRecord {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make([]RevisionRecord, len(wm.revisionLog))
	copy(cp, wm.revisionLog)
	return cp
}

// --- Tool Call Log ---

func (wm *WorkingMemory) LogToolCall(record ToolCallRecord) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	record.SequenceNumber = len(wm.toolCallLog) + 1
	wm.toolCallLog = append(wm.toolCallLog, record)
}

func (wm *WorkingMemory) GetToolCallLog() []ToolCallRecord {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	cp := make([]ToolCallRecord, len(wm.toolCallLog))
	copy(cp, wm.toolCallLog)
	return cp
}

// --- Snapshot ---

// Snapshot returns a human-readable summary of current state for agent context.
func (wm *WorkingMemory) Snapshot() string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var latestScore string
	if len(wm.qualityTrajectory) > 0 {
		latestScore = fmt.Sprintf("%.2f", wm.qualityTrajectory[len(wm.qualityTrajectory)-1])
	} else {
		latestScore = "not yet evaluated"
	}

	return fmt.Sprintf(
		"State: %d articles, %d summaries, %d embeddings, %d clusters, %d narratives, %d reflections. Quality: %s",
		len(wm.articles),
		len(wm.summaries),
		len(wm.embeddings),
		len(wm.clusters),
		len(wm.narratives),
		len(wm.reflections),
		latestScore,
	)
}

// sanitizeReadTime clamps reading time to reasonable bounds.
// The fetcher's word count can be inflated by nav/footer HTML, so we cap at 30 min
// for most content. Landing pages with very little text get 0 (omitted from display).
func sanitizeReadTime(minutes int) int {
	if minutes <= 0 {
		return 0
	}
	if minutes > 30 {
		return 30 // Cap at 30 min — anything longer is likely inflated
	}
	return minutes
}
