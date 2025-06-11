package visual

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DALLEClient handles OpenAI DALL-E API interactions
type DALLEClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewDALLEClient creates a new DALL-E API client
func NewDALLEClient(apiKey string) *DALLEClient {
	return &DALLEClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.openai.com/v1",
	}
}

// DALLERequest represents a DALL-E image generation request (latest API)
type DALLERequest struct {
	Model  string `json:"model"` // Should be "gpt-image-1"
	Prompt string `json:"prompt"`
	N      int    `json:"n"`    // Number of images to generate
	Size   string `json:"size"` // Image size like "1024x1024"
}

// DALLEResponse represents a DALL-E API response (latest API)
type DALLEResponse struct {
	Created int64              `json:"created"`
	Data    []DALLEImageResult `json:"data"`
	Usage   *DALLEUsage        `json:"usage,omitempty"`
}

// DALLEImageResult represents a single image result from DALL-E
type DALLEImageResult struct {
	B64JSON       string `json:"b64_json"`      // Base64 encoded image
	URL           string `json:"url,omitempty"` // Image URL (if response_format is url)
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// DALLEUsage represents token usage information
type DALLEUsage struct {
	TotalTokens        int                      `json:"total_tokens"`
	InputTokens        int                      `json:"input_tokens"`
	OutputTokens       int                      `json:"output_tokens"`
	InputTokensDetails *DALLEInputTokensDetails `json:"input_tokens_details,omitempty"`
}

// DALLEInputTokensDetails represents detailed input token breakdown
type DALLEInputTokensDetails struct {
	TextTokens  int `json:"text_tokens"`
	ImageTokens int `json:"image_tokens"`
}

// GenerateImage generates an image using DALL-E API (latest version)
func (c *DALLEClient) GenerateImage(ctx context.Context, prompt string, size string, quality string) (*DALLEResponse, error) {
	request := DALLERequest{
		Model:  "gpt-image-1", // Latest image generation model
		Prompt: prompt,
		N:      1,    // Generate one image
		Size:   size, // Image size like "1024x1024"
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/images/generations", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DALL-E API error (status %d): %s", resp.StatusCode, string(body))
	}

	var dalleResp DALLEResponse
	if err := json.Unmarshal(body, &dalleResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &dalleResp, nil
}

// SaveBase64Image saves a base64 encoded image to the specified path
func (c *DALLEClient) SaveBase64Image(ctx context.Context, base64Data, outputPath string) error {
	// Decode base64 image data
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write image data to file
	err = os.WriteFile(outputPath, imageData, 0644)
	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	return nil
}

// DownloadImage downloads an image from a URL and saves it to the specified path (legacy method)
func (c *DALLEClient) DownloadImage(ctx context.Context, imageURL, outputPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	return nil
}

// GetImageSize maps banner config size to DALL-E API size format
func GetImageSize(width, height int) string {
	// DALL-E API supports: '1024x1024', '1024x1536', '1536x1024', '1792x1024', '1024x1792'
	// Choose best fit based on requested dimensions and aspect ratio

	// Check for exact matches first
	sizeStr := fmt.Sprintf("%dx%d", width, height)
	switch sizeStr {
	case "1024x1024", "1024x1536", "1536x1024", "1792x1024", "1024x1792":
		return sizeStr
	}

	// For non-exact matches, find the closest supported size
	if width == height {
		return "1024x1024" // Square
	}

	if width > height {
		// Landscape orientation
		aspectRatio := float64(width) / float64(height)
		if aspectRatio >= 1.7 { // Wide landscape (16:9 or wider)
			return "1792x1024" // 1.75 aspect ratio
		}
		return "1536x1024" // 1.5 aspect ratio
	} else {
		// Portrait orientation
		aspectRatio := float64(height) / float64(width)
		if aspectRatio >= 1.7 { // Tall portrait
			return "1024x1792" // 1:1.75 aspect ratio
		}
		return "1024x1536" // 1:1.5 aspect ratio
	}
}
