package deepresearch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"briefly/internal/llm"
	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
)

// LLMSynthesizer implements the Synthesizer interface using an LLM
type LLMSynthesizer struct {
	llmClient *llm.Client
}

// NewLLMSynthesizer creates a new LLM-based synthesizer
func NewLLMSynthesizer(llmClient *llm.Client) *LLMSynthesizer {
	return &LLMSynthesizer{
		llmClient: llmClient,
	}
}

// SynthesizeBrief generates a comprehensive research brief from sources
func (s *LLMSynthesizer) SynthesizeBrief(ctx context.Context, topic string, sources []Source, subQueries []string) (*ResearchBrief, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("no sources provided for synthesis")
	}

	// Build the synthesis prompt
	prompt := s.buildSynthesisPrompt(topic, sources, subQueries)

	// Generate the brief content
	model := s.llmClient.GetGenaiModel()
	model.SetTemperature(0.2)
	model.SetTopP(0.9)
	model.SetMaxOutputTokens(4096)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate synthesis: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content generated by the API")
	}

	responsePart := resp.Candidates[0].Content.Parts[0]
	responseText, ok := responsePart.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response format from API, expected genai.Text")
	}

	response := string(responseText)

	// Parse the response into structured sections
	brief := s.parseResponse(response, topic, sources, subQueries)

	return brief, nil
}

// buildSynthesisPrompt creates a comprehensive prompt for research synthesis
func (s *LLMSynthesizer) buildSynthesisPrompt(topic string, sources []Source, subQueries []string) string {
	// Build sources section with citations
	sourcesText := s.buildSourcesSection(sources)

	// Build sub-queries section
	queriesText := strings.Join(subQueries, "\n- ")

	return fmt.Sprintf(`You are a research analyst tasked with synthesizing information from multiple sources into a comprehensive research brief. 

TOPIC: %s

SUB-QUERIES INVESTIGATED:
- %s

SOURCES:
%s

Your task is to create a well-structured research brief that synthesizes the information from these sources. Follow this exact format:

## Executive Summary
Write a 2-3 paragraph executive summary that captures the key insights, current state, and main conclusions about the topic. This should be accessible to a general audience while being comprehensive.

## Detailed Findings

### [Finding Topic 1]
Provide detailed analysis with inline citations [1], [2], etc. Each factual statement should reference specific sources.

### [Finding Topic 2]  
Continue with additional sections as needed, each focused on a specific aspect of the research topic.

### [Finding Topic 3]
Include diverse perspectives and any controversies or debates in the field.

## Open Questions
List 3-5 important questions that remain unanswered or areas needing further research:
- Question 1
- Question 2
- etc.

## Sources
[1] Title - Domain (URL)
[2] Title - Domain (URL)
[Continue listing all sources used]

REQUIREMENTS:
- Every factual claim must have an inline citation [n] referencing the sources list
- Maintain objectivity and present multiple perspectives when they exist
- Focus on recent developments while providing necessary context
- Synthesize rather than just summarize - look for patterns, connections, and insights across sources
- Keep the total length under 2000 words
- Use clear, professional language appropriate for an informed audience

Begin your research brief:`, topic, queriesText, sourcesText)
}

// buildSourcesSection creates the sources section for the prompt
func (s *LLMSynthesizer) buildSourcesSection(sources []Source) string {
	var sourcesText strings.Builder

	for i, source := range sources {
		sourcesText.WriteString(fmt.Sprintf("[%d] %s - %s (%s)\n",
			i+1, source.Title, source.Domain, source.URL))

		// Include first part of content for context
		content := source.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		sourcesText.WriteString(fmt.Sprintf("Content: %s\n\n", content))
	}

	return sourcesText.String()
}

// parseResponse parses the LLM response into a structured ResearchBrief
func (s *LLMSynthesizer) parseResponse(response, topic string, sources []Source, subQueries []string) *ResearchBrief {
	brief := &ResearchBrief{
		ID:          uuid.New().String(),
		Topic:       topic,
		Sources:     sources,
		SubQueries:  subQueries,
		GeneratedAt: time.Now(),
	}

	// Parse sections from the response
	sections := s.extractSections(response)

	// Extract executive summary
	if summary, exists := sections["Executive Summary"]; exists {
		brief.ExecutiveSummary = summary
	}

	// Extract detailed findings
	brief.DetailedFindings = s.extractDetailedFindings(sections)

	// Extract open questions
	if questions, exists := sections["Open Questions"]; exists {
		brief.OpenQuestions = s.parseOpenQuestions(questions)
	}

	return brief
}

// extractSections parses the response into named sections
func (s *LLMSynthesizer) extractSections(response string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(response, "\n")

	var currentSection string
	var currentContent strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check if this is a section header
		if strings.HasPrefix(line, "## ") {
			// Save previous section if exists
			if currentSection != "" {
				sections[currentSection] = strings.TrimSpace(currentContent.String())
			}

			// Start new section
			currentSection = strings.TrimPrefix(line, "## ")
			currentContent.Reset()
		} else if currentSection != "" {
			// Add to current section content
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	// Save the last section
	if currentSection != "" {
		sections[currentSection] = strings.TrimSpace(currentContent.String())
	}

	return sections
}

// extractDetailedFindings parses detailed findings from sections
func (s *LLMSynthesizer) extractDetailedFindings(sections map[string]string) []DetailedFinding {
	var findings []DetailedFinding

	// Look for subsections within "Detailed Findings"
	detailedContent, exists := sections["Detailed Findings"]
	if !exists {
		return findings
	}

	// Split by subsection headers (###)
	subsections := strings.Split(detailedContent, "### ")

	for _, subsection := range subsections {
		if strings.TrimSpace(subsection) == "" {
			continue
		}

		lines := strings.Split(subsection, "\n")
		if len(lines) == 0 {
			continue
		}

		topic := strings.TrimSpace(lines[0])
		content := strings.Join(lines[1:], "\n")
		content = strings.TrimSpace(content)

		if topic != "" && content != "" {
			finding := DetailedFinding{
				Topic:      topic,
				Content:    content,
				Citations:  s.extractCitations(content),
				Confidence: 0.8, // Default confidence
			}
			findings = append(findings, finding)
		}
	}

	return findings
}

// extractCitations finds citation numbers in text like [1], [2], etc.
func (s *LLMSynthesizer) extractCitations(text string) []int {
	var citations []int

	// Simple regex alternative using string parsing
	words := strings.Fields(text)
	for _, word := range words {
		if strings.HasPrefix(word, "[") && strings.HasSuffix(word, "]") {
			numStr := word[1 : len(word)-1]
			if num := s.parseSimpleInt(numStr); num > 0 {
				citations = append(citations, num)
			}
		}
	}

	return citations
}

// parseSimpleInt parses a simple integer string
func (s *LLMSynthesizer) parseSimpleInt(str string) int {
	num := 0
	for _, char := range str {
		if char >= '0' && char <= '9' {
			num = num*10 + int(char-'0')
		} else {
			return 0 // Not a pure number
		}
	}
	return num
}

// parseOpenQuestions extracts questions from the open questions section
func (s *LLMSynthesizer) parseOpenQuestions(questionsText string) []string {
	var questions []string

	lines := strings.Split(questionsText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			question := strings.TrimPrefix(line, "- ")
			question = strings.TrimSpace(question)
			if question != "" {
				questions = append(questions, question)
			}
		} else if strings.HasPrefix(line, "* ") {
			question := strings.TrimPrefix(line, "* ")
			question = strings.TrimSpace(question)
			if question != "" {
				questions = append(questions, question)
			}
		}
	}

	return questions
}
