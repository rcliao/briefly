package agent

// TopicCategory defines a stable, recurring topic for digest grouping.
// These categories are consistent across weeks so readers can quickly find
// the sections they care about.
type TopicCategory struct {
	ID          string // Short identifier
	Label       string // Display name with emoji
	Description string // One-line description for LLM context
}

// StableTopics returns the canonical set of topic categories for digest grouping.
// Derived from analysis of 15+ weeks of past digests (Nov 2025 - Mar 2026).
// Articles should be mapped to the BEST-FIT topic. Not every topic appears every week.
func StableTopics() []TopicCategory {
	return []TopicCategory{
		{
			ID:          "model_releases",
			Label:       "🧠 Model Releases & Capabilities",
			Description: "New model launches, benchmark results, architecture breakthroughs (e.g., Claude 4.6, Gemini 3, Mistral, DeepSeek)",
		},
		{
			ID:          "agentic_patterns",
			Label:       "🤖 Agentic Engineering",
			Description: "Agent frameworks, MCP integrations, autonomous workflows, multi-agent patterns, eval frameworks",
		},
		{
			ID:          "dev_tools",
			Label:       "🛠️ Developer Tools & Productivity",
			Description: "IDE tools, code editors, CLI utilities, developer experience improvements (e.g., Claude Code, Cursor, Replit)",
		},
		{
			ID:          "infra_deployment",
			Label:       "☁️ Infrastructure & Deployment",
			Description: "Sandboxing, scaling, edge deployment, cost optimization, context window management, quantization",
		},
		{
			ID:          "security_privacy",
			Label:       "🔒 Security & Privacy",
			Description: "Secret management, data exfiltration, compliance, sandboxing security, credential vaults",
		},
		{
			ID:          "open_source",
			Label:       "📦 Open Source & Models",
			Description: "Open-weight model releases, fine-tuning, community tools, licensing changes",
		},
		{
			ID:          "research",
			Label:       "🔬 Research & Reasoning",
			Description: "Academic papers, reasoning architectures, interpretability, novel techniques",
		},
		{
			ID:          "industry",
			Label:       "📡 Industry & Business",
			Description: "Funding, partnerships, market shifts, org changes, economic impact of AI",
		},
	}
}

// TopicListForPrompt returns a formatted string of all topics for LLM prompts.
func TopicListForPrompt() string {
	var sb string
	for _, t := range StableTopics() {
		sb += t.ID + ": " + t.Description + "\n"
	}
	return sb
}
