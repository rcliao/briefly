package email

import (
	"briefly/internal/render"
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// EmailTemplate represents an HTML email template configuration
type EmailTemplate struct {
	Name              string
	Subject           string
	IncludeCSS        bool
	HeaderColor       string
	BackgroundColor   string
	TextColor         string
	LinkColor         string
	BorderColor       string
	MaxWidth          string
	FontFamily        string
	ShowTopicClusters bool
	ShowInsights      bool
}

// EmailData contains all data needed for email rendering
type EmailData struct {
	Title               string
	Date                string
	Introduction        string
	ExecutiveSummary    string
	DigestItems         []render.DigestData
	TopicGroups         []TopicGroup
	OverallSentiment    string
	AlertsSummary       string
	TrendsSummary       string
	ResearchSuggestions []string
	Conclusion          string
}

// TopicGroup represents a group of articles with the same topic cluster
type TopicGroup struct {
	TopicCluster  string
	Articles      []render.DigestData
	AvgConfidence float64
}

// GetDefaultEmailTemplate returns a modern, responsive HTML email template
func GetDefaultEmailTemplate() *EmailTemplate {
	return &EmailTemplate{
		Name:              "default",
		Subject:           "Your Briefly Digest - {{.Date}}",
		IncludeCSS:        true,
		HeaderColor:       "#2563eb", // Blue-600
		BackgroundColor:   "#f8fafc", // Slate-50
		TextColor:         "#1e293b", // Slate-800
		LinkColor:         "#3b82f6", // Blue-500
		BorderColor:       "#e2e8f0", // Slate-200
		MaxWidth:          "600px",
		FontFamily:        "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif",
		ShowTopicClusters: true,
		ShowInsights:      true,
	}
}

// GetNewsletterEmailTemplate returns a newsletter-style email template
func GetNewsletterEmailTemplate() *EmailTemplate {
	return &EmailTemplate{
		Name:              "newsletter",
		Subject:           "Weekly Newsletter - {{.Date}}",
		IncludeCSS:        true,
		HeaderColor:       "#059669", // Emerald-600
		BackgroundColor:   "#f0fdf4", // Green-50
		TextColor:         "#064e3b", // Emerald-900
		LinkColor:         "#10b981", // Emerald-500
		BorderColor:       "#d1fae5", // Emerald-100
		MaxWidth:          "700px",
		FontFamily:        "Georgia, 'Times New Roman', serif",
		ShowTopicClusters: true,
		ShowInsights:      true,
	}
}

// GetMinimalEmailTemplate returns a clean, minimal email template
func GetMinimalEmailTemplate() *EmailTemplate {
	return &EmailTemplate{
		Name:              "minimal",
		Subject:           "Digest - {{.Date}}",
		IncludeCSS:        true,
		HeaderColor:       "#374151", // Gray-700
		BackgroundColor:   "#ffffff", // White
		TextColor:         "#111827", // Gray-900
		LinkColor:         "#6366f1", // Indigo-500
		BorderColor:       "#e5e7eb", // Gray-200
		MaxWidth:          "560px",
		FontFamily:        "Inter, system-ui, sans-serif",
		ShowTopicClusters: false,
		ShowInsights:      false,
	}
}

// getEmailCSS returns responsive CSS for the email template
func getEmailCSS(tmpl *EmailTemplate) string {
	return fmt.Sprintf(`
<style type="text/css">
  /* Reset styles */
  body, table, td, p, a, li, blockquote {
    -webkit-text-size-adjust: 100%%;
    -ms-text-size-adjust: 100%%;
  }
  table, td {
    mso-table-lspace: 0pt;
    mso-table-rspace: 0pt;
  }
  img {
    -ms-interpolation-mode: bicubic;
    border: 0;
    height: auto;
    line-height: 100%%;
    outline: none;
    text-decoration: none;
  }

  /* Base styles */
  body {
    margin: 0 !important;
    padding: 0 !important;
    background-color: %s;
    font-family: %s;
    color: %s;
    line-height: 1.6;
  }

  /* Container */
  .container {
    max-width: %s;
    margin: 0 auto;
    background-color: #ffffff;
    border: 1px solid %s;
    border-radius: 8px;
    overflow: hidden;
  }

  /* Header */
  .header {
    background-color: %s;
    color: #ffffff;
    padding: 24px;
    text-align: center;
  }
  .header h1 {
    margin: 0;
    font-size: 24px;
    font-weight: 600;
  }
  .header .date {
    margin: 8px 0 0 0;
    font-size: 14px;
    opacity: 0.9;
  }

  /* Content */
  .content {
    padding: 24px;
  }

  /* Typography */
  h2 {
    color: %s;
    font-size: 20px;
    font-weight: 600;
    margin: 32px 0 16px 0;
    border-bottom: 2px solid %s;
    padding-bottom: 8px;
  }
  h3 {
    color: %s;
    font-size: 18px;
    font-weight: 600;
    margin: 24px 0 12px 0;
  }
  h4 {
    color: %s;
    font-size: 16px;
    font-weight: 600;
    margin: 20px 0 8px 0;
  }
  p {
    margin: 0 0 16px 0;
    font-size: 16px;
    line-height: 1.6;
  }
  a {
    color: %s;
    text-decoration: none;
  }
  a:hover {
    text-decoration: underline;
  }

  /* Article cards */
  .article-card {
    background-color: #f8fafc;
    border: 1px solid %s;
    border-radius: 6px;
    padding: 20px;
    margin: 16px 0;
  }
  .article-title {
    font-size: 18px;
    font-weight: 600;
    color: %s;
    margin: 0 0 12px 0;
  }
  .article-summary {
    font-size: 15px;
    line-height: 1.6;
    margin: 0 0 16px 0;
  }
  .article-meta {
    font-size: 13px;
    color: #64748b;
    margin: 12px 0 0 0;
  }

  /* Topic groups */
  .topic-group {
    margin: 24px 0;
    border-left: 4px solid %s;
    padding-left: 16px;
  }
  .topic-title {
    color: %s;
    font-size: 16px;
    font-weight: 600;
    margin: 0 0 16px 0;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  /* Insights section */
  .insights-section {
    background: linear-gradient(135deg, #f0f9ff 0%%, #e0f2fe 100%%);
    border: 1px solid #bae6fd;
    border-radius: 8px;
    padding: 20px;
    margin: 24px 0;
  }
  .insights-title {
    color: #0c4a6e;
    font-size: 18px;
    font-weight: 600;
    margin: 0 0 16px 0;
    display: flex;
    align-items: center;
  }
  .insight-item {
    margin: 12px 0;
    padding: 12px;
    background-color: rgba(255, 255, 255, 0.7);
    border-radius: 6px;
  }
  .insight-label {
    font-weight: 600;
    color: #0369a1;
    margin-bottom: 4px;
  }

  /* Buttons */
  .btn {
    display: inline-block;
    padding: 12px 24px;
    background-color: %s;
    color: #ffffff;
    border-radius: 6px;
    text-decoration: none;
    font-weight: 600;
    margin: 8px 0;
  }
  .btn:hover {
    background-color: #1d4ed8;
    text-decoration: none;
  }

  /* Footer */
  .footer {
    background-color: #f1f5f9;
    padding: 20px 24px;
    text-align: center;
    font-size: 14px;
    color: #64748b;
    border-top: 1px solid %s;
  }

  /* Responsive */
  @media only screen and (max-width: 600px) {
    .container {
      margin: 0 !important;
      border-radius: 0 !important;
      border-left: none !important;
      border-right: none !important;
    }
    .content {
      padding: 16px !important;
    }
    .header {
      padding: 16px !important;
    }
    h2 {
      font-size: 18px !important;
    }
    h3 {
      font-size: 16px !important;
    }
    .article-card {
      padding: 16px !important;
    }
  }
</style>
`,
		tmpl.BackgroundColor, tmpl.FontFamily, tmpl.TextColor, tmpl.MaxWidth,
		tmpl.BorderColor, tmpl.HeaderColor, tmpl.HeaderColor, tmpl.BorderColor,
		tmpl.TextColor, tmpl.TextColor, tmpl.LinkColor, tmpl.BorderColor,
		tmpl.TextColor, tmpl.HeaderColor, tmpl.HeaderColor, tmpl.LinkColor,
		tmpl.BorderColor)
}

// groupArticlesByTopic groups articles by their topic clusters
func groupArticlesByTopic(digestItems []render.DigestData) []TopicGroup {
	if len(digestItems) == 0 {
		return []TopicGroup{}
	}

	// Group articles by topic cluster
	topicMap := make(map[string][]render.DigestData)
	confidenceMap := make(map[string][]float64)

	for _, item := range digestItems {
		topicCluster := item.TopicCluster
		if topicCluster == "" {
			topicCluster = "General"
		}

		topicMap[topicCluster] = append(topicMap[topicCluster], item)
		confidenceMap[topicCluster] = append(confidenceMap[topicCluster], item.TopicConfidence)
	}

	// Convert to TopicGroup slice and calculate average confidence
	var groups []TopicGroup
	for cluster, articles := range topicMap {
		var totalConfidence float64
		for _, conf := range confidenceMap[cluster] {
			totalConfidence += conf
		}
		avgConfidence := totalConfidence / float64(len(articles))

		groups = append(groups, TopicGroup{
			TopicCluster:  cluster,
			Articles:      articles,
			AvgConfidence: avgConfidence,
		})
	}

	return groups
}

// RenderHTMLEmail renders digest data as HTML email
func RenderHTMLEmail(data EmailData, emailTemplate *EmailTemplate) (string, error) {
	// Group articles by topic if enabled
	if emailTemplate.ShowTopicClusters {
		data.TopicGroups = groupArticlesByTopic(data.DigestItems)
	}

	// Create the HTML template
	htmlTemplate := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    {{if .Template.IncludeCSS}}{{.CSS}}{{end}}
</head>
<body>
    <table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%">
        <tr>
            <td align="center">
                <div class="container">
                    <!-- Header -->
                    <div class="header">
                        <h1>{{.Data.Title}}</h1>
                        <p class="date">{{.Data.Date}}</p>
                    </div>

                    <!-- Content -->
                    <div class="content">
                        {{if .Data.Introduction}}
                        <p>{{.Data.Introduction}}</p>
                        {{end}}

                        {{if .Data.ExecutiveSummary}}
                        <h2>📋 Executive Summary</h2>
                        <p>{{.Data.ExecutiveSummary}}</p>
                        {{end}}

                        {{if and .Template.ShowInsights (or .Data.OverallSentiment .Data.AlertsSummary .Data.TrendsSummary .Data.ResearchSuggestions)}}
                        <div class="insights-section">
                            <h2 class="insights-title">🧠 AI-Powered Insights</h2>
                            
                            {{if .Data.OverallSentiment}}
                            <div class="insight-item">
                                <div class="insight-label">📊 Sentiment Analysis</div>
                                <div>{{.Data.OverallSentiment}}</div>
                            </div>
                            {{end}}

                            {{if .Data.AlertsSummary}}
                            <div class="insight-item">
                                <div class="insight-label">🚨 Alerts</div>
                                <div>{{.Data.AlertsSummary}}</div>
                            </div>
                            {{end}}

                            {{if .Data.TrendsSummary}}
                            <div class="insight-item">
                                <div class="insight-label">📈 Trends</div>
                                <div>{{.Data.TrendsSummary}}</div>
                            </div>
                            {{end}}

                            {{if .Data.ResearchSuggestions}}
                            <div class="insight-item">
                                <div class="insight-label">🔍 Research Suggestions</div>
                                <ul>
                                {{range .Data.ResearchSuggestions}}
                                <li>{{.}}</li>
                                {{end}}
                                </ul>
                            </div>
                            {{end}}
                        </div>
                        {{end}}

                        {{if .Template.ShowTopicClusters}}
                        <!-- Topic-based articles -->
                        {{range .Data.TopicGroups}}
                        <div class="topic-group">
                            <h2 class="topic-title">📑 {{.TopicCluster}}</h2>
                            {{range .Articles}}
                            <div class="article-card">
                                <h3 class="article-title">
                                    {{if .SentimentEmoji}}{{.SentimentEmoji}} {{end}}{{.Title}}
                                </h3>
                                {{if .SummaryText}}
                                <div class="article-summary">{{.SummaryText}}</div>
                                {{end}}
                                {{if .MyTake}}
                                <div style="background-color: #fef3c7; padding: 12px; border-radius: 4px; margin: 12px 0; border-left: 4px solid #f59e0b;">
                                    <strong>💡 Key Insight:</strong> {{.MyTake}}
                                </div>
                                {{end}}
                                <div class="article-meta">
                                    <a href="{{.URL}}" class="btn">Read Article</a>
                                    {{if .AlertTriggered}}
                                    <span style="color: #dc2626; font-weight: 600; margin-left: 12px;">🚨 Alert Triggered</span>
                                    {{end}}
                                </div>
                            </div>
                            {{end}}
                        </div>
                        {{end}}
                        {{else}}
                        <!-- Traditional article listing -->
                        <h2>📄 Articles</h2>
                        {{range $index, $article := .Data.DigestItems}}
                        <div class="article-card">
                            <h3 class="article-title">
                                {{if $article.SentimentEmoji}}{{$article.SentimentEmoji}} {{end}}{{$article.Title}}
                            </h3>
                            {{if $article.SummaryText}}
                            <div class="article-summary">{{$article.SummaryText}}</div>
                            {{end}}
                            {{if $article.MyTake}}
                            <div style="background-color: #fef3c7; padding: 12px; border-radius: 4px; margin: 12px 0; border-left: 4px solid #f59e0b;">
                                <strong>💡 Key Insight:</strong> {{$article.MyTake}}
                            </div>
                            {{end}}
                            <div class="article-meta">
                                <a href="{{$article.URL}}" class="btn">Read Article</a>
                                {{if $article.AlertTriggered}}
                                <span style="color: #dc2626; font-weight: 600; margin-left: 12px;">🚨 Alert Triggered</span>
                                {{end}}
                            </div>
                        </div>
                        {{end}}
                        {{end}}

                        {{if .Data.Conclusion}}
                        <h2>🎯 Conclusion</h2>
                        <p>{{.Data.Conclusion}}</p>
                        {{end}}
                    </div>

                    <!-- Footer -->
                    <div class="footer">
                        <p>Generated by <a href="https://github.com/rcliao/briefly">Briefly</a> on {{.Data.Date}}</p>
                        <p style="font-size: 12px; margin-top: 8px;">
                            This digest was created using AI-powered analysis and insights.
                        </p>
                    </div>
                </div>
            </td>
        </tr>
    </table>
</body>
</html>`

	// Parse and execute template
	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse email template: %w", err)
	}

	templateData := struct {
		Data     EmailData
		Template *EmailTemplate
		CSS      template.HTML
	}{
		Data:     data,
		Template: emailTemplate,
		CSS:      template.HTML(getEmailCSS(emailTemplate)),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

// GenerateSubject generates email subject using template
func GenerateSubject(emailTemplate *EmailTemplate, title string, date string) (string, error) {
	tmpl, err := template.New("subject").Parse(emailTemplate.Subject)
	if err != nil {
		return "", fmt.Errorf("failed to parse subject template: %w", err)
	}

	data := struct {
		Title string
		Date  string
	}{
		Title: title,
		Date:  date,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute subject template: %w", err)
	}

	return buf.String(), nil
}

// WriteHTMLEmail writes HTML email content to file
func WriteHTMLEmail(content string, outputDir string, filename string) (string, error) {
	if !strings.HasSuffix(filename, ".html") {
		filename = strings.TrimSuffix(filename, ".md") + ".html"
	}

	return render.WriteDigestToFile(content, outputDir, filename)
}

// ConvertDigestToEmail converts digest data to email format
func ConvertDigestToEmail(digestItems []render.DigestData, title string, introduction string, executiveSummary string, conclusion string, overallSentiment string, alertsSummary string, trendsSummary string, researchSuggestions []string) EmailData {
	return EmailData{
		Title:               title,
		Date:                time.Now().Format("January 2, 2006"),
		Introduction:        introduction,
		ExecutiveSummary:    executiveSummary,
		DigestItems:         digestItems,
		OverallSentiment:    overallSentiment,
		AlertsSummary:       alertsSummary,
		TrendsSummary:       trendsSummary,
		ResearchSuggestions: researchSuggestions,
		Conclusion:          conclusion,
	}
}
