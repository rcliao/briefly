package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"briefly/llmclient" // Corrected import path
)

// ArticleData holds information about each processed article
type ArticleData struct {
	URL              string // Original URL from the markdown file
	RawText          string // Extracted raw text from the article
	SanitizedFileName string // Sanitized filename for saving locally
	LocalPath        string // Full local path where the article is saved
	Summary          string // Summary of the article
	Error            error   // Error encountered during processing, if any
}

var ( // Renamed from articles to processedArticles for clarity
	processedArticles []ArticleData
	geminiAPIKey      string
)

const ( // Default values for configuration
	defaultTempContentPath = "./temp_content/"
	defaultOutputDir       = "./digests/"
	defaultModelName       = "gemini-1.5-flash-latest"
)

// readMarkdownFile reads the content of a file given its path.
func readMarkdownFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read markdown file %s: %w", filePath, err)
	}
	return string(content), nil
}

// extractURLs parses the given markdown content and extracts all URLs.
func extractURLs(markdownContent string) ([]string, error) {
	// Regex for Markdown links: extracts the URL part from [text](URL)
	// This handles URLs with parentheses by matching non-parentheses chars and parentheses pairs
	markdownLinkRegex := regexp.MustCompile(`\[[^\]]*\]\((https?://[^)]*\([^)]*\)[^)]*|https?://[^)]*)\)`)
	// Regex for standalone URLs: finds http(s):// followed by non-whitespace characters,
	// excluding common trailing punctuation if not part of the URL itself.
	standaloneURLRegex := regexp.MustCompile(`https?://[^\s<>"\'']+(?:[^\s<>"\'\.?!,;:)])`)

	foundURLs := make(map[string]bool)
	var urls []string

	// First, extract URLs from Markdown links
	markdownMatches := markdownLinkRegex.FindAllStringSubmatch(markdownContent, -1)
	for _, match := range markdownMatches {
		if len(match) > 1 && match[1] != "" {
			url := match[1]
			if !foundURLs[url] {
				urls = append(urls, url)
				foundURLs[url] = true
			}
		}
	}

	// Then, extract standalone URLs (and avoid re-adding those already found in Markdown links)
	// To avoid capturing parts of markdown links again, we can temporarily replace them.
	// This is a simplification; a proper parser would be better for complex cases.
	tempContent := markdownLinkRegex.ReplaceAllString(markdownContent, "")
	standaloneMatches := standaloneURLRegex.FindAllString(tempContent, -1)
	for _, url := range standaloneMatches {
		// Basic cleaning of common trailing punctuation for standalone URLs
		cleanedURL := strings.TrimRight(url, ".?!,;:")
		if !foundURLs[cleanedURL] {
			urls = append(urls, cleanedURL)
			foundURLs[cleanedURL] = true
		}
	}

	if len(urls) == 0 {
		return []string{}, nil
	}
	return urls, nil
}

// fetchURLContent fetches the content of a given URL.
func fetchURLContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch URL %s: status code %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from %s: %w", url, err)
	}
	return string(body), nil
}

// extractTextFromHTML parses HTML content and extracts plain text.
func extractTextFromHTML(htmlContent string, sourceURL string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML from %s: %w", sourceURL, err)
	}

	// Remove common non-content elements
	doc.Find("script, style, nav, footer, header, aside, form, iframe, noscript").Remove()

	// Try to get text from common main content selectors first
	var text string
	mainContentSelectors := []string{"article", "main", ".main", "#main", ".content", "#content", ".post-body", ".entry-content"}
	for _, selector := range mainContentSelectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			text += s.Text() + " "
		})
		if strings.TrimSpace(text) != "" {
			break // Found content with a specific selector
		}
	}

	// If no specific main content found, get all text from the body
	if strings.TrimSpace(text) == "" {
		text = doc.Find("body").Text()
	}

	// Basic cleaning: replace multiple newlines/spaces with a single space, and trim.
	cleanedText := strings.Join(strings.Fields(text), " ")

	if strings.TrimSpace(cleanedText) == "" {
		return "", fmt.Errorf("no meaningful text content found in %s after parsing", sourceURL)
	}

	return cleanedText, nil
}

// sanitizeFilename creates a safe filename from a URL.
func sanitizeFilename(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	var hostAndPath string
	if err == nil {
		hostAndPath = parsedURL.Host + strings.ReplaceAll(parsedURL.Path, "/", "_")
	} else {
		// Fallback for unparseable URLs
		hostAndPath = strings.ReplaceAll(rawURL, "://", "_")
		hostAndPath = strings.ReplaceAll(hostAndPath, "/", "_")
	}

	// Remove or replace characters not suitable for filenames
	reg := regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)
	sanitized := reg.ReplaceAllString(hostAndPath, "")

	// Truncate if too long
	maxLength := 100
	if len(sanitized) > maxLength {
		sanitized = sanitized[:maxLength]
	}
	if sanitized == "" {
		return "default_filename.txt" // Fallback for completely empty sanitized names
	}
	return sanitized + ".txt"
}

