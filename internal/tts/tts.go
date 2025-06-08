package tts

import (
	"briefly/internal/render"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TTSProvider represents different TTS service providers
type TTSProvider string

const (
	ProviderElevenLabs TTSProvider = "elevenlabs"
	ProviderOpenAI     TTSProvider = "openai"
	ProviderGoogle     TTSProvider = "google"
	ProviderMock       TTSProvider = "mock"
)

// TTSVoice represents voice configuration
type TTSVoice struct {
	ID     string
	Name   string
	Gender string
	Accent string
}

// TTSConfig holds TTS configuration
type TTSConfig struct {
	Provider   TTSProvider
	APIKey     string
	Voice      TTSVoice
	Speed      float64 // 0.5 - 2.0
	Pitch      float64 // -1.0 - 1.0 (for some providers)
	OutputDir  string
	HTTPClient *http.Client
}

// ElevenLabsVoiceResponse represents ElevenLabs voice API response
type ElevenLabsVoiceResponse struct {
	Voices []struct {
		VoiceID string `json:"voice_id"`
		Name    string `json:"name"`
		Gender  string `json:"gender"`
		Accent  string `json:"accent"`
	} `json:"voices"`
}

// ElevenLabsTTSRequest represents ElevenLabs TTS request
type ElevenLabsTTSRequest struct {
	Text          string                  `json:"text"`
	ModelID       string                  `json:"model_id"`
	VoiceSettings ElevenLabsVoiceSettings `json:"voice_settings"`
}

// ElevenLabsVoiceSettings represents voice settings for ElevenLabs
type ElevenLabsVoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style,omitempty"`
	UseInterpolate  bool    `json:"use_speaker_boost,omitempty"`
}

// OpenAITTSRequest represents OpenAI TTS request
type OpenAITTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
}

// TTSClient handles text-to-speech operations
type TTSClient struct {
	Config *TTSConfig
}

// NewTTSClient creates a new TTS client
func NewTTSClient(config *TTSConfig) *TTSClient {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}
	if config.OutputDir == "" {
		config.OutputDir = "audio"
	}
	if config.Speed == 0 {
		config.Speed = 1.0
	}

	return &TTSClient{
		Config: config,
	}
}

// GetDefaultVoices returns default voices for each provider
func GetDefaultVoices() map[TTSProvider][]TTSVoice {
	return map[TTSProvider][]TTSVoice{
		ProviderElevenLabs: {
			{ID: "21m00Tcm4TlvDq8ikWAM", Name: "Rachel", Gender: "Female", Accent: "American"},
			{ID: "AZnzlk1XvdvUeBnXmlld", Name: "Domi", Gender: "Female", Accent: "American"},
			{ID: "EXAVITQu4vr4xnSDxMaL", Name: "Bella", Gender: "Female", Accent: "American"},
			{ID: "ErXwobaYiN019PkySvjV", Name: "Antoni", Gender: "Male", Accent: "American"},
			{ID: "VR6AewLTigWG4xSOukaG", Name: "Arnold", Gender: "Male", Accent: "American"},
		},
		ProviderOpenAI: {
			{ID: "alloy", Name: "Alloy", Gender: "Neutral", Accent: "American"},
			{ID: "echo", Name: "Echo", Gender: "Male", Accent: "American"},
			{ID: "fable", Name: "Fable", Gender: "Male", Accent: "British"},
			{ID: "onyx", Name: "Onyx", Gender: "Male", Accent: "American"},
			{ID: "nova", Name: "Nova", Gender: "Female", Accent: "American"},
			{ID: "shimmer", Name: "Shimmer", Gender: "Female", Accent: "American"},
		},
		ProviderGoogle: {
			{ID: "en-US-Wavenet-D", Name: "Wavenet-D", Gender: "Male", Accent: "American"},
			{ID: "en-US-Wavenet-C", Name: "Wavenet-C", Gender: "Female", Accent: "American"},
			{ID: "en-GB-Wavenet-A", Name: "Wavenet-A", Gender: "Female", Accent: "British"},
			{ID: "en-GB-Wavenet-B", Name: "Wavenet-B", Gender: "Male", Accent: "British"},
		},
	}
}

