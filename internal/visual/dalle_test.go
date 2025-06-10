package visual

import (
	"encoding/json"
	"testing"
)

func TestDALLERequestFormat(t *testing.T) {
	// Test that the request format matches the latest API specification
	request := DALLERequest{
		Model:  "gpt-image-1",
		Prompt: "A test image prompt",
		N:      1,
		Size:   "1024x1024",
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	expected := `{"model":"gpt-image-1","prompt":"A test image prompt","n":1,"size":"1024x1024"}`
	actual := string(jsonData)

	if actual != expected {
		t.Errorf("Request format mismatch.\nExpected: %s\nActual: %s", expected, actual)
	}
}

func TestDALLEResponseFormat(t *testing.T) {
	// Test that the response format can handle the latest API response
	responseJSON := `{
		"created": 1234567890,
		"data": [
			{
				"b64_json": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9jU8=",
				"revised_prompt": "A revised test prompt"
			}
		],
		"usage": {
			"total_tokens": 100,
			"input_tokens": 10,
			"output_tokens": 90,
			"input_tokens_details": {
				"text_tokens": 8,
				"image_tokens": 2
			}
		}
	}`

	var response DALLEResponse
	err := json.Unmarshal([]byte(responseJSON), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify response structure
	if response.Created != 1234567890 {
		t.Errorf("Expected created=1234567890, got %d", response.Created)
	}

	if len(response.Data) != 1 {
		t.Errorf("Expected 1 data item, got %d", len(response.Data))
	}

	if response.Data[0].B64JSON == "" {
		t.Error("Expected b64_json to be set")
	}

	if response.Data[0].RevisedPrompt != "A revised test prompt" {
		t.Errorf("Expected revised_prompt='A revised test prompt', got '%s'", response.Data[0].RevisedPrompt)
	}

	// Verify usage information
	if response.Usage == nil {
		t.Error("Expected usage information to be present")
	} else {
		if response.Usage.TotalTokens != 100 {
			t.Errorf("Expected total_tokens=100, got %d", response.Usage.TotalTokens)
		}
		if response.Usage.InputTokens != 10 {
			t.Errorf("Expected input_tokens=10, got %d", response.Usage.InputTokens)
		}
		if response.Usage.OutputTokens != 90 {
			t.Errorf("Expected output_tokens=90, got %d", response.Usage.OutputTokens)
		}

		if response.Usage.InputTokensDetails == nil {
			t.Error("Expected input_tokens_details to be present")
		} else {
			if response.Usage.InputTokensDetails.TextTokens != 8 {
				t.Errorf("Expected text_tokens=8, got %d", response.Usage.InputTokensDetails.TextTokens)
			}
			if response.Usage.InputTokensDetails.ImageTokens != 2 {
				t.Errorf("Expected image_tokens=2, got %d", response.Usage.InputTokensDetails.ImageTokens)
			}
		}
	}
}

func TestGetImageSize(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		expected string
	}{
		{
			name:     "Landscape (16:9-ish)",
			width:    1792,
			height:   1024,
			expected: "1792x1024",
		},
		{
			name:     "Portrait",
			width:    1024,
			height:   1792,
			expected: "1024x1792",
		},
		{
			name:     "Square",
			width:    1024,
			height:   1024,
			expected: "1024x1024",
		},
		{
			name:     "Wide landscape",
			width:    1920,
			height:   1080,
			expected: "1792x1024", // Closest supported size
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetImageSize(tt.width, tt.height)
			if result != tt.expected {
				t.Errorf("GetImageSize(%d, %d) = %s, expected %s", tt.width, tt.height, result, tt.expected)
			}
		})
	}
}