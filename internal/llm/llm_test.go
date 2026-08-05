package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestNewClient_Success(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-api-key")

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client.modelName != DefaultModel {
		t.Errorf("modelName = %q, want default %q", client.modelName, DefaultModel)
	}
}

func TestNewClient_NoAPIKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_AI_API_KEY", "")
	viper.Reset()

	_, err := NewClient("")
	if err == nil {
		t.Fatal("expected error when no API key is configured")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Errorf("error should mention GEMINI_API_KEY, got: %v", err)
	}
}

func TestNewClient_CustomModel(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-api-key")

	client, err := NewClient("custom-model")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client.modelName != "custom-model" {
		t.Errorf("modelName = %q, want custom-model", client.modelName)
	}
}

func TestGenerateText_EmptyPrompt(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-api-key")
	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if _, err := client.GenerateText(context.Background(), "", TextGenerationOptions{}); err == nil {
		t.Error("expected error for empty prompt")
	}
}

func TestGenerateImage_Validation(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-api-key")
	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if _, _, err := client.GenerateImage(context.Background(), "model", "", "16:9"); err == nil {
		t.Error("expected error for empty prompt")
	}
	if _, _, err := client.GenerateImage(context.Background(), "", "prompt", "16:9"); err == nil {
		t.Error("expected error for empty model")
	}
}
