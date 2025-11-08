package summarize

import (
	"fmt"
	"strings"
)

// PromptType defines different types of summarization prompts
type PromptType string

const (
	PromptTypeDigest    PromptType = "digest"     // Weekly digest summaries
	PromptTypeQuickRead PromptType = "quick_read" // Quick single article reads
	PromptTypeDetailed  PromptType = "detailed"   // Detailed analysis
)

// PromptOptions configures prompt generation
type PromptOptions struct {
	Type             PromptType
	MaxWords         int    // Target word count for summary
	IncludeKeyPoints bool   // Whether to include key points
	KeyPointCount    int    // Number of key points (3-5)
	Format           string // Output format preference
}

// DefaultDigestOptions returns default options for digest summaries
func DefaultDigestOptions() PromptOptions {
	return PromptOptions{
		Type:             PromptTypeDigest,
		MaxWords:         150,
		IncludeKeyPoints: true,
		KeyPointCount:    5,
		Format:           "markdown",
	}
}

// DefaultQuickReadOptions returns default options for quick reads
func DefaultQuickReadOptions() PromptOptions {
	return PromptOptions{
		Type:             PromptTypeQuickRead,
		MaxWords:         200,
		IncludeKeyPoints: true,
		KeyPointCount:    5,
		Format:           "markdown",
	}
}

// BuildSummarizationPrompt creates a prompt for article summarization with fact extraction
func BuildSummarizationPrompt(title, content string, opts PromptOptions) string {
	var prompt strings.Builder

	prompt.WriteString("Summarize this article with CONCRETE FACTS and SPECIFIC DETAILS.\n\n")

	// Article details
	if title != "" {
		prompt.WriteString(fmt.Sprintf("**Title:** %s\n\n", title))
	}

	prompt.WriteString(fmt.Sprintf("**Content:**\n%s\n\n", truncateContent(content, 4000)))

	// PHASE 1: Fact Extraction (NEW)
	prompt.WriteString("**PHASE 1: Extract Concrete Facts**\n")
	prompt.WriteString("Before summarizing, identify these SPECIFIC facts:\n\n")
	prompt.WriteString("WHO:\n")
	prompt.WriteString("- Specific people mentioned (full names, titles)\n")
	prompt.WriteString("- Specific companies/organizations\n")
	prompt.WriteString("- Specific products/technologies\n\n")

	prompt.WriteString("WHAT:\n")
	prompt.WriteString("- Exact numbers, percentages, metrics (e.g., \"40% faster\", \"$1.5M\", \"768 dimensions\")\n")
	prompt.WriteString("- Specific technologies, versions, features\n")
	prompt.WriteString("- Concrete actions taken or announcements made\n\n")

	prompt.WriteString("WHEN:\n")
	prompt.WriteString("- Exact dates mentioned (e.g., \"November 7, 2025\", \"Q4 2024\")\n")
	prompt.WriteString("- Specific timelines (e.g., \"within 6 months\", \"by end of year\")\n\n")

	prompt.WriteString("WHY/IMPACT:\n")
	prompt.WriteString("- Specific problems solved\n")
	prompt.WriteString("- Measurable impact or improvements\n")
	prompt.WriteString("- Target audience or use cases\n\n")

	// PHASE 2: Summary with Facts (ENHANCED)
	prompt.WriteString("**PHASE 2: Create Summary Using Facts**\n")
	prompt.WriteString(fmt.Sprintf("Write a %d-word summary that:\n", opts.MaxWords))
	prompt.WriteString("1. Uses ONLY the concrete facts you extracted above\n")
	prompt.WriteString("2. Includes specific numbers, names, and dates (not vague terms)\n")
	prompt.WriteString("3. Focuses on what actually happened and why it matters\n\n")

	// BANNED PHRASES (NEW)
	prompt.WriteString("**CRITICAL RULES - BANNED VAGUE PHRASES:**\n")
	prompt.WriteString("❌ NEVER use: \"several\", \"various\", \"multiple\", \"many\", \"some\", \"a few\", \"numerous\"\n")
	prompt.WriteString("✅ INSTEAD use: Exact counts or specific examples\n")
	prompt.WriteString("   Example: NOT \"several companies\" → USE \"Google, Meta, and Anthropic\"\n")
	prompt.WriteString("   Example: NOT \"significantly faster\" → USE \"40% faster (2.1s vs 3.5s)\"\n\n")

	// Key Points (if requested)
	if opts.IncludeKeyPoints {
		prompt.WriteString(fmt.Sprintf("**PHASE 3: Extract %d Key Points**\n", opts.KeyPointCount))
		prompt.WriteString("Each key point must:\n")
		prompt.WriteString("- Be specific (include numbers, names, or dates)\n")
		prompt.WriteString("- State ONE concrete fact or insight\n")
		prompt.WriteString("- Avoid vague generalities\n\n")
	}

	// Output format
	prompt.WriteString("**OUTPUT FORMAT:**\n")
	prompt.WriteString("FACTS EXTRACTED:\n")
	prompt.WriteString("WHO: [List specific names/companies]\n")
	prompt.WriteString("WHAT: [List specific numbers/metrics/technologies]\n")
	prompt.WriteString("WHEN: [List specific dates/timelines]\n")
	prompt.WriteString("IMPACT: [List specific problems solved/improvements]\n\n")

	prompt.WriteString("SUMMARY:\n")
	prompt.WriteString(fmt.Sprintf("[Your %d-word summary using facts above]\n\n", opts.MaxWords))

	if opts.IncludeKeyPoints {
		prompt.WriteString("KEY POINTS:\n")
		for i := 1; i <= opts.KeyPointCount; i++ {
			prompt.WriteString(fmt.Sprintf("- [Specific key point %d with numbers/names/dates]\n", i))
		}
	}

	return prompt.String()
}

