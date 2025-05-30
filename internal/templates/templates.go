package templates

import (
	"briefly/internal/render"
	"fmt"
	"strings"
	"time"
)

// DigestFormat represents different digest formats
type DigestFormat string

const (
	// FormatBrief creates a concise digest with key highlights only
	FormatBrief DigestFormat = "brief"
	// FormatStandard creates a balanced digest with summaries and key points
	FormatStandard DigestFormat = "standard"
	// FormatDetailed creates comprehensive digest with full summaries and analysis
	FormatDetailed DigestFormat = "detailed"
	// FormatNewsletter creates a newsletter-style digest optimized for sharing
	FormatNewsletter DigestFormat = "newsletter"
)

// DigestTemplate holds template configuration for different formats
type DigestTemplate struct {
	Format              DigestFormat
	Title               string
	IncludeSummaries    bool
	IncludeKeyInsights  bool
	IncludeActionItems  bool
	IncludeSourceLinks  bool
	MaxSummaryLength    int // 0 for no limit
	IntroductionText    string
	ConclusionText      string
	SectionSeparator    string
}

// GetTemplate returns a pre-configured template for the specified format
func GetTemplate(format DigestFormat) *DigestTemplate {
	switch format {
	case FormatBrief:
		return &DigestTemplate{
			Format:             FormatBrief,
			Title:              "Brief Digest",
			IncludeSummaries:   true,
			IncludeKeyInsights: false,
			IncludeActionItems: false,
			IncludeSourceLinks: true,
			MaxSummaryLength:   150,
			IntroductionText:   "Quick highlights from today's reading:",
			ConclusionText:     "",
			SectionSeparator:   "\n\n---\n\n",
		}
	case FormatStandard:
		return &DigestTemplate{
			Format:             FormatStandard,
			Title:              "Daily Digest",
			IncludeSummaries:   true,
			IncludeKeyInsights: true,
			IncludeActionItems: false,
			IncludeSourceLinks: true,
			MaxSummaryLength:   300,
			IntroductionText:   "Here's what's worth knowing from today's articles:",
			ConclusionText:     "",
			SectionSeparator:   "\n\n---\n\n",
		}
	case FormatDetailed:
		return &DigestTemplate{
			Format:             FormatDetailed,
			Title:              "Comprehensive Digest",
			IncludeSummaries:   true,
			IncludeKeyInsights: true,
			IncludeActionItems: true,
			IncludeSourceLinks: true,
			MaxSummaryLength:   0, // No limit
			IntroductionText:   "In-depth analysis of today's key articles:",
			ConclusionText:     "These insights provide a comprehensive view of current developments in the field.",
			SectionSeparator:   "\n\n---\n\n",
		}
	case FormatNewsletter:
		return &DigestTemplate{
			Format:             FormatNewsletter,
			Title:              "Weekly Newsletter",
			IncludeSummaries:   true,
			IncludeKeyInsights: true,
			IncludeActionItems: true,
			IncludeSourceLinks: true,
			MaxSummaryLength:   250,
			IntroductionText:   "Welcome to this week's curated selection of insights! Here's what caught our attention:",
			ConclusionText:     "Thank you for reading! Forward this to colleagues who might find it valuable.",
			SectionSeparator:   "\n\n💡 **Key Insight**\n\n",
		}
	default:
		return GetTemplate(FormatStandard)
	}
}

// RenderWithTemplate renders a digest using the specified template
func RenderWithTemplate(digestItems []render.DigestData, outputDir string, finalDigest string, template *DigestTemplate) (string, error) {
	dateStr := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("digest_%s_%s.md", strings.ToLower(string(template.Format)), dateStr)
	
	if outputDir == "" {
		outputDir = "digests"
	}

	var content strings.Builder

	// Header
	content.WriteString(fmt.Sprintf("# %s - %s\n\n", template.Title, dateStr))
	
	// Introduction
	if template.IntroductionText != "" {
		content.WriteString(fmt.Sprintf("%s\n\n", template.IntroductionText))
	}

	// Final digest summary (if provided)
	if finalDigest != "" {
		content.WriteString("## Executive Summary\n\n")
		content.WriteString(finalDigest)
		content.WriteString("\n\n")
		content.WriteString("## Individual Articles\n\n")
	}

	// Process each article
	for i, item := range digestItems {
		if i > 0 {
			content.WriteString(template.SectionSeparator)
		}

		// Article title and source
		if template.IncludeSourceLinks {
			content.WriteString(fmt.Sprintf("### %d. [%s](%s)\n\n", i+1, item.Title, item.URL))
		} else {
			content.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, item.Title))
		}

		// Summary
		if template.IncludeSummaries && item.SummaryText != "" {
			summary := item.SummaryText
			if template.MaxSummaryLength > 0 && len(summary) > template.MaxSummaryLength {
				summary = summary[:template.MaxSummaryLength] + "..."
			}
			content.WriteString(fmt.Sprintf("%s\n\n", summary))
		}

		// Key insights (if template supports and data available)
		if template.IncludeKeyInsights && item.MyTake != "" {
			content.WriteString(fmt.Sprintf("**Key Insight:** %s\n\n", item.MyTake))
		}

		// Source link (if not already included in title)
		if !template.IncludeSourceLinks {
			content.WriteString(fmt.Sprintf("*Source: %s*\n\n", item.URL))
		}
	}

	// Conclusion
	if template.ConclusionText != "" {
		content.WriteString(template.SectionSeparator)
		content.WriteString(template.ConclusionText)
		content.WriteString("\n")
	}

	// Write to file
	return render.WriteDigestToFile(content.String(), outputDir, filename)
}

// GetAvailableFormats returns a list of all available format names
func GetAvailableFormats() []string {
	return []string{
		string(FormatBrief),
		string(FormatStandard),
		string(FormatDetailed),
		string(FormatNewsletter),
	}
}
