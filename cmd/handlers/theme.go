package handlers

import (
	"briefly/internal/core"
	"briefly/internal/logger"
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// NewThemeCmd creates the theme management command
func NewThemeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "theme",
		Short: "Manage article classification themes",
		Long: `Manage themes for article classification.

Themes are categories that articles can be classified into based on their content.
Each theme has a name, description, and keywords that help the LLM classify articles.

Subcommands:
  add       Add a new theme
  remove    Remove a theme
  list      List all themes
  enable    Enable a theme
  disable   Disable a theme
  update    Update theme details`,
	}

	cmd.AddCommand(newThemeAddCmd())
	cmd.AddCommand(newThemeRemoveCmd())
	cmd.AddCommand(newThemeListCmd())
	cmd.AddCommand(newThemeEnableCmd())
	cmd.AddCommand(newThemeDisableCmd())
	cmd.AddCommand(newThemeUpdateCmd())

	return cmd
}

func newThemeAddCmd() *cobra.Command {
	var description string
	var keywords []string

	cmd := &cobra.Command{
		Use:   "add <theme-name>",
		Short: "Add a new theme",
		Long: `Add a new theme for article classification.

The theme name should be concise and descriptive. You can optionally provide:
  • A description explaining what articles belong to this theme
  • Keywords that help identify articles of this theme

Examples:
  briefly theme add "AI & Machine Learning" --description "Articles about artificial intelligence, ML models, and AI research" --keywords "AI,machine learning,neural networks,deep learning"
  briefly theme add "Cloud Infrastructure" --description "Cloud services, DevOps, and infrastructure topics" --keywords "AWS,Azure,GCP,Kubernetes,Docker"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			themeName := args[0]
			return runThemeAdd(cmd.Context(), themeName, description, keywords)
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Theme description")
	cmd.Flags().StringSliceVarP(&keywords, "keywords", "k", nil, "Comma-separated keywords")

	return cmd
}

func newThemeRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <theme-id>",
		Short: "Remove a theme",
		Long: `Remove a theme by ID.

Note: This will not delete the theme if there are articles already classified
under it. Instead, it will be marked as disabled.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			themeID := args[0]
			return runThemeRemove(cmd.Context(), themeID)
		},
	}
}

func newThemeListCmd() *cobra.Command {
	var showDisabled bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all themes",
		Long: `List all configured themes.

By default, only enabled themes are shown. Use --all to show disabled themes as well.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThemeList(cmd.Context(), showDisabled)
		},
	}

	cmd.Flags().BoolVar(&showDisabled, "all", false, "Show disabled themes as well")

	return cmd
}

func newThemeEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <theme-id>",
		Short: "Enable a theme",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			themeID := args[0]
			return runThemeToggle(cmd.Context(), themeID, true)
		},
	}
}

func newThemeDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <theme-id>",
		Short: "Disable a theme",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			themeID := args[0]
			return runThemeToggle(cmd.Context(), themeID, false)
		},
	}
}

func newThemeUpdateCmd() *cobra.Command {
	var description string
	var keywords []string

	cmd := &cobra.Command{
		Use:   "update <theme-id>",
		Short: "Update theme details",
		Long: `Update a theme's description or keywords.

Only the fields you specify will be updated. Use --description to update the description,
and --keywords to update the keyword list (this replaces the existing keywords).

Examples:
  briefly theme update abc123 --description "New description"
  briefly theme update abc123 --keywords "new,keywords,here"
  briefly theme update abc123 --description "New description" --keywords "new,keywords"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			themeID := args[0]
			return runThemeUpdate(cmd.Context(), themeID, description, keywords, cmd.Flags().Changed("description"), cmd.Flags().Changed("keywords"))
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "New theme description")
	cmd.Flags().StringSliceVarP(&keywords, "keywords", "k", nil, "New comma-separated keywords")

	return cmd
}

// Implementation functions

