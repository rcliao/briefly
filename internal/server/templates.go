package server

import (
	"fmt"
	"html/template"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TemplateRenderer manages HTML templates with hot-reload support
type TemplateRenderer struct {
	templates *template.Template
	mu        sync.RWMutex
	devMode   bool
	templateDir string
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer(devMode bool, templateDir string) (*TemplateRenderer, error) {
	if templateDir == "" {
		templateDir = "web/templates"
	}

	tr := &TemplateRenderer{
		devMode:     devMode,
		templateDir: templateDir,
	}

	if err := tr.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return tr, nil
}

// loadTemplates parses all HTML templates from the template directory
func (tr *TemplateRenderer) loadTemplates() error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	// Custom template functions
	funcMap := template.FuncMap{
		"truncate":         truncateString,
		"formatDate":       formatDate,
		"formatDateShort":  formatDateShort,
		"readTime":         calculateReadTime,
		"themeEmoji":       getThemeEmoji,
		"extractDomain":    extractDomain,
		"add":              func(a, b int) int { return a + b },
		"sub":              func(a, b int) int { return a - b },
		"mul":              func(a, b float64) float64 { return a * b },
		"div":              func(a, b int) int { if b == 0 { return 0 }; return a / b },
		"eq":               func(a, b interface{}) bool { return a == b },
		"ne":               func(a, b interface{}) bool { return a != b },
		"gt":               func(a, b int) bool { return a > b },
		"lt":               func(a, b int) bool { return a < b },
		"len":              func(s interface{}) int {
			switch v := s.(type) {
			case []interface{}:
				return len(v)
			case string:
				return len(v)
			default:
				return 0
			}
		},
	}

	// Create new template with functions
	tmpl := template.New("").Funcs(funcMap)

	// Walk the template directory and parse all .html files
	err := filepath.WalkDir(tr.templateDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-HTML files
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Read template file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		// Template name is relative path from template directory
		name := strings.TrimPrefix(path, tr.templateDir+"/")

		// Parse template
		_, err = tmpl.New(name).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	tr.templates = tmpl
	return nil
}

// Render executes a template with the given data
func (tr *TemplateRenderer) Render(w io.Writer, name string, data interface{}) error {
	// In dev mode, reload templates on each request
	if tr.devMode {
		if err := tr.loadTemplates(); err != nil {
			return fmt.Errorf("failed to reload templates: %w", err)
		}
	}

	tr.mu.RLock()
	defer tr.mu.RUnlock()

	if tr.templates == nil {
		return fmt.Errorf("templates not loaded")
	}

	err := tr.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		return fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return nil
}

// Helper functions for templates

// truncateString truncates a string to the specified length and adds "..."
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// formatDate formats a time.Time as "Jan 2, 2006"
func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006")
}

// formatDateShort formats a time.Time as "Jan 2"
func formatDateShort(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 2")
}

// calculateReadTime estimates reading time based on word count (200 words/min)
func calculateReadTime(text string) int {
	if text == "" {
		return 0
	}
	words := len(strings.Fields(text))
	minutes := int(math.Ceil(float64(words) / 200.0))
	if minutes < 1 {
		return 1
	}
	return minutes
}

// getThemeEmoji returns an emoji for a given theme name
func getThemeEmoji(themeName string) string {
	emojis := map[string]string{
		"AI/ML":                    "🤖",
		"Artificial Intelligence":  "🤖",
		"Machine Learning":         "🧠",
		"Cloud Computing":          "☁️",
		"Cloud":                    "☁️",
		"DevOps":                   "🚀",
		"Security":                 "🔒",
		"Cybersecurity":            "🛡️",
		"Frontend":                 "🎨",
		"Backend":                  "⚙️",
		"Data Science":             "📊",
		"Data":                     "📊",
		"Mobile Development":       "📱",
		"Mobile":                   "📱",
		"Web Development":          "🌐",
		"Blockchain":               "⛓️",
		"IoT":                      "📡",
		"Quantum Computing":        "⚛️",
		"Open Source":              "🔓",
		"Software Engineering":     "💻",
		"Programming":              "👨‍💻",
		"Databases":                "🗄️",
		"Networking":               "🌐",
		"Infrastructure":           "🏗️",
		"Containerization":         "🐳",
		"Microservices":            "🔧",
	}

	// Try exact match first
	if emoji, ok := emojis[themeName]; ok {
		return emoji
	}

	// Try case-insensitive partial match
	lowerTheme := strings.ToLower(themeName)
	for key, emoji := range emojis {
		if strings.Contains(lowerTheme, strings.ToLower(key)) {
			return emoji
		}
	}

	// Default emoji
	return "📰"
}

// extractDomain extracts the domain from a URL
func extractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	domain := parsedURL.Host

	// Remove "www." prefix
	domain = strings.TrimPrefix(domain, "www.")

	return domain
}
