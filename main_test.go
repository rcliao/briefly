package main

import (
	"reflect"
	"testing"
)

func TestExtractURLs(t *testing.T) {
	testCases := []struct {
		name            string
		markdownContent string
		expectedURLs    []string
		expectError     bool
	}{
		{
			name:            "No URLs",
			markdownContent: "This is a test string with no URLs.",
			expectedURLs:    []string{},
			expectError:     false,
		},
		{
			name:            "Single URL",
			markdownContent: "Check out https://example.com for more info.",
			expectedURLs:    []string{"https://example.com"},
			expectError:     false,
		},
		{
			name:            "Multiple URLs",
			markdownContent: "Here are two URLs: http://example.org and https://another.example.net/path?query=value.",
			expectedURLs:    []string{"http://example.org", "https://another.example.net/path?query=value"},
			expectError:     false,
		},
		{
			name:            "URLs mixed with markdown links",
			markdownContent: "A link [example](https://example.com) and a plain URL http://example.org.",
			expectedURLs:    []string{"https://example.com", "http://example.org"},
			expectError:     false,
		},
		{
			name:            "URL at the beginning of a line",
			markdownContent: "https://example.com is a great site.",
			expectedURLs:    []string{"https://example.com"},
			expectError:     false,
		},
		{
			name:            "URL at the end of a line",
			markdownContent: "A great site is https://example.com",
			expectedURLs:    []string{"https://example.com"},
			expectError:     false,
		},
		{
			name:            "URL with parentheses in markdown link",
			markdownContent: "([link](https://example.com/page_(foo)))",
			expectedURLs:    []string{"https://example.com/page_(foo)"},
			expectError:     false,
		},
		{
			name:            "URL with no trailing slash",
			markdownContent: "https://example.com/path",
			expectedURLs:    []string{"https://example.com/path"},
			expectError:     false,
		},
		{
			name:            "Empty content",
			markdownContent: "",
			expectedURLs:    []string{},
			expectError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			urls, err := extractURLs(tc.markdownContent)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error, but got: %v", err)
				}
				if !reflect.DeepEqual(urls, tc.expectedURLs) {
					t.Errorf("Expected URLs %v, but got %v", tc.expectedURLs, urls)
				}
			}
		})
	}
}