// BuildKeyPointsPrompt creates a prompt for extracting key points
func BuildKeyPointsPrompt(content string, count int) string {
	return fmt.Sprintf(`Extract the %d most important key points from this content.

**Content:**
%s

**Instructions:**
1. Identify the %d most significant insights, findings, or takeaways
2. Each key point should be concise (1-2 sentences)
3. Focus on actionable insights and important conclusions
4. Order by importance (most important first)

**Output Format:**
Return exactly %d key points as a bulleted list:
- [Key point 1]
- [Key point 2]
...

Key points:`, count, truncateContent(content, 3000), count, count)
}

// BuildTitlePrompt creates a prompt for generating article titles
func BuildTitlePrompt(content string) string {
	return fmt.Sprintf(`Generate a clear, descriptive title for this article.

**Content:**
%s

**Instructions:**
1. Create a title that accurately describes the article's main topic
2. Keep it concise: 5-10 words
3. Use clear, professional language
4. Avoid clickbait or sensational phrasing
5. Capture the essence of the content

**Output Format:**
Return only the title, nothing else.

Title:`, truncateContent(content, 1500))
}

// BuildThemePrompt creates a prompt for identifying article theme
func BuildThemePrompt(title, summary string) string {
	return fmt.Sprintf(`Identify the main theme or topic category for this article.

**Title:** %s

**Summary:** %s

**Instructions:**
1. Identify the primary topic or theme (e.g., "AI/ML", "Security", "Development", "Cloud")
2. Use 2-4 words maximum
3. Be specific but not overly narrow
4. Use standard industry terminology

**Output Format:**
Return only the theme, nothing else.

Theme:`, title, truncateContent(summary, 500))
}

// BuildComparisonPrompt creates a prompt for comparing multiple summaries
func BuildComparisonPrompt(summaries []string) string {
	var prompt strings.Builder

	prompt.WriteString("Analyze these article summaries and identify common themes and connections.\n\n")

	for i, summary := range summaries {
		prompt.WriteString(fmt.Sprintf("**Article %d:**\n%s\n\n", i+1, truncateContent(summary, 300)))
	}

	prompt.WriteString(`**Instructions:**
1. Identify common themes across these articles
2. Note any contrasting viewpoints or complementary insights
3. Highlight the most significant shared topics
4. Keep response concise (100 words max)

**Output Format:**
Return a brief analysis of connections and themes.

Analysis:`)

	return prompt.String()
}

// BuildRefinePrompt creates a prompt for refining a summary
func BuildRefinePrompt(originalSummary, feedback string, targetWords int) string {
	return fmt.Sprintf(`Refine this summary based on the feedback provided.

**Original Summary:**
%s

**Feedback:**
%s

**Instructions:**
1. Address the feedback while maintaining accuracy
2. Keep the refined summary to %d words
3. Preserve key information and insights
4. Improve clarity and readability

**Output Format:**
Return only the refined summary.

Refined summary:`, originalSummary, feedback, targetWords)
}

// BuildSimplificationPrompt creates a prompt for simplifying technical content
func BuildSimplificationPrompt(content string, targetAudience string) string {
	return fmt.Sprintf(`Simplify this technical content for a %s audience.

**Content:**
%s

**Instructions:**
1. Explain technical concepts in accessible language
2. Maintain accuracy while reducing jargon
3. Use analogies or examples where helpful
4. Keep summary to 150-200 words

**Output Format:**
Return the simplified summary.

Simplified summary:`, targetAudience, truncateContent(content, 2000))
}

// truncateContent truncates content to a maximum character length
func truncateContent(content string, maxChars int) string {
	if len(content) <= maxChars {
		return content
	}

	truncated := content[:maxChars]

	// Try to break at sentence boundary
	lastPeriod := strings.LastIndex(truncated, ". ")
	if lastPeriod > maxChars/2 { // Only break at sentence if we're past halfway
		truncated = truncated[:lastPeriod+1]
	} else {
		// Break at word boundary
		lastSpace := strings.LastIndex(truncated, " ")
		if lastSpace > 0 {
			truncated = truncated[:lastSpace]
		}
	}

	return truncated + "..."
}

// ParseSummaryResponse parses the LLM response to extract summary and key points
func ParseSummaryResponse(response string) (summary string, keyPoints []string) {
	lines := strings.Split(response, "\n")

	var inSummary bool
	var inKeyPoints bool
	var summaryLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for section headers
		if strings.HasPrefix(strings.ToUpper(line), "SUMMARY:") {
			inSummary = true
			inKeyPoints = false
			continue
		}

		if strings.HasPrefix(strings.ToUpper(line), "KEY POINTS:") ||
			strings.HasPrefix(strings.ToUpper(line), "KEY TAKEAWAYS:") {
			inSummary = false
			inKeyPoints = true
			continue
		}

		// Parse content
		if inSummary && line != "" {
			summaryLines = append(summaryLines, line)
		}

		if inKeyPoints && line != "" {
			// Extract bullet points
			if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") || strings.HasPrefix(line, "*") {
				point := strings.TrimSpace(line[1:])
				if point != "" {
					keyPoints = append(keyPoints, point)
				}
			} else if len(line) > 2 && line[0] >= '1' && line[0] <= '9' && (line[1] == '.' || line[1] == ')') {
				// Numbered list
				point := strings.TrimSpace(line[2:])
				if point != "" {
					keyPoints = append(keyPoints, point)
				}
			}
		}
	}

	summary = strings.Join(summaryLines, " ")
	summary = strings.TrimSpace(summary)

	return summary, keyPoints
}
