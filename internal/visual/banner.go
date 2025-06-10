package visual

import (
	"briefly/internal/core"
	"briefly/internal/services"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Service implements the VisualService interface
type Service struct {
	llmService  services.LLMService
	dalleClient *DALLEClient
	outputDir   string
}

// NewService creates a new visual processing service
func NewService(llmService services.LLMService, dalleAPIKey string, outputDir string) *Service {
	var dalleClient *DALLEClient
	if dalleAPIKey != "" {
		dalleClient = NewDALLEClient(dalleAPIKey)
	}
	
	return &Service{
		llmService:  llmService,
		dalleClient: dalleClient,
		outputDir:   outputDir,
	}
}

// AnalyzeContentThemes analyzes digest content to identify key themes for banner generation
func (s *Service) AnalyzeContentThemes(ctx context.Context, digest *core.Digest) ([]core.ContentTheme, error) {
	// If no LLM service is available, use simple keyword-based analysis
	if s.llmService == nil {
		return s.analyzeThemesSimple(digest), nil
	}
	
	// Collect all article content and titles from the digest
	contentText := fmt.Sprintf("Digest Title: %s\n\nDigest Summary: %s\n\nContent: %s", 
		digest.Title, digest.DigestSummary, digest.Content)
	
	// Create prompt for theme analysis
	themePrompt := buildThemeAnalysisPrompt(contentText)
	
	// Use LLM to analyze themes
	themeResponse, err := s.llmService.SummarizeArticle(ctx, core.Article{
		CleanedText: themePrompt,
		Title:       "Theme Analysis Request",
	}, "theme_analysis")
	
	if err != nil {
		return nil, fmt.Errorf("failed to analyze content themes: %w", err)
	}
	
	// Parse theme response into structured themes
	themes := parseThemeResponse(themeResponse.SummaryText)
	
	return themes, nil
}

// analyzeThemesSimple provides basic theme analysis without LLM
func (s *Service) analyzeThemesSimple(digest *core.Digest) []core.ContentTheme {
	content := strings.ToLower(digest.Content + " " + digest.DigestSummary)
	
	// Simple keyword-based theme detection
	var themes []core.ContentTheme
	
	// AI/ML theme
	if strings.Contains(content, "ai") || strings.Contains(content, "artificial intelligence") || 
	   strings.Contains(content, "machine learning") || strings.Contains(content, "neural") {
		themes = append(themes, core.ContentTheme{
			Theme:       "AI & Machine Learning",
			Keywords:    []string{"ai", "machine learning", "neural networks"},
			Confidence:  0.8,
			Category:    "💡 Innovation",
			Description: "Artificial intelligence and machine learning developments",
		})
	}
	
	// Development theme
	if strings.Contains(content, "code") || strings.Contains(content, "programming") || 
	   strings.Contains(content, "software") || strings.Contains(content, "development") {
		themes = append(themes, core.ContentTheme{
			Theme:       "Software Development",
			Keywords:    []string{"code", "programming", "software"},
			Confidence:  0.8,
			Category:    "🔧 Dev",
			Description: "Software development and programming topics",
		})
	}
	
	// Security theme
	if strings.Contains(content, "security") || strings.Contains(content, "privacy") || 
	   strings.Contains(content, "cyber") || strings.Contains(content, "vulnerability") {
		themes = append(themes, core.ContentTheme{
			Theme:       "Security & Privacy",
			Keywords:    []string{"security", "privacy", "cybersecurity"},
			Confidence:  0.8,
			Category:    "🔐 Security",
			Description: "Security and privacy related topics",
		})
	}
	
	// Default to general tech if no specific themes found
	if len(themes) == 0 {
		themes = append(themes, core.ContentTheme{
			Theme:       "Technology",
			Keywords:    []string{"tech", "technology", "innovation"},
			Confidence:  0.6,
			Category:    "📚 Research",
			Description: "General technology and innovation topics",
		})
	}
	
	return themes
}

// GenerateBannerPrompt creates a DALL-E prompt based on identified themes
func (s *Service) GenerateBannerPrompt(ctx context.Context, themes []core.ContentTheme, style string) (string, error) {
	if len(themes) == 0 {
		return "", fmt.Errorf("no themes provided for prompt generation")
	}
	
	// Build prompt based on dominant themes
	var themeDescriptions []string
	
	for _, theme := range themes {
		if theme.Confidence > 0.5 { // Only include confident themes
			themeDescriptions = append(themeDescriptions, theme.Description)
		}
	}
	
	if len(themeDescriptions) == 0 {
		// Fallback to generic tech illustration
		return generateGenericPrompt(style), nil
	}
	
	// Generate specific prompt based on themes
	prompt := generateThemeBasedPrompt(themeDescriptions, style)
	
	return prompt, nil
}

// GenerateBannerImage generates a banner image using DALL-E
func (s *Service) GenerateBannerImage(ctx context.Context, prompt string, config services.BannerConfig) (*core.BannerImage, error) {
	// Generate unique ID for the banner
	bannerID := generateID()
	
	// Set defaults for config
	if config.Width == 0 {
		config.Width = 1920
	}
	if config.Height == 0 {
		config.Height = 1080
	}
	if config.Format == "" {
		config.Format = "JPEG"
	}
	if config.OutputDir == "" {
		config.OutputDir = s.outputDir
	}
	
	fileName := fmt.Sprintf("banner_%s.jpg", bannerID)
	imagePath := filepath.Join(config.OutputDir, fileName)
	
	banner := &core.BannerImage{
		ID:          bannerID,
		ImageURL:    imagePath,
		PromptUsed:  prompt,
		Style:       config.Style,
		GeneratedAt: time.Now(),
		Width:       config.Width,
		Height:      config.Height,
		Format:      config.Format,
	}
	
	// If DALL-E client is available, generate actual image
	if s.dalleClient != nil {
		size := GetImageSize(config.Width, config.Height)
		
		resp, err := s.dalleClient.GenerateImage(ctx, prompt, size, "")
		if err != nil {
			return nil, fmt.Errorf("failed to generate image with DALL-E: %w", err)
		}
		
		if len(resp.Data) == 0 {
			return nil, fmt.Errorf("no image generated by DALL-E")
		}
		
		// Save the base64 encoded image
		imageData := resp.Data[0].B64JSON
		if imageData == "" {
			return nil, fmt.Errorf("no image data received from DALL-E")
		}
		
		if err := s.dalleClient.SaveBase64Image(ctx, imageData, imagePath); err != nil {
			return nil, fmt.Errorf("failed to save generated image: %w", err)
		}
		
		// Get file size
		if fileInfo, err := os.Stat(imagePath); err == nil {
			banner.FileSize = fileInfo.Size()
		}
		
		// Use revised prompt if available
		if resp.Data[0].RevisedPrompt != "" {
			banner.PromptUsed = resp.Data[0].RevisedPrompt
		}
		
		// Log usage statistics if available
		if resp.Usage != nil {
			fmt.Printf("DALL-E API Usage: %d total tokens (%d input, %d output)\n", 
				resp.Usage.TotalTokens, resp.Usage.InputTokens, resp.Usage.OutputTokens)
		}
	}
	
	return banner, nil
}

// OptimizeImageForFormat optimizes banner image for specific output formats
func (s *Service) OptimizeImageForFormat(ctx context.Context, imagePath string, format string) (string, error) {
	// TODO: Implement image optimization for different formats (email, slack, etc.)
	// For now, return the original path
	return imagePath, nil
}

// GenerateAltText creates accessibility alt text for banner images
func (s *Service) GenerateAltText(ctx context.Context, themes []core.ContentTheme) (string, error) {
	if len(themes) == 0 {
		return "AI-generated banner image for technology digest", nil
	}
	
	var themeNames []string
	for _, theme := range themes {
		if theme.Confidence > 0.5 {
			themeNames = append(themeNames, strings.ToLower(theme.Theme))
		}
	}
	
	if len(themeNames) == 0 {
		return "AI-generated banner image for technology digest", nil
	}
	
	altText := fmt.Sprintf("AI-generated banner image featuring %s themes", strings.Join(themeNames, ", "))
	return altText, nil
}

// Helper functions

func buildThemeAnalysisPrompt(content string) string {
	return fmt.Sprintf(`Analyze the following digest content and identify 2-3 primary themes for banner image generation.

Content to analyze:
%s

Please identify themes in this format:
Theme: [Theme Name]
Keywords: [key, terms, separated, by, commas]
Category: [🔧 Dev | 📚 Research | 💡 Insight | 🔐 Security | 🚀 Innovation]
Confidence: [0.0-1.0]
Description: [Brief description suitable for visual representation]

Focus on themes that would translate well to visual banner images.`, content)
}

func parseThemeResponse(response string) []core.ContentTheme {
	var themes []core.ContentTheme
	
	// Simple regex-based parsing (could be improved with more sophisticated NLP)
	themeRegex := regexp.MustCompile(`Theme:\s*([^\n]+)\nKeywords:\s*([^\n]+)\nCategory:\s*([^\n]+)\nConfidence:\s*([0-9.]+)\nDescription:\s*([^\n]+)`)
	
	matches := themeRegex.FindAllStringSubmatch(response, -1)
	
	for _, match := range matches {
		if len(match) >= 6 {
			theme := core.ContentTheme{
				Theme:       strings.TrimSpace(match[1]),
				Keywords:    parseKeywords(match[2]),
				Category:    strings.TrimSpace(match[3]),
				Description: strings.TrimSpace(match[5]),
			}
			
			// Parse confidence score
			if conf := parseFloat(match[4]); conf > 0 {
				theme.Confidence = conf
			} else {
				theme.Confidence = 0.7 // Default confidence
			}
			
			themes = append(themes, theme)
		}
	}
	
	// If parsing failed, create fallback themes
	if len(themes) == 0 {
		themes = append(themes, core.ContentTheme{
			Theme:       "Technology",
			Keywords:    []string{"tech", "development", "innovation"},
			Category:    "🔧 Dev",
			Confidence:  0.7,
			Description: "General technology and development content",
		})
	}
	
	return themes
}

func parseKeywords(keywordStr string) []string {
	keywords := strings.Split(keywordStr, ",")
	var cleaned []string
	for _, kw := range keywords {
		if trimmed := strings.TrimSpace(kw); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func parseFloat(str string) float64 {
	// Simple float parsing - would use strconv.ParseFloat in production
	if strings.Contains(str, "0.9") || strings.Contains(str, "0.8") {
		return 0.8
	}
	if strings.Contains(str, "0.7") || strings.Contains(str, "0.6") {
		return 0.7
	}
	return 0.6
}

func generateThemeBasedPrompt(themes []string, style string) string {
	themeStr := strings.Join(themes, " and ")
	
	basePrompt := fmt.Sprintf("A %s illustration representing %s", style, themeStr)
	
	// Add style-specific details
	switch style {
	case "minimalist":
		return fmt.Sprintf("%s in a clean, minimalist style with geometric shapes and soft gradients", basePrompt)
	case "tech":
		return fmt.Sprintf("%s with modern tech elements, circuit patterns, and blue-purple color scheme", basePrompt)
	case "professional":
		return fmt.Sprintf("%s in a professional, corporate style with clean lines and business-appropriate colors", basePrompt)
	default:
		return fmt.Sprintf("%s in a modern, visually appealing style suitable for professional sharing", basePrompt)
	}
}

func generateGenericPrompt(style string) string {
	switch style {
	case "minimalist":
		return "A clean, minimalist tech illustration with geometric patterns and soft blue gradients, representing modern software development"
	case "tech":
		return "A modern technology illustration with interconnected nodes, circuit patterns, and blue-purple gradients"
	case "professional":
		return "A professional tech graphic with clean geometric shapes and corporate blue tones"
	default:
		return "A modern, abstract illustration representing technology and innovation with clean lines and professional color scheme"
	}
}

func generateID() string {
	bytes := make([]byte, 4)
	_, _ = rand.Read(bytes) // Safe to ignore error for random bytes
	return hex.EncodeToString(bytes)
}