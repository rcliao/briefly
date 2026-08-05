// Package llm wraps the Gemini API for text and image generation.
package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/viper"
	"google.golang.org/genai"
)

// DefaultModel is the default Gemini model to use for text generation.
const DefaultModel = "gemini-3.6-flash"

// Client represents a client for interacting with the Gemini API.
type Client struct {
	apiKey    string
	modelName string
	gClient   *genai.Client
}

// TextGenerationOptions contains options for text generation
type TextGenerationOptions struct {
	MaxTokens      int32         // Maximum number of tokens to generate
	Temperature    float32       // Temperature for randomness (0.0 to 1.0)
	Model          string        // Model to use (optional, defaults to client's model)
	ResponseSchema *genai.Schema // Optional: Schema for structured output
}

// NewClient creates a new LLM client.
// It supports multiple ways to get the API key (in order of preference):
// 1. Environment variable: GEMINI_API_KEY (or alternatives)
// 2. Viper configuration: gemini.api_key
func NewClient(modelName string) (*Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		if apiKey = os.Getenv("GOOGLE_GEMINI_API_KEY"); apiKey == "" {
			if apiKey = os.Getenv("GOOGLE_AI_API_KEY"); apiKey == "" {
				apiKey = viper.GetString("gemini.api_key")
			}
		}
	}
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is required. Set GEMINI_API_KEY environment variable or gemini.api_key in config file.\nGet your API key from: https://makersuite.google.com/app/apikey")
	}

	if modelName == "" {
		modelName = viper.GetString("gemini.model")
		if modelName == "" {
			modelName = DefaultModel
		}
	}

	gClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Client{
		apiKey:    apiKey,
		modelName: modelName,
		gClient:   gClient,
	}, nil
}

// Close releases client resources. The genai client needs no explicit close;
// this exists for symmetric defer usage.
func (c *Client) Close() {}

// GenerateText generates text using the LLM with specified options
// GenerateImage generates an image from a text prompt using a Gemini image
// model (e.g. gemini-3.1-flash-image). It returns the raw image bytes and
// their MIME type (the model chooses the format, typically image/jpeg).
func (c *Client) GenerateImage(ctx context.Context, model, prompt, aspectRatio string) ([]byte, string, error) {
	if prompt == "" {
		return nil, "", fmt.Errorf("prompt cannot be empty")
	}
	if model == "" {
		return nil, "", fmt.Errorf("image model cannot be empty")
	}

	contents := []*genai.Content{{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  "user",
	}}

	config := &genai.GenerateContentConfig{
		ResponseModalities: []string{"TEXT", "IMAGE"},
	}
	if aspectRatio != "" {
		config.ImageConfig = &genai.ImageConfig{AspectRatio: aspectRatio}
	}

	resp, err := c.gClient.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate image: %w", err)
	}

	for _, candidate := range resp.Candidates {
		if candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && len(part.InlineData.Data) > 0 {
				return part.InlineData.Data, part.InlineData.MIMEType, nil
			}
		}
	}

	return nil, "", fmt.Errorf("model returned no image data (possibly blocked by safety filters)")
}

func (c *Client) GenerateText(ctx context.Context, prompt string, options TextGenerationOptions) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	// Determine which model to use
	modelName := c.modelName
	if options.Model != "" {
		modelName = options.Model
	}

	// Build contents
	contents := []*genai.Content{{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  "user",
	}}

	// Build config if options are provided
	var config *genai.GenerateContentConfig
	if options.MaxTokens > 0 || options.Temperature > 0 || options.ResponseSchema != nil {
		config = &genai.GenerateContentConfig{}
		if options.MaxTokens > 0 {
			config.MaxOutputTokens = options.MaxTokens
		}
		if options.Temperature > 0 {
			temp := float32(options.Temperature)
			config.Temperature = &temp
		}
		// Phase 1: Structured output support
		if options.ResponseSchema != nil {
			config.ResponseMIMEType = "application/json"
			config.ResponseSchema = options.ResponseSchema
		}
	}

	// Generate content
	resp, err := c.gClient.Models.GenerateContent(ctx, modelName, contents, config)
	if err != nil {
		return "", fmt.Errorf("failed to generate text: %w", err)
	}

	// A MAX_TOKENS-truncated response is not a usable result: downstream
	// JSON parsing would fail or, worse, silently degrade the digest.
	if len(resp.Candidates) > 0 && resp.Candidates[0].FinishReason == "MAX_TOKENS" {
		return "", fmt.Errorf("response truncated at MaxTokens=%d (Gemini thinking tokens count against the budget; raise MaxTokens)", options.MaxTokens)
	}

	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	return text, nil
}
