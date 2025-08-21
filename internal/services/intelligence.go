package services

import (
	"context"
	"time"

	"briefly/internal/core"
)

// IntelligenceService provides unified AI-powered content processing (v3.0)
type IntelligenceService interface {
	// Digest operations
	ProcessContent(ctx context.Context, input ContentInput) (*core.Digest, error)
	
	// Research operations  
	StartResearchSession(ctx context.Context, query string) (*core.ResearchSession, error)
	ContinueResearch(ctx context.Context, sessionID string, userInput string) (*ResearchResponse, error)
	
	// Exploration
	ExploreTopicFurther(ctx context.Context, topic string, currentDigest *core.Digest) (*core.ResearchSession, error)
	
	// Learning
	RecordUserFeedback(ctx context.Context, feedback core.UserFeedback) error
	GetPersonalizationProfile(ctx context.Context) (*core.UserProfile, error)
}

// ContentInput represents input for content processing
type ContentInput struct {
	URLs        []string            `json:"urls,omitempty"`
	FilePath    string              `json:"file_path,omitempty"`
	Options     ProcessingOptions   `json:"options"`
}

// ProcessingOptions controls how content is processed
type ProcessingOptions struct {
	ForceCloudAI     bool    `json:"force_cloud_ai"`     // Override local processing
	MaxWordCount     int     `json:"max_word_count"`     // Hard limit (default 300)
	QualityThreshold float64 `json:"quality_threshold"`  // Minimum article quality
	UserContext      string  `json:"user_context"`       // Additional context
}

// ResearchResponse represents response from research operations
type ResearchResponse struct {
	Message         string                 `json:"message"`
	DiscoveredItems []core.ResearchItem    `json:"discovered_items,omitempty"`
	Actions         []AvailableAction      `json:"actions"`
	ContinueSession bool                   `json:"continue_session"`
	ProcessingCost  core.ProcessingCost    `json:"processing_cost"`
}

// AvailableAction represents an action the user can take
type AvailableAction struct {
	ID          string `json:"id"`           // "1", "2", "3"
	Description string `json:"description"`  // "Technical details"
	Command     string `json:"command"`      // What gets executed
}

// AIRouter handles intelligent routing between local and cloud models (v3.0)
type AIRouter interface {
	RouteRequest(ctx context.Context, request AIRequest) (*AIResponse, error)
	EstimateCost(ctx context.Context, request AIRequest) (*CostEstimate, error)
	GetCapabilities(ctx context.Context) (*AICapabilities, error)
}

// AIRequest represents a request for AI processing
type AIRequest struct {
	Task        AITask            `json:"task"`
	Content     string            `json:"content"`
	Priority    Priority          `json:"priority"`    // low, medium, high
	MaxCost     *float64          `json:"max_cost"`    // USD limit
	UserPrefs   core.UserProfile  `json:"user_prefs"`
}

// AITask represents different types of AI tasks
type AITask string

const (
	TaskCategorize AITask = "categorize" // Local model
	TaskSummarize  AITask = "summarize"  // Local for simple, cloud for complex
	TaskSynthesize AITask = "synthesize" // Cloud model
	TaskGenerate   AITask = "generate"   // Cloud model
	TaskAnalyze    AITask = "analyze"    // Hybrid routing
)

// Priority represents task priority levels
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// AIResponse represents the response from AI processing
type AIResponse struct {
	Result        interface{}         `json:"result"`
	ProcessedBy   string             `json:"processed_by"`   // "local", "cloud"
	ActualCost    core.ProcessingCost `json:"actual_cost"`
	Quality       float64            `json:"quality"`        // 0.0-1.0
	Confidence    float64            `json:"confidence"`     // 0.0-1.0
}

// CostEstimate represents estimated processing costs
type CostEstimate struct {
	EstimatedCost  float64       `json:"estimated_cost"`
	LocalTokens    int           `json:"local_tokens"`
	CloudTokens    int           `json:"cloud_tokens"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// AICapabilities represents what the AI system can do
type AICapabilities struct {
	LocalModelsAvailable  bool     `json:"local_models_available"`
	CloudModelsAvailable  bool     `json:"cloud_models_available"`
	SupportedTasks        []AITask `json:"supported_tasks"`
	MaxLocalComplexity    float64  `json:"max_local_complexity"`  // 0.0-1.0
	CostPerToken          float64  `json:"cost_per_token"`
}

// LocalModelService handles local AI model operations (v3.0)
type LocalModelService interface {
	// Basic operations
	IsAvailable(ctx context.Context) (bool, error)
	Initialize(ctx context.Context) error
	
	// Processing operations
	CategorizeContent(ctx context.Context, content string) (string, float64, error)
	FilterByQuality(ctx context.Context, articles []core.Article, threshold float64) ([]core.Article, error)
	ClusterArticles(ctx context.Context, articles []core.Article) ([]core.ArticleGroup, error)
	
	// Utility operations
	AnalyzeComplexity(ctx context.Context, content string) (float64, error)  // 0.0-1.0
	GenerateBasicSummary(ctx context.Context, content string, maxWords int) (string, error)
}

// CostController manages AI processing costs (v3.0)
type CostController interface {
	// Budget management
	GetDailyBudget(ctx context.Context) (float64, error)
	GetSpentToday(ctx context.Context) (float64, error)
	RecordCost(ctx context.Context, cost core.ProcessingCost) error
	
	// Decision making
	ShouldUseCloud(ctx context.Context, task AITask, content string) (bool, error)
	GetPreferredModel(ctx context.Context, task AITask) (string, error)
	
	// Analytics
	GetCostAnalytics(ctx context.Context, days int) (*CostAnalytics, error)
}

// CostAnalytics represents cost analysis data
type CostAnalytics struct {
	TotalSpent       float64            `json:"total_spent"`
	LocalVsCloud     map[string]float64 `json:"local_vs_cloud"`  // local: 0.20, cloud: 2.30
	CostByTask       map[string]float64 `json:"cost_by_task"`    // summarize: 1.20, synthesize: 1.30
	DailyCosts       []DailyCost        `json:"daily_costs"`
	Recommendations  []string           `json:"recommendations"`
}

// DailyCost represents daily cost breakdown
type DailyCost struct {
	Date       time.Time `json:"date"`
	LocalCost  float64   `json:"local_cost"`
	CloudCost  float64   `json:"cloud_cost"`
	TaskCount  int       `json:"task_count"`
}