// saveTextToFile saves the given text content to a file in the specified directory.
func saveTextToFile(filePath string, textContent string) error {
	err := os.WriteFile(filePath, []byte(textContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}
	return nil
}

// prepareFinalDigestPrompt prepares content from articles and calls an LLM to generate the final digest.
func prepareFinalDigestPrompt(articles []ArticleData) string {
	var builder strings.Builder
	for _, article := range articles {
		if article.Error != nil {
			// Optionally include a note about articles that failed processing
			builder.WriteString(fmt.Sprintf("URL: %s\nError: Could not process or summarize this article (%v).\n\n", article.URL, article.Error))
		} else if article.Summary != "" {
			builder.WriteString(fmt.Sprintf("URL: %s\nSummary: %s\n\n", article.URL, article.Summary))
		}
	}
	return strings.TrimSpace(builder.String())
}

func main() {
	// Load .env file. This is a good place for the API key.
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on environment variables or flags for API key")
	}

	// --- Configuration Setup using flags --- 
	inputFile := flag.String("input", "", "Path to the Markdown file containing URLs")
	apiKeyFlag := flag.String("api-key", os.Getenv("GEMINI_API_KEY"), "Gemini API Key")
	modelNameFlag := flag.String("model", defaultModelName, "Gemini model name (e.g., gemini-1.5-flash-latest)")
	tempPathFlag := flag.String("temp-path", defaultTempContentPath, "Path to temporary folder for fetched content")
	outputPathFlag := flag.String("output-path", defaultOutputDir, "Directory to save the final digest")
	flag.Parse()

	// --- Setup Logger ---
	log.SetFlags(log.LstdFlags | log.Lmicroseconds) // Add microsecond precision to logs
	log.Println("-----------------------------------------------------")
	log.Println("🚀 Starting Briefly: AI-Powered Digest Generator 🚀")
	log.Println("-----------------------------------------------------")

	// --- Configuration Validation and Setup ---
	log.Println("[CONFIG] Validating configuration...")
	if *inputFile == "" {
		log.Fatal("[ERROR] Input file path is required. Use -input <filepath>")
	}

	geminiAPIKey = *apiKeyFlag
	if geminiAPIKey == "" {
		log.Fatal("[ERROR] Gemini API Key is required. Set GEMINI_API_KEY environment variable or use -api-key flag.")
	}

	modelName := *modelNameFlag
	tempContentPath := *tempPathFlag
	outputDir := *outputPathFlag

	log.Printf("[CONFIG] Input file: %s", *inputFile)
	log.Printf("[CONFIG] Temporary content path: %s", tempContentPath)
	log.Printf("[CONFIG] Output directory: %s", outputDir)
	log.Printf("[CONFIG] Using LLM model: %s", modelName)

	// Ensure temp and output directories exist
	log.Println("[SETUP] Ensuring temporary and output directories exist...")
	if err := os.MkdirAll(tempContentPath, os.ModePerm); err != nil {
		log.Fatalf("[ERROR] Failed to create temporary content directory '%s': %v", tempContentPath, err)
	}
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Fatalf("[ERROR] Failed to create output directory '%s': %v", outputDir, err)
	}
	log.Println("[SETUP] Directories ensured.")

	// --- Main Application Logic ---
	log.Println("-----------------------------------------------------")
	log.Println("🔍 Phase 1: URL Extraction & Validation")
	log.Println("-----------------------------------------------------")
	log.Printf("[EXTRACT] Reading input file: %s...", *inputFile)
	markdownContent, err := readMarkdownFile(*inputFile)
	if err != nil {
		log.Fatalf("[ERROR] Failed to read markdown file: %v", err)
	}
	log.Println("[EXTRACT] Input file read successfully.")

	log.Println("[EXTRACT] Extracting URLs from markdown content...")
	urls, err := extractURLs(markdownContent)
	if err != nil {
		log.Fatalf("[ERROR] Failed to extract URLs: %v", err)
	}

	if len(urls) == 0 {
		log.Println("[INFO] No URLs found in the input file. Exiting.")
		return
	}
	log.Printf("[EXTRACT] Found %d URLs to process initially.", len(urls))

	log.Println("[VALIDATE] Validating extracted URLs...")
	validURLs := []string{}
	for _, u := range urls {
		parsedURL, err := url.ParseRequestURI(u)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			log.Printf("[WARN] Invalid or unsupported URL scheme for '%s': %v. Skipping.", u, err)
			continue
		}
		validURLs = append(validURLs, u)
	}

	if len(validURLs) == 0 {
		log.Println("[INFO] No valid HTTP/HTTPS URLs found after validation. Exiting.")
		return
	}
	log.Printf("[VALIDATE] %d valid URLs to process.", len(validURLs))

	log.Println("-----------------------------------------------------")
	log.Println("🔄 Phase 2: Content Processing & Summarization")
	log.Println("-----------------------------------------------------")
	for i, u := range validURLs {
		log.Printf("[PROCESS %d/%d] Starting URL: %s", i+1, len(validURLs), u)
		article := ArticleData{URL: u}

		log.Printf("[FETCH %d/%d] Fetching content...", i+1, len(validURLs))
		content, err := fetchURLContent(u)
		if err != nil {
			log.Printf("[ERROR %d/%d] Fetching content for %s: %v. Skipping.", i+1, len(validURLs), u, err)
			article.Error = err
			processedArticles = append(processedArticles, article)
			continue
		}
		log.Printf("[FETCH %d/%d] Content fetched successfully.", i+1, len(validURLs))

		log.Printf("[PARSE %d/%d] Extracting text from HTML...", i+1, len(validURLs))
		article.RawText, err = extractTextFromHTML(content, u)
		if err != nil {
			log.Printf("[ERROR %d/%d] Extracting text for %s: %v. Skipping.", i+1, len(validURLs), u, err)
			article.Error = err
			processedArticles = append(processedArticles, article)
			continue
		}

		if strings.TrimSpace(article.RawText) == "" {
			log.Printf("[WARN %d/%d] No significant text content extracted from %s. Skipping summarization.", i+1, len(validURLs), u)
			article.Error = fmt.Errorf("no significant text content extracted")
			processedArticles = append(processedArticles, article)
			continue
		}
		log.Printf("[PARSE %d/%d] Text extracted successfully. Length: %d chars.", i+1, len(validURLs), len(article.RawText))

		article.SanitizedFileName = sanitizeFilename(u) // .txt is added by sanitizeFilename
		article.LocalPath = filepath.Join(tempContentPath, article.SanitizedFileName)

		log.Printf("[SAVE %d/%d] Saving extracted text to: %s...", i+1, len(validURLs), article.LocalPath)
		err = saveTextToFile(article.LocalPath, article.RawText)
		if err != nil {
			log.Printf("[ERROR %d/%d] Saving text for %s to %s: %v. Skipping.", i+1, len(validURLs), u, article.LocalPath, err)
			article.Error = err
			processedArticles = append(processedArticles, article)
			continue
		}
		log.Printf("[SAVE %d/%d] Saved extracted text successfully.", i+1, len(validURLs))

		log.Printf("[SUMMARIZE %d/%d] Summarizing content (Model: %s)...", i+1, len(validURLs), modelName)
		summary, err := llmclient.SummarizeText(geminiAPIKey, modelName, article.RawText)
		if err != nil {
			log.Printf("[ERROR %d/%d] Summarizing text for %s: %v", i+1, len(validURLs), u, err)
			article.Error = err
		} else {
			article.Summary = summary
			log.Printf("[SUMMARIZE %d/%d] Successfully summarized. Length: %d chars.", i+1, len(validURLs), len(summary))
		}
		processedArticles = append(processedArticles, article)
		log.Printf("[PROCESS %d/%d] Finished URL: %s", i+1, len(validURLs), u)
	}

	log.Println("-----------------------------------------------------")
	log.Println("📝 Phase 3: Final Digest Generation")
	log.Println("-----------------------------------------------------")
	log.Println("[DIGEST] Preparing content for final digest...")
	finalDigestPromptInput := prepareFinalDigestPrompt(processedArticles)

	if strings.TrimSpace(finalDigestPromptInput) == "" {
		log.Println("[INFO] No summaries available to generate a final digest. Exiting.")
		return
	}
	log.Printf("[DIGEST] Content prepared. Prompt length for final digest: %d chars.", len(finalDigestPromptInput))

	log.Printf("[DIGEST] Generating final digest (Model: %s)...", modelName)
	finalDigest, err := llmclient.GenerateFinalDigest(geminiAPIKey, modelName, finalDigestPromptInput)
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate final digest: %v", err)
	}
	log.Println("[DIGEST] Final digest generated successfully.")

	// --- Outputting and Saving Digest ---
	log.Println("-----------------------------------------------------")
	log.Println("📄 Phase 4: Outputting Digest")
	log.Println("-----------------------------------------------------")
	fmt.Println("\n✨ --- FINAL DIGEST --- ✨")
	fmt.Println(finalDigest)
	fmt.Println("✨ --- END OF DIGEST --- ✨")

	dateStr := time.Now().Format("2006-01-02")
	digestFilename := fmt.Sprintf("digest_%s.md", dateStr)
	digestFilepath := filepath.Join(outputDir, digestFilename)

	log.Printf("[SAVE_DIGEST] Saving final digest to: %s...", digestFilepath)
	err = ioutil.WriteFile(digestFilepath, []byte(finalDigest), 0644)
	if err != nil {
		log.Fatalf("[ERROR] Failed to save digest to %s: %v", digestFilepath, err)
	}
	log.Printf("[SAVE_DIGEST] Digest saved successfully to %s", digestFilepath)

	log.Println("-----------------------------------------------------")
	log.Println("✅ Briefly finished successfully! ✅")
	log.Println("-----------------------------------------------------")
}
