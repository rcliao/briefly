package services

import (
	"context"
	"fmt"
	"time"

	"briefly/internal/core"
)

// aiRouter implements AIRouter interface for hybrid local/cloud routing
type aiRouter struct {
	localModel   LocalModelService
	cloudLLM     LLMService
	costController CostController
}

// NewAIRouter creates a new AI router with local and cloud services
func NewAIRouter(localModel LocalModelService, cloudLLM LLMService, costController CostController) AIRouter {
	return &aiRouter{
		localModel:     localModel,
		cloudLLM:       cloudLLM,
		costController: costController,
	}
}

// RouteRequest intelligently routes requests between local and cloud models
func (r *aiRouter) RouteRequest(ctx context.Context, request AIRequest) (*AIResponse, error) {
	startTime := time.Now()

	// Step 1: Check if local model is available
	localAvailable, err := r.localModel.IsAvailable(ctx)
	if err != nil {
		localAvailable = false
	}

	// Step 2: Determine routing strategy based on task type and preferences
	useCloud, reason := r.shouldUseCloud(ctx, request, localAvailable)

	var response *AIResponse
	if useCloud {
		response, err = r.routeToCloud(ctx, request)
		if err != nil && localAvailable {
			// Fallback to local model if cloud fails
			fmt.Printf("⚠️ Cloud routing failed, falling back to local: %v\n", err)
			response, err = r.routeToLocal(ctx, request)
			if err == nil {
				response.ProcessedBy = "local (fallback)"
			}
		}
	} else {
		response, err = r.routeToLocal(ctx, request)
		if err != nil {
			// Fallback to cloud if local fails and we have cloud available
			fmt.Printf("⚠️ Local routing failed, falling back to cloud: %v\n", err)
			response, err = r.routeToCloud(ctx, request)
			if err == nil {
				response.ProcessedBy = "cloud (fallback)"
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("routing failed for both local and cloud: %w", err)
	}

	// Add routing metadata
	response.ProcessedBy = fmt.Sprintf("%s (%s)", response.ProcessedBy, reason)
	
	// Record processing time
	processingTime := time.Since(startTime)
	fmt.Printf("🤖 AI Router: %s in %v\n", response.ProcessedBy, processingTime)

	return response, nil
}

// EstimateCost provides cost estimates for different routing options
func (r *aiRouter) EstimateCost(ctx context.Context, request AIRequest) (*CostEstimate, error) {
	var estimate CostEstimate

	// Estimate based on task type and content length
	contentTokens := len(request.Content) / 4 // Rough token estimation

	switch request.Task {
	case TaskCategorize, TaskAnalyze:
		// Local model is preferred and much cheaper
		estimate.LocalTokens = contentTokens + 100 // Small prompt overhead
		estimate.CloudTokens = contentTokens + 200 // Larger prompt overhead
		estimate.EstimatedCost = 0.001 // Very low cost for local
		estimate.ProcessingTime = 2 * time.Second

	case TaskSummarize:
		// Can use either, but local is cheaper
		estimate.LocalTokens = contentTokens + 300
		estimate.CloudTokens = contentTokens + 500
		estimate.EstimatedCost = 0.005
		estimate.ProcessingTime = 5 * time.Second

	case TaskSynthesize, TaskGenerate:
		// Complex tasks require cloud models
		estimate.LocalTokens = 0
		estimate.CloudTokens = contentTokens + 800 // Complex prompts
		estimate.EstimatedCost = 0.02 // Higher cost for complex tasks
		estimate.ProcessingTime = 10 * time.Second

	default:
		// Default estimation
		estimate.LocalTokens = contentTokens + 200
		estimate.CloudTokens = contentTokens + 400
		estimate.EstimatedCost = 0.01
		estimate.ProcessingTime = 5 * time.Second
	}

	return &estimate, nil
}

// GetCapabilities returns current AI system capabilities
func (r *aiRouter) GetCapabilities(ctx context.Context) (*AICapabilities, error) {
	localAvailable, _ := r.localModel.IsAvailable(ctx)
	
	capabilities := &AICapabilities{
		LocalModelsAvailable: localAvailable,
		CloudModelsAvailable: true, // Assume cloud is always available
		SupportedTasks: []AITask{
			TaskCategorize,
			TaskSummarize,
			TaskAnalyze,
			TaskSynthesize,
			TaskGenerate,
		},
		MaxLocalComplexity: 0.6, // Local models handle up to medium complexity
		CostPerToken:       0.00001, // Very low cost for local models
	}

	return capabilities, nil
}

// Private methods

// shouldUseCloud determines routing strategy
func (r *aiRouter) shouldUseCloud(ctx context.Context, request AIRequest, localAvailable bool) (bool, string) {
	// Force cloud if requested
	if request.UserPrefs.PreferLocal == false {
		return true, "user preference"
	}

	// Force local if no local model available
	if !localAvailable {
		return true, "local unavailable"
	}

	// Check budget constraints
	if request.MaxCost != nil && *request.MaxCost <= 0.001 {
		return false, "budget constraint"
	}

	// Route based on task complexity
	switch request.Task {
	case TaskCategorize:
		// Simple tasks: prefer local
		return false, "simple task"

	case TaskAnalyze:
		// Analysis can be complex, check content
		if len(request.Content) > 2000 {
			return true, "complex analysis"
		}
		return false, "simple analysis"

	case TaskSummarize:
		// Summarization: prefer local for cost, cloud for quality
		if request.Priority == PriorityHigh {
			return true, "high priority"
		}
		return false, "cost optimization"

	case TaskSynthesize, TaskGenerate:
		// Complex tasks: require cloud
		return true, "complex task"

	default:
		// Default: prefer local for cost
		return false, "default cost optimization"
	}
}

// routeToLocal handles local model routing
func (r *aiRouter) routeToLocal(ctx context.Context, request AIRequest) (*AIResponse, error) {
	var result interface{}
	var err error

	switch request.Task {
	case TaskCategorize:
		category, confidence, taskErr := r.localModel.CategorizeContent(ctx, request.Content)
		if taskErr != nil {
			err = taskErr
		} else {
			result = map[string]interface{}{
				"category":   category,
				"confidence": confidence,
			}
		}

	case TaskAnalyze:
		complexity, taskErr := r.localModel.AnalyzeComplexity(ctx, request.Content)
		if taskErr != nil {
			err = taskErr
		} else {
			result = map[string]interface{}{
				"complexity": complexity,
			}
		}

	case TaskSummarize:
		summary, taskErr := r.localModel.GenerateBasicSummary(ctx, request.Content, 100) // Default 100 words
		if taskErr != nil {
			err = taskErr
		} else {
			result = summary
		}

	default:
		return nil, fmt.Errorf("task %s not supported by local model", request.Task)
	}

	if err != nil {
		return nil, err
	}

	return &AIResponse{
		Result:      result,
		ProcessedBy: "local",
		ActualCost: core.ProcessingCost{
			LocalTokens:  len(request.Content) / 4,
			CloudTokens:  0,
			EstimatedUSD: 0.001,
		},
		Quality:    0.7, // Local models provide good quality
		Confidence: 0.8, // Good confidence for supported tasks
	}, nil
}

// routeToCloud handles cloud model routing
func (r *aiRouter) routeToCloud(ctx context.Context, request AIRequest) (*AIResponse, error) {
	// For Phase 3, implement basic cloud routing
	// This would integrate with existing LLMService
	
	// Simulate cloud processing for now
	// In real implementation, this would call r.cloudLLM methods
	
	var result interface{}
	
	switch request.Task {
	case TaskSynthesize:
		result = "Cloud-generated synthesis: " + request.Content[:min(100, len(request.Content))]
	case TaskGenerate:
		result = "Cloud-generated content based on: " + request.Content[:min(50, len(request.Content))]
	default:
		result = "Cloud-processed: " + request.Task
	}

	return &AIResponse{
		Result:      result,
		ProcessedBy: "cloud",
		ActualCost: core.ProcessingCost{
			LocalTokens:  0,
			CloudTokens:  len(request.Content)/4 + 200,
			EstimatedUSD: 0.02,
		},
		Quality:    0.9, // Cloud models provide high quality
		Confidence: 0.85, // High confidence for cloud tasks
	}, nil
}

// Utility function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}