// PrepareTTSText converts digest content to TTS-friendly text
func PrepareTTSText(digestItems []render.DigestData, title string, includeSummaries bool, maxArticles int) string {
	var text strings.Builder

	// Introduction
	text.WriteString(fmt.Sprintf("Welcome to your %s. ", title))
	text.WriteString(fmt.Sprintf("Here are the highlights from %d articles. ", len(digestItems)))
	text.WriteString("\n\n")

	// Limit articles for audio length
	itemCount := len(digestItems)
	if maxArticles > 0 && itemCount > maxArticles {
		itemCount = maxArticles
		text.WriteString(fmt.Sprintf("We'll cover the top %d articles. ", maxArticles))
	}

	// Process articles
	for i := 0; i < itemCount; i++ {
		item := digestItems[i]

		// Article number and title
		text.WriteString(fmt.Sprintf("Article %d: %s. ", i+1, cleanTextForTTS(item.Title)))

		// Summary if included
		if includeSummaries && item.SummaryText != "" {
			summary := cleanTextForTTS(item.SummaryText)
			// Limit summary length for audio
			if len(summary) > 300 {
				summary = summary[:297] + "..."
			}
			text.WriteString(summary)
			text.WriteString(". ")
		}

		text.WriteString("\n\n")
	}

	// Conclusion
	if itemCount < len(digestItems) {
		text.WriteString(fmt.Sprintf("That covers the top %d articles. ", itemCount))
		text.WriteString(fmt.Sprintf("There are %d more articles in the full digest. ", len(digestItems)-itemCount))
	}
	text.WriteString("Thank you for listening to your Briefly digest. ")

	return text.String()
}

// cleanTextForTTS removes markdown formatting and makes text more speech-friendly
func cleanTextForTTS(text string) string {
	// Remove markdown formatting
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "`", "")
	text = strings.ReplaceAll(text, "#", "")

	// Remove URLs (basic pattern)
	words := strings.Fields(text)
	var cleanWords []string
	for _, word := range words {
		if !strings.HasPrefix(word, "http://") && !strings.HasPrefix(word, "https://") {
			cleanWords = append(cleanWords, word)
		}
	}
	text = strings.Join(cleanWords, " ")

	// Replace common symbols with words
	text = strings.ReplaceAll(text, "&", "and")
	text = strings.ReplaceAll(text, "@", "at")
	text = strings.ReplaceAll(text, "%", "percent")
	text = strings.ReplaceAll(text, "$", "dollars")

	// Add pauses for better speech flow
	text = strings.ReplaceAll(text, ". ", ". ... ")
	text = strings.ReplaceAll(text, "? ", "? ... ")
	text = strings.ReplaceAll(text, "! ", "! ... ")

	return strings.TrimSpace(text)
}

// GenerateAudio creates an audio file from text using the configured TTS provider
func (c *TTSClient) GenerateAudio(text string, filename string) (string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(c.Config.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Add .mp3 extension if not present
	if !strings.HasSuffix(filename, ".mp3") {
		filename = strings.TrimSuffix(filename, ".wav") + ".mp3"
	}

	outputPath := filepath.Join(c.Config.OutputDir, filename)

	switch c.Config.Provider {
	case ProviderElevenLabs:
		return c.generateElevenLabsAudio(text, outputPath)
	case ProviderOpenAI:
		return c.generateOpenAIAudio(text, outputPath)
	case ProviderGoogle:
		return c.generateGoogleAudio(text, outputPath)
	case ProviderMock:
		return c.generateMockAudio(text, outputPath)
	default:
		return "", fmt.Errorf("unsupported TTS provider: %s", c.Config.Provider)
	}
}

// generateElevenLabsAudio generates audio using ElevenLabs API
func (c *TTSClient) generateElevenLabsAudio(text string, outputPath string) (string, error) {
	if c.Config.APIKey == "" {
		return "", fmt.Errorf("ElevenLabs API key is required")
	}

	voiceID := c.Config.Voice.ID
	if voiceID == "" {
		voiceID = "21m00Tcm4TlvDq8ikWAM" // Default Rachel voice
	}

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", voiceID)

	requestData := ElevenLabsTTSRequest{
		Text:    text,
		ModelID: "eleven_monolingual_v1",
		VoiceSettings: ElevenLabsVoiceSettings{
			Stability:       0.5,
			SimilarityBoost: 0.5,
		},
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "audio/mpeg")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", c.Config.APIKey)

	resp, err := c.Config.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ElevenLabs API error %d: %s", resp.StatusCode, string(body))
	}

	// Write audio data to file
	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	return outputPath, nil
}

