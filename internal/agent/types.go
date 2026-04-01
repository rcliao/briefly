package agent

import (
	"briefly/internal/core"
	"time"
)

// AgentSession holds configuration for one agentic digest generation run.
type AgentSession struct {
	ID               string
	InputFile        string
	OutputPath       string
	MaxIterations    int     // Max reflect/revise cycles (default: 3)
	QualityThreshold float64 // Min acceptable quality score 0-1 (default: 0.7)
	UseCache         bool    // Whether to use SQLite cache (default: true)
	OutputFormat     string  // "markdown" or "slack"
	StartedAt        time.Time
	CompletedAt      *time.Time
	Status           string // "running", "completed", "failed", "fallback"
	FallbackReason   string
	CurrentPhase     string // "ingestion", "analysis", "generation", "quality", "rendering"
	CurrentIteration int
}

// ArticleIndexEntry maps a stable citation number to an article ID.
type ArticleIndexEntry struct {
	CitationNum      int    `json:"citation_num"` // [1], [2], etc. — stable across all tools
	ArticleID        string `json:"article_id"`
	Title            string `json:"title"`
	URL              string `json:"url"`
	ReadMinutes      int    `json:"read_minutes"`
	ReaderIntent     string `json:"reader_intent"`     // "skim", "read", "deep_dive"
	TopicID          string `json:"topic_id"`          // Stable topic category ID
	EditorialSummary string `json:"editorial_summary"` // 1-2 sentence human-voice summary
}

// TriageScore holds per-article relevance/quality assessment from the triage tool.
type TriageScore struct {
	ArticleID         string  `json:"article_id"`
	Title             string  `json:"title"`
	RelevanceScore    float64 `json:"relevance_score"`
	QualityScore      float64 `json:"quality_score"`
	SignalStrength    float64 `json:"signal_strength"`
	Reasoning         string  `json:"reasoning"`
	RecommendedAction string  `json:"recommended_action"` // "include", "deprioritize", "exclude"
	ReaderIntent      string  `json:"reader_intent"`      // "skim", "read", "deep_dive"
	TopicID           string  `json:"topic_id"`           // Stable topic category ID
	EditorialSummary  string  `json:"editorial_summary"`  // 1-2 sentence human-voice summary
}

// ClusterEvaluation holds quality assessment for a topic cluster.
type ClusterEvaluation struct {
	ClusterID           string  `json:"cluster_id"`
	Label               string  `json:"label"`
	CoherenceScore      float64 `json:"coherence_score"`
	SeparationScore     float64 `json:"separation_score"`
	SizeAppropriateness string  `json:"size_appropriateness"` // "too_small", "appropriate", "too_large"
	SuggestedAction     string  `json:"suggested_action"`     // "keep", "merge_with:<id>", "split", "dissolve"
	Reasoning           string  `json:"reasoning"`
}

// DimensionScores holds quality scores across the five reflect dimensions.
type DimensionScores struct {
	Specificity float64 `json:"specificity"`
	Grounding   float64 `json:"grounding"`
	Coherence   float64 `json:"coherence"`
	ReaderValue float64 `json:"reader_value"`
	Coverage    float64 `json:"coverage"`
}

// WeightedAverage returns the overall quality score as a weighted average.
func (d DimensionScores) WeightedAverage() float64 {
	// Weight grounding and coverage higher since they're verifiable
	return (d.Specificity*0.15 + d.Grounding*0.25 + d.Coherence*0.15 + d.ReaderValue*0.20 + d.Coverage*0.25)
}

// Weakness represents a specific quality problem identified during reflection.
type Weakness struct {
	Section      string `json:"section"`      // "cluster_narrative:<id>", "executive_summary", "title", "tldr"
	Dimension    string `json:"dimension"`    // Which quality dimension is affected
	Description  string `json:"description"`  // What the problem is
	Severity     string `json:"severity"`     // "critical", "major", "minor"
	SuggestedFix string `json:"suggested_fix"`
}

// ReflectionReport holds quality assessment from a reflect tool call.
type ReflectionReport struct {
	Iteration        int             `json:"iteration"`
	Timestamp        time.Time       `json:"timestamp"`
	OverallScore     float64         `json:"overall_score"`
	Dimensions       DimensionScores `json:"dimension_scores"`
	Weaknesses       []Weakness      `json:"weaknesses"`
	Strengths        []string        `json:"strengths"`
	ImprovementDelta float64         `json:"improvement_delta"`
	ShouldContinue   bool            `json:"should_continue"`
}

// RevisionRecord tracks a revision applied to a specific section.
type RevisionRecord struct {
	Iteration         int       `json:"iteration"`
	Timestamp         time.Time `json:"timestamp"`
	TargetSection     string    `json:"target_section"`
	WeaknessAddressed string    `json:"weakness_addressed"`
	OriginalContent   string    `json:"original_content"`
	RevisedContent    string    `json:"revised_content"`
	ChangesMade       string    `json:"changes_made"`
}

// ToolCallRecord logs a single tool invocation.
type ToolCallRecord struct {
	SequenceNumber int            `json:"sequence_number"`
	ToolName       string         `json:"tool_name"`
	Parameters     map[string]any `json:"parameters"`
	StartedAt      time.Time      `json:"started_at"`
	CompletedAt    time.Time      `json:"completed_at"`
	DurationMs     int64          `json:"duration_ms"`
	Status         string         `json:"status"` // "success", "error", "timeout"
	ErrorMessage   string         `json:"error_message,omitempty"`
	ResultSummary  string         `json:"result_summary"`
}

// AgentDigestResult is the final output of agentic digest generation.
type AgentDigestResult struct {
	Digest        *core.Digest
	MarkdownPath  string
	AgentMetadata AgentMetadata
}

// AgentMetadata captures the agent's decision-making process.
type AgentMetadata struct {
	TotalIterations   int            `json:"total_iterations"`
	FinalQualityScore float64        `json:"final_quality_score"`
	QualityTrajectory []float64      `json:"quality_trajectory"`
	TotalToolCalls    int            `json:"total_tool_calls"`
	TotalLLMCalls     int            `json:"total_llm_calls"`
	ToolCallBreakdown map[string]int `json:"tool_call_breakdown"`
	StrategyUsed      string         `json:"strategy_used"`
	EarlyStopReason   string         `json:"early_stop_reason"`
	Warnings          []string       `json:"warnings"`
	TotalDurationMs   int64          `json:"total_duration_ms"`
}
