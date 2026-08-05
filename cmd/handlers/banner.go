package handlers

import (
	"briefly/internal/config"
	"briefly/internal/llm"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// bannerPromptCount is how many subject variants the LLM proposes; the user
// picks one interactively (default: the first).
const bannerPromptCount = 3

// NewBannerCmd creates the standalone banner command for (re)generating a
// banner image from an existing digest file.
func NewBannerCmd() *cobra.Command {
	var promptOverride string

	cmd := &cobra.Command{
		Use:   "banner <digest.md>",
		Short: "Generate a LinkedIn banner image for a digest",
		Long: `Generate a banner image for an existing digest file.

Reads the digest content (including any hand edits), proposes image prompt
subjects, lets you pick one, and generates a pixel-art banner PNG next to
the digest.

Examples:
  # Regenerate the banner for a digest
  briefly banner digests/digest_2026-08-05.md

  # Skip prompt suggestions and use your own
  briefly banner digests/digest_2026-08-05.md --prompt "a robot mailman sorting glowing letters"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			digestPath := args[0]
			content, err := os.ReadFile(digestPath)
			if err != nil {
				return fmt.Errorf("failed to read digest: %w", err)
			}

			if _, err := config.Load(cfgFile); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			cfg := config.Get()

			llmClient, err := llm.NewClient(cfg.AI.Gemini.Model)
			if err != nil {
				return fmt.Errorf("failed to create LLM client: %w", err)
			}
			defer llmClient.Close()

			// Banner assets live next to the digest file (dated-folder layout
			// means simple names; legacy flat files get "banner" beside them too)
			outDir := filepath.Dir(digestPath)

			return generateBanner(cmd.Context(), llmClient, cfg, string(content), outDir, "banner", promptOverride)
		},
	}

	cmd.Flags().StringVar(&promptOverride, "prompt", "", "Use this prompt subject directly (skips suggestion step)")

	return cmd
}

// generateBanner runs the full banner flow: propose prompt subjects, let the
// user pick one, generate the image, and write PNG + prompt sidecar.
// Failures never propagate as hard errors from the digest pipeline; the
// caller decides whether to treat the returned error as fatal.
func generateBanner(ctx context.Context, llmClient *llm.Client, cfg *config.Config, digestContent, outDir, baseName, promptOverride string) error {
	style := cfg.Banner.Style
	aspect := cfg.Banner.AspectRatio
	model := cfg.Banner.Model

	subject := promptOverride
	if subject == "" {
		subjects, err := generateBannerSubjects(ctx, llmClient, digestContent)
		if err != nil {
			return fmt.Errorf("failed to generate banner prompts: %w", err)
		}
		subject = selectBannerSubject(subjects, os.Stdin)
	}

	fullPrompt := subject + ", " + style

	fmt.Printf("   🎨 Generating banner (%s)...\n", model)
	fmt.Printf("      Prompt: %s\n", fullPrompt)

	imageBytes, mimeType, err := llmClient.GenerateImage(ctx, model, fullPrompt, aspect)
	if err != nil {
		// Still leave the prompt behind so the manual path is one paste away
		promptPath := filepath.Join(outDir, baseName+".prompt.txt")
		_ = os.WriteFile(promptPath, []byte(fullPrompt+"\n"), 0644)
		return fmt.Errorf("image generation failed (prompt saved to %s): %w", promptPath, err)
	}

	ext := ".png"
	if strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg") {
		ext = ".jpg"
	}
	imagePath := filepath.Join(outDir, baseName+ext)
	if err := os.WriteFile(imagePath, imageBytes, 0644); err != nil {
		return fmt.Errorf("failed to write banner image: %w", err)
	}

	promptPath := filepath.Join(outDir, baseName+".prompt.txt")
	if err := os.WriteFile(promptPath, []byte(fullPrompt+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write prompt file: %w", err)
	}

	fmt.Printf("   ✓ Banner saved: %s\n", imagePath)
	fmt.Printf("   ✓ Prompt saved: %s\n", promptPath)

	return nil
}

// generateBannerSubjects asks the LLM for visual subject ideas based on the
// digest content. The style half of the prompt is fixed configuration; the
// model only invents the subject.
func generateBannerSubjects(ctx context.Context, llmClient *llm.Client, digestContent string) ([]string, error) {
	if len(digestContent) > 4000 {
		digestContent = digestContent[:4000]
	}

	prompt := fmt.Sprintf(`You write image-generation prompt subjects for a weekly tech digest's LinkedIn banner.

Based on the digest below, propose %d different visual SUBJECTS that capture this week's dominant theme.

RULES for each subject:
- One concrete visual scene or metaphor, not an abstract concept. Good: "a tiny robot passing a glowing envelope between two towering server racks". Bad: "the concept of AI communication".
- Under 25 words.
- No text, letters, numbers, logos, or UI screenshots in the scene.
- Simple enough to read clearly as a small pixel-art image.

Respond with ONLY a JSON object: {"subjects": ["...", "...", "..."]}

DIGEST:

%s`, bannerPromptCount, digestContent)

	// Generous cap: Gemini 3.x thinking tokens count against MaxTokens, and a
	// truncated response here means unparseable JSON.
	raw, err := llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		Temperature: 0.9,
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, err
	}

	// Models sometimes wrap the JSON in prose or fences; extract the object.
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end <= start {
		return nil, fmt.Errorf("prompt response contains no JSON object: %.120q", raw)
	}

	var parsed struct {
		Subjects []string `json:"subjects"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &parsed); err != nil {
		return nil, fmt.Errorf("prompt response is not valid JSON: %w", err)
	}
	var subjects []string
	for _, s := range parsed.Subjects {
		if strings.TrimSpace(s) != "" {
			subjects = append(subjects, strings.TrimSpace(s))
		}
	}
	if len(subjects) == 0 {
		return nil, fmt.Errorf("no prompt subjects returned")
	}
	return subjects, nil
}

// selectBannerSubject shows the proposed subjects and reads a 1-based choice
// from in. On empty input, invalid input, or a non-interactive stdin, it
// defaults to the first subject.
func selectBannerSubject(subjects []string, in *os.File) string {
	if len(subjects) == 1 {
		return subjects[0]
	}

	fmt.Println("\n🎨 Banner prompt options:")
	for i, s := range subjects {
		fmt.Printf("   [%d] %s\n", i+1, s)
	}

	if !isInteractive(in) {
		fmt.Println("   (non-interactive: using option 1)")
		return subjects[0]
	}

	fmt.Printf("   Pick one [1-%d, Enter=1]: ", len(subjects))
	scanner := bufio.NewScanner(in)
	if scanner.Scan() {
		choice := strings.TrimSpace(scanner.Text())
		if choice >= "1" && choice <= fmt.Sprintf("%d", len(subjects)) && len(choice) == 1 {
			return subjects[int(choice[0]-'0')-1]
		}
	}
	return subjects[0]
}

// isInteractive reports whether f is a terminal (as opposed to a pipe/file).
func isInteractive(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
