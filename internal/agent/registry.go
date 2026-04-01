package agent

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/genai"
)

// Tool is the interface all agent tools implement.
type Tool interface {
	// Name returns the tool identifier (matches Gemini function name).
	Name() string
	// Description returns a human-readable description for the LLM.
	Description() string
	// Parameters returns the JSON Schema for the tool's input parameters.
	Parameters() *genai.Schema
	// Execute runs the tool with the given params, reading/writing state through memory.
	Execute(ctx context.Context, memory *WorkingMemory, params map[string]any) (map[string]any, error)
}

// ToolRegistry manages tool registration and dispatch.
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates an empty tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry. Returns error if name is already registered.
func (r *ToolRegistry) Register(tool Tool) error {
	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by name.
func (r *ToolRegistry) Get(name string) (Tool, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %q", name)
	}
	return tool, nil
}

// FunctionDeclarations returns all registered tools as Gemini FunctionDeclaration slice.
func (r *ToolRegistry) FunctionDeclarations() []*genai.FunctionDeclaration {
	decls := make([]*genai.FunctionDeclaration, 0, len(r.tools))
	for _, tool := range r.tools {
		decls = append(decls, &genai.FunctionDeclaration{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return decls
}

// Execute runs a tool by name, logging the call to working memory.
func (r *ToolRegistry) Execute(ctx context.Context, memory *WorkingMemory, name string, params map[string]any) (map[string]any, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	record := ToolCallRecord{
		ToolName:   name,
		Parameters: params,
		StartedAt:  time.Now(),
		Status:     "success",
	}

	result, err := tool.Execute(ctx, memory, params)

	record.CompletedAt = time.Now()
	record.DurationMs = record.CompletedAt.Sub(record.StartedAt).Milliseconds()

	if err != nil {
		record.Status = "error"
		record.ErrorMessage = err.Error()
		record.ResultSummary = fmt.Sprintf("error: %s", err.Error())
		memory.LogToolCall(record)
		return map[string]any{"error": err.Error()}, nil // Return error as data, not Go error
	}

	// Build brief summary from result
	record.ResultSummary = buildResultSummary(name, result)
	memory.LogToolCall(record)

	return result, nil
}

// ToolCount returns the number of registered tools.
func (r *ToolRegistry) ToolCount() int {
	return len(r.tools)
}

// buildResultSummary creates a brief description of a tool result for logging.
func buildResultSummary(toolName string, result map[string]any) string {
	switch toolName {
	case "fetch_articles":
		return fmt.Sprintf("fetched %v/%v articles (%v cache hits)",
			result["successful"], result["total_urls"], result["cache_hits"])
	case "summarize_batch":
		return fmt.Sprintf("summarized %v articles (%v cache hits)",
			result["summaries_generated"], result["cache_hits"])
	case "triage_articles":
		return fmt.Sprintf("triaged: %v include, %v deprioritize, %v exclude",
			result["include_count"], result["deprioritize_count"], result["exclude_count"])
	case "generate_embeddings":
		return fmt.Sprintf("generated %v embeddings", result["embeddings_generated"])
	case "cluster_articles":
		return fmt.Sprintf("created %v clusters", result["total_clusters"])
	case "evaluate_clusters":
		return fmt.Sprintf("evaluated clusters: %v quality", result["overall_quality"])
	case "generate_cluster_narrative":
		return fmt.Sprintf("narrative for cluster %v (confidence: %v)", result["cluster_id"], result["confidence"])
	case "generate_executive_summary":
		return fmt.Sprintf("executive summary generated: %v", truncate(fmt.Sprintf("%v", result["title"]), 50))
	case "reflect":
		return fmt.Sprintf("reflection score: %v (should_continue: %v)", result["overall_score"], result["should_continue"])
	case "revise_section":
		return fmt.Sprintf("revised section: %v", result["section"])
	case "render_digest":
		return fmt.Sprintf("rendered to %v", result["file_path"])
	default:
		return fmt.Sprintf("%s completed", toolName)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
