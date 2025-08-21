package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"briefly/internal/core"
)

// costController implements CostController interface
type costController struct {
	configPath   string
	dailyBudget  float64
	costHistory  []DailyCost
}

// CostConfig represents stored cost configuration
type CostConfig struct {
	DailyBudget float64     `json:"daily_budget"`
	History     []DailyCost `json:"history"`
	LastUpdated time.Time   `json:"last_updated"`
}

// NewCostController creates a new cost controller
func NewCostController(dataDir string, dailyBudget float64) CostController {
	if dailyBudget <= 0 {
		dailyBudget = 1.0 // Default $1/day budget
	}

	configPath := filepath.Join(dataDir, "cost_config.json")

	controller := &costController{
		configPath:  configPath,
		dailyBudget: dailyBudget,
		costHistory: make([]DailyCost, 0),
	}

	// Load existing configuration
	controller.loadConfig()

	return controller
}

// GetDailyBudget returns the current daily budget
func (c *costController) GetDailyBudget(ctx context.Context) (float64, error) {
	return c.dailyBudget, nil
}

// GetSpentToday returns total spent today
func (c *costController) GetSpentToday(ctx context.Context) (float64, error) {
	today := time.Now().Format("2006-01-02")
	
	for _, daily := range c.costHistory {
		if daily.Date.Format("2006-01-02") == today {
			return daily.LocalCost + daily.CloudCost, nil
		}
	}

	return 0.0, nil
}

// RecordCost records a new cost entry
func (c *costController) RecordCost(ctx context.Context, cost core.ProcessingCost) error {
	today := time.Now()
	todayStr := today.Format("2006-01-02")

	// Find or create today's entry
	var todayEntry *DailyCost
	for i := range c.costHistory {
		if c.costHistory[i].Date.Format("2006-01-02") == todayStr {
			todayEntry = &c.costHistory[i]
			break
		}
	}

	if todayEntry == nil {
		// Create new entry
		newEntry := DailyCost{
			Date:       today,
			LocalCost:  0,
			CloudCost:  0,
			TaskCount:  0,
		}
		c.costHistory = append(c.costHistory, newEntry)
		todayEntry = &c.costHistory[len(c.costHistory)-1]
	}

	// Update costs
	if cost.LocalTokens > 0 {
		todayEntry.LocalCost += cost.EstimatedUSD * 0.1 // Local models are ~10% of cloud cost
	}
	if cost.CloudTokens > 0 {
		todayEntry.CloudCost += cost.EstimatedUSD
	}
	todayEntry.TaskCount++

	// Save configuration
	return c.saveConfig()
}

// ShouldUseCloud determines if cloud models should be used based on budget
func (c *costController) ShouldUseCloud(ctx context.Context, task AITask, content string) (bool, error) {
	spentToday, err := c.GetSpentToday(ctx)
	if err != nil {
		return false, err
	}

	// Estimate cost of cloud processing
	estimatedCloudCost := c.estimateCloudCost(task, content)

	// Check if we would exceed budget
	if spentToday+estimatedCloudCost > c.dailyBudget {
		return false, nil // Use local to stay within budget
	}

	// Check task complexity
	switch task {
	case TaskCategorize:
		return false, nil // Always use local for simple tasks

	case TaskAnalyze:
		// Use cloud for complex analysis
		return len(content) > 1000, nil

	case TaskSummarize:
		// Balance between cost and quality
		return spentToday < c.dailyBudget*0.5, nil // Use cloud if we've spent less than 50% of budget

	case TaskSynthesize, TaskGenerate:
		return true, nil // Always use cloud for complex tasks

	default:
		return false, nil // Default to local
	}
}

// GetPreferredModel returns preferred model for a task
func (c *costController) GetPreferredModel(ctx context.Context, task AITask) (string, error) {
	shouldUseCloud, err := c.ShouldUseCloud(ctx, task, "")
	if err != nil {
		return "", err
	}

	if shouldUseCloud {
		return "gemini-2.5-flash-preview-05-20", nil // Cloud model
	}

	return "llama3.2:3b", nil // Local model
}

// GetCostAnalytics provides cost analysis over specified days
func (c *costController) GetCostAnalytics(ctx context.Context, days int) (*CostAnalytics, error) {
	if days <= 0 {
		days = 7 // Default to last 7 days
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var relevantDays []DailyCost

	totalSpent := 0.0
	localTotal := 0.0
	cloudTotal := 0.0
	totalTasks := 0

	for _, daily := range c.costHistory {
		if daily.Date.After(cutoff) {
			relevantDays = append(relevantDays, daily)
			totalSpent += daily.LocalCost + daily.CloudCost
			localTotal += daily.LocalCost
			cloudTotal += daily.CloudCost
			totalTasks += daily.TaskCount
		}
	}

	// Calculate cost by task (simplified for Phase 3)
	costByTask := map[string]float64{
		"categorize": localTotal * 0.3,
		"summarize":  (localTotal * 0.4) + (cloudTotal * 0.3),
		"analyze":    (localTotal * 0.2) + (cloudTotal * 0.2),
		"synthesize": cloudTotal * 0.3,
		"generate":   cloudTotal * 0.2,
	}

	// Generate recommendations
	var recommendations []string
	
	localPercent := 0.0
	if totalSpent > 0 {
		localPercent = (localTotal / totalSpent) * 100
	}

	if localPercent < 50 {
		recommendations = append(recommendations, "Consider using local models more for simple tasks to reduce costs")
	}
	if totalSpent > c.dailyBudget*float64(days) {
		recommendations = append(recommendations, fmt.Sprintf("Consider increasing daily budget or optimizing usage (current: $%.2f/day)", c.dailyBudget))
	}
	if len(relevantDays) > 0 && totalTasks/len(relevantDays) > 50 {
		recommendations = append(recommendations, "High task volume detected - consider batch processing")
	}

	return &CostAnalytics{
		TotalSpent: totalSpent,
		LocalVsCloud: map[string]float64{
			"local": localTotal,
			"cloud": cloudTotal,
		},
		CostByTask:      costByTask,
		DailyCosts:      relevantDays,
		Recommendations: recommendations,
	}, nil
}

// Private methods

// loadConfig loads cost configuration from file
func (c *costController) loadConfig() error {
	if _, err := os.Stat(c.configPath); os.IsNotExist(err) {
		return nil // No config file yet, use defaults
	}

	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return err
	}

	var config CostConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Update fields
	if config.DailyBudget > 0 {
		c.dailyBudget = config.DailyBudget
	}
	c.costHistory = config.History

	return nil
}

// saveConfig saves cost configuration to file
func (c *costController) saveConfig() error {
	// Ensure directory exists
	dir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	config := CostConfig{
		DailyBudget: c.dailyBudget,
		History:     c.costHistory,
		LastUpdated: time.Now(),
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.configPath, data, 0644)
}

// estimateCloudCost estimates cost for cloud processing
func (c *costController) estimateCloudCost(task AITask, content string) float64 {
	baseTokens := len(content) / 4 // Rough token estimation

	switch task {
	case TaskCategorize:
		return float64(baseTokens+100) * 0.000001 // Very cheap

	case TaskAnalyze:
		return float64(baseTokens+300) * 0.000002

	case TaskSummarize:
		return float64(baseTokens+500) * 0.000003

	case TaskSynthesize:
		return float64(baseTokens+1000) * 0.000005

	case TaskGenerate:
		return float64(baseTokens+1500) * 0.000007

	default:
		return float64(baseTokens+200) * 0.000002
	}
}