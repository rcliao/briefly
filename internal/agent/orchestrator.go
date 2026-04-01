package agent

import (
	"briefly/internal/llm"
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/genai"
)

const (
	// maxToolCalls is the safety limit on total tool invocations per session.
	maxToolCalls = 60
)

// systemPrompt is the system instruction for the agent orchestrator.
const systemPrompt = `You are a senior editorial AI curating a weekly GenAI newsletter for software engineers.
The goal is simple: take a list of interesting links, group them by topic, and write short editorial
summaries so readers can quickly catch up on what happened this week.

Your workflow:
1. FETCH: Call fetch_articles to load all articles from the input file
2. SUMMARIZE: Call summarize_batch to generate article summaries
3. TRIAGE: Call triage_articles — this is the KEY step. It assigns each article:
   - A topic category (model_releases, agentic_patterns, dev_tools, infra_deployment, security_privacy, open_source, research, industry)
   - A reader intent (skim/read/deep_dive)
   - An editorial summary (1-2 sentences in a natural, human voice)
4. EXECUTIVE SUMMARY: Call generate_executive_summary to create a brief week overview title and TL;DR
5. REFLECT: Call reflect to check quality of the editorial summaries and topic assignments
6. REVISE: If needed, call revise_section for any weak editorial summaries
7. RENDER: Call render_digest to produce the final markdown newsletter. This MUST be your last tool call.

KEY PRINCIPLES:
- The LINKS are the content. Every article should appear as a clickable link with context.
- Each article appears ONCE, under its assigned topic. No redundancy.
- Editorial summaries should sound like a human recommending a link: "Cloudflare figured out how to..." not "This article discusses..."
- Topics are stable categories that recur weekly. Not every topic appears every week.
- Skip clustering/embeddings unless you specifically need them. Triage assigns topics directly.
- Keep it scannable. A reader should be able to scroll top-to-bottom in under 2 minutes.

When done, output a brief text summary (do NOT call more tools after render_digest).`

// Orchestrator drives agentic digest generation using Gemini function-calling.
type Orchestrator struct {
	llmClient *llm.Client
	registry  *ToolRegistry
}

// NewOrchestrator creates a new agent orchestrator.
func NewOrchestrator(llmClient *llm.Client, registry *ToolRegistry) *Orchestrator {
	return &Orchestrator{
		llmClient: llmClient,
		registry:  registry,
	}
}