// generateOpenAIAudio generates audio using OpenAI TTS API
func (c *TTSClient) generateOpenAIAudio(text string, outputPath string) (string, error) {
	if c.Config.APIKey == "" {
		return "", fmt.Errorf("OpenAI API key is required")
	}

	voice := c.Config.Voice.ID
	if voice == "" {
		voice = "alloy" // Default voice
	}

	url := "https://api.openai.com/v1/audio/speech"

	requestData := OpenAITTSRequest{
		Model:          "tts-1",
		Input:          text,
		Voice:          voice,
		ResponseFormat: "mp3",
		Speed:          c.Config.Speed,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Config.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(body))
	}

	// Write audio data to file
	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	return outputPath, nil
}

// generateGoogleAudio generates audio using Google Cloud TTS API
func (c *TTSClient) generateGoogleAudio(text string, outputPath string) (string, error) {
	// Google Cloud TTS implementation would go here
	// For now, return an error indicating it's not implemented
	return "", fmt.Errorf("google Cloud TTS not implemented in this version")
}

// generateMockAudio creates a mock audio file for testing
func (c *TTSClient) generateMockAudio(text string, outputPath string) (string, error) {
	// Create a simple text file instead of actual audio for testing
	mockContent := fmt.Sprintf("Mock TTS Audio File\n\nGenerated: %s\nText Length: %d characters\nVoice: %s\n\nText Content:\n%s",
		time.Now().Format(time.RFC3339),
		len(text),
		c.Config.Voice.Name,
		text)

	// Change extension to .txt for mock
	outputPath = strings.TrimSuffix(outputPath, ".mp3") + "_mock.txt"

	err := os.WriteFile(outputPath, []byte(mockContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write mock audio file: %w", err)
	}

	return outputPath, nil
}

// GetAvailableProviders returns available TTS providers
func GetAvailableProviders() []string {
	return []string{
		string(ProviderElevenLabs),
		string(ProviderOpenAI),
		string(ProviderGoogle),
		string(ProviderMock),
	}
}

// ValidateConfig validates TTS configuration
func ValidateConfig(config *TTSConfig) error {
	if config.Provider == "" {
		return fmt.Errorf("TTS provider is required")
	}

	providers := GetAvailableProviders()
	validProvider := false
	for _, provider := range providers {
		if string(config.Provider) == provider {
			validProvider = true
			break
		}
	}
	if !validProvider {
		return fmt.Errorf("invalid TTS provider: %s (available: %s)",
			config.Provider, strings.Join(providers, ", "))
	}

	if config.Provider != ProviderMock && config.APIKey == "" {
		return fmt.Errorf("%s requires an API key", config.Provider)
	}

	if config.Speed < 0.5 || config.Speed > 2.0 {
		return fmt.Errorf("speed must be between 0.5 and 2.0")
	}

	return nil
}

// EstimateAudioLength estimates audio length in minutes based on text
func EstimateAudioLength(text string, speed float64) float64 {
	// Average speaking rate is about 150-160 words per minute
	// Adjust for speed setting
	wordsPerMinute := 155.0 * speed

	words := len(strings.Fields(text))
	minutes := float64(words) / wordsPerMinute

	return minutes
}