func runThemeAdd(ctx context.Context, name, description string, keywords []string) error {
	log := logger.Get()
	log.Info("Adding new theme", "name", name)

	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// Check if theme already exists
	existing, err := db.Themes().GetByName(ctx, name)
	if err == nil && existing != nil {
		return fmt.Errorf("theme '%s' already exists (ID: %s)", name, existing.ID)
	}

	theme := &core.Theme{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		Keywords:    keywords,
		Enabled:     true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := db.Themes().Create(ctx, theme); err != nil {
		return fmt.Errorf("failed to create theme: %w", err)
	}

	fmt.Println("✅ Theme added successfully")
	fmt.Printf("   ID:          %s\n", theme.ID)
	fmt.Printf("   Name:        %s\n", theme.Name)
	if theme.Description != "" {
		fmt.Printf("   Description: %s\n", theme.Description)
	}
	if len(theme.Keywords) > 0 {
		fmt.Printf("   Keywords:    %s\n", strings.Join(theme.Keywords, ", "))
	}
	fmt.Printf("   Status:      Enabled\n")

	return nil
}

func runThemeRemove(ctx context.Context, themeID string) error {
	log := logger.Get()
	log.Info("Removing theme", "id", themeID)

	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// Check if theme exists
	theme, err := db.Themes().Get(ctx, themeID)
	if err != nil {
		return fmt.Errorf("theme not found: %w", err)
	}

	if err := db.Themes().Delete(ctx, themeID); err != nil {
		return fmt.Errorf("failed to remove theme: %w", err)
	}

	fmt.Println("✅ Theme removed successfully")
	fmt.Printf("   Name: %s\n", theme.Name)

	return nil
}

func runThemeList(ctx context.Context, showDisabled bool) error {
	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	themes, err := db.Themes().List(ctx, !showDisabled)
	if err != nil {
		return fmt.Errorf("failed to list themes: %w", err)
	}

	if len(themes) == 0 {
		fmt.Println("No themes found")
		fmt.Println("\nAdd your first theme:")
		fmt.Println("  briefly theme add <theme-name> --description \"...\" --keywords \"...\"")
		return nil
	}

	// Display themes in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tName\tKeywords\tEnabled\n")
	fmt.Fprintf(w, "━━━━━━━━━━\t━━━━━━━━━━━━━━━━━━━━\t━━━━━━━━━━━━━━━━━━━━\t━━━━━━━\n")

	for _, theme := range themes {
		status := "✓"
		if !theme.Enabled {
			status = "✗"
		}

		nameShort := theme.Name
		if len(nameShort) > 30 {
			nameShort = nameShort[:27] + "..."
		}

		keywordsShort := strings.Join(theme.Keywords, ", ")
		if len(keywordsShort) > 40 {
			keywordsShort = keywordsShort[:37] + "..."
		}
		if len(theme.Keywords) == 0 {
			keywordsShort = "(none)"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			theme.ID[:8]+"...", nameShort, keywordsShort, status,
		)
	}
	w.Flush()

	fmt.Printf("\nTotal themes: %d\n", len(themes))
	if !showDisabled {
		fmt.Println("Use --all to show disabled themes")
	}

	return nil
}

func runThemeToggle(ctx context.Context, themeID string, enabled bool) error {
	log := logger.Get()
	action := "Enabling"
	if !enabled {
		action = "Disabling"
	}
	log.Info(action+" theme", "id", themeID)

	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	theme, err := db.Themes().Get(ctx, themeID)
	if err != nil {
		return fmt.Errorf("theme not found: %w", err)
	}

	theme.Enabled = enabled
	theme.UpdatedAt = time.Now().UTC()

	if err := db.Themes().Update(ctx, theme); err != nil {
		return fmt.Errorf("failed to update theme: %w", err)
	}

	status := "enabled"
	if !enabled {
		status = "disabled"
	}
	fmt.Printf("✅ Theme '%s' %s\n", theme.Name, status)
	return nil
}

func runThemeUpdate(ctx context.Context, themeID, description string, keywords []string, updateDesc, updateKeywords bool) error {
	log := logger.Get()
	log.Info("Updating theme", "id", themeID)

	if !updateDesc && !updateKeywords {
		return fmt.Errorf("no updates specified (use --description or --keywords)")
	}

	db, err := getDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	theme, err := db.Themes().Get(ctx, themeID)
	if err != nil {
		return fmt.Errorf("theme not found: %w", err)
	}

	if updateDesc {
		theme.Description = description
	}
	if updateKeywords {
		theme.Keywords = keywords
	}
	theme.UpdatedAt = time.Now().UTC()

	if err := db.Themes().Update(ctx, theme); err != nil {
		return fmt.Errorf("failed to update theme: %w", err)
	}

	fmt.Println("✅ Theme updated successfully")
	fmt.Printf("   Name:        %s\n", theme.Name)
	if updateDesc {
		fmt.Printf("   Description: %s\n", theme.Description)
	}
	if updateKeywords {
		fmt.Printf("   Keywords:    %s\n", strings.Join(theme.Keywords, ", "))
	}

	return nil
}