// Run executes the full agentic digest generation loop.
func (o *Orchestrator) Run(ctx context.Context, session AgentSession) (*AgentDigestResult, error) {
	startTime := time.Now()
	memory := NewWorkingMemory(session.ID)

	// Build Gemini config with tools
	declarations := o.registry.FunctionDeclarations()
	tools := []*genai.Tool{{
		FunctionDeclarations: declarations,
	}}

	temp := float32(0.3)
	maxTokens := int32(8192)
	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
			Role:  "user",
		},
		Temperature:    &temp,
		MaxOutputTokens: maxTokens,
		ToolConfig: &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAuto,
			},
		},
	}

	// Build initial user message
	userMessage := fmt.Sprintf(
		"Generate a digest from %q. Output to %q. Quality threshold: %.2f. Max iterations: %d.",
		session.InputFile, session.OutputPath, session.QualityThreshold, session.MaxIterations,
	)

	history := []*genai.Content{
		{
			Parts: []*genai.Part{{Text: userMessage}},
			Role:  "user",
		},
	}

	totalToolCalls := 0
	toolCallBreakdown := make(map[string]int)

	fmt.Printf("\n🤖 Agent Orchestrator started\n")
	fmt.Printf("   Input: %s\n", session.InputFile)
	fmt.Printf("   Quality threshold: %.2f\n", session.QualityThreshold)
	fmt.Printf("   Max iterations: %d\n\n", session.MaxIterations)

	// Main conversation loop
	for {
		// Safety: check tool call limit
		if totalToolCalls >= maxToolCalls {
			fmt.Printf("   ⚠️  Tool call limit reached (%d). Stopping.\n", maxToolCalls)
			break
		}

		// Call Gemini with tool-use
		resp, err := o.llmClient.GenerateContentWithTools(ctx, history, tools, config)
		if err != nil {
			return nil, fmt.Errorf("Gemini API call failed: %w", err)
		}

		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
			fmt.Printf("   ⚠️  Empty response from Gemini. Stopping.\n")
			break
		}

		modelContent := resp.Candidates[0].Content

		// Check if the model wants to call functions
		var functionCalls []*genai.FunctionCall
		var hasText bool
		for _, part := range modelContent.Parts {
			if part.FunctionCall != nil {
				functionCalls = append(functionCalls, part.FunctionCall)
			}
			if part.Text != "" {
				hasText = true
				fmt.Printf("   💭 Agent: %s\n", truncate(part.Text, 200))
			}
		}

		// Add model response to history
		history = append(history, modelContent)

		// If no function calls, the agent is done
		if len(functionCalls) == 0 {
			if hasText {
				fmt.Printf("   ✅ Agent completed (text-only response)\n")
			}
			break
		}

		// Execute each function call and build response parts
		responseParts := make([]*genai.Part, 0, len(functionCalls))
		for _, fc := range functionCalls {
			totalToolCalls++
			toolCallBreakdown[fc.Name]++

			fmt.Printf("   🔧 [%d] %s", totalToolCalls, fc.Name)

			result, execErr := o.registry.Execute(ctx, memory, fc.Name, fc.Args)
			if execErr != nil {
				log.Printf("      ❌ Error: %v\n", execErr)
				result = map[string]any{"error": execErr.Error()}
			} else {
				fmt.Printf(" ✓\n")
			}

			responseParts = append(responseParts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     fc.Name,
					ID:       fc.ID,
					Response: result,
				},
			})
		}

		// Add function responses to history
		history = append(history, &genai.Content{
			Parts: responseParts,
			Role:  "tool",
		})
	}

	// Ensure render_digest is always called if we have a digest draft
	renderCalled := false
	for _, record := range memory.GetToolCallLog() {
		if record.ToolName == "render_digest" && record.Status == "success" {
			renderCalled = true
			break
		}
	}
	if !renderCalled && memory.GetDigestDraft() != nil {
		fmt.Printf("   🔧 [auto] render_digest (agent didn't call it)")
		renderResult, renderErr := o.registry.Execute(ctx, memory, "render_digest", map[string]any{
			"output_path": session.OutputPath,
		})
		if renderErr != nil {
			fmt.Printf(" ❌ %v\n", renderErr)
		} else {
			fmt.Printf(" ✓\n")
			totalToolCalls++
			toolCallBreakdown["render_digest"]++
			if fp, ok := renderResult["file_path"]; ok {
				fmt.Printf("      → %v\n", fp)
			}
		}
	}

	// Build result
	duration := time.Since(startTime)
	fmt.Printf("\n📊 Agent completed in %s (%d tool calls)\n", duration.Round(time.Second), totalToolCalls)

	result := &AgentDigestResult{
		Digest:       memory.GetDigestDraft(),
		MarkdownPath: "",
		AgentMetadata: AgentMetadata{
			TotalToolCalls:    totalToolCalls,
			ToolCallBreakdown: toolCallBreakdown,
			TotalDurationMs:   duration.Milliseconds(),
		},
	}

	// Extract metadata from memory
	trajectory := memory.GetQualityTrajectory()
	result.AgentMetadata.QualityTrajectory = trajectory
	result.AgentMetadata.TotalIterations = len(memory.GetReflections())

	if len(trajectory) > 0 {
		result.AgentMetadata.FinalQualityScore = trajectory[len(trajectory)-1]
	}

	// Determine early stop reason
	if lastReflection := memory.GetLatestReflection(); lastReflection != nil {
		if lastReflection.OverallScore >= session.QualityThreshold {
			result.AgentMetadata.EarlyStopReason = "threshold_met"
		} else if !lastReflection.ShouldContinue {
			result.AgentMetadata.EarlyStopReason = "diminishing_returns"
		} else if result.AgentMetadata.TotalIterations >= session.MaxIterations {
			result.AgentMetadata.EarlyStopReason = "max_iterations"
		}
	}

	// Extract rendered file path from tool log
	for _, record := range memory.GetToolCallLog() {
		if record.ToolName == "render_digest" && record.Status == "success" {
			// ResultSummary has format "rendered to <path>"
			summary := record.ResultSummary
			if idx := len("rendered to "); len(summary) > idx {
				result.MarkdownPath = summary[idx:]
			}
		}
	}

	// Print quality trajectory
	if len(trajectory) > 0 {
		fmt.Printf("   Quality trajectory: ")
		for i, score := range trajectory {
			if i > 0 {
				fmt.Printf(" → ")
			}
			fmt.Printf("%.2f", score)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("   Tool breakdown: %v\n\n", toolCallBreakdown)

	return result, nil
}
