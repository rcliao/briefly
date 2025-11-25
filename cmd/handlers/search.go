package handlers

import (
	"briefly/internal/config"
	"briefly/internal/llm"
	"briefly/internal/persistence"
	"briefly/internal/vectorstore"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewSearchCmd creates the parent search command with subcommands
func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Semantic search for articles",
		Long: `Search for articles using semantic similarity with pgvector HNSW index.

Subcommands:
  query   - Search articles by text query
  similar - Find articles similar to a specific article
  stats   - Show vector store statistics

Examples:
  # Search by text
  briefly search query "artificial intelligence trends"

  # Find similar articles
  briefly search similar abc123de

  # Show vector store stats
  briefly search stats`,
	}

	// Add subcommands
	cmd.AddCommand(NewSearchQueryCmd())
	cmd.AddCommand(NewSearchSimilarCmd())
	cmd.AddCommand(NewSearchStatsCmd())

	return cmd
}

// NewSearchQueryCmd creates the query subcommand
func NewSearchQueryCmd() *cobra.Command {
	var (
		limit     int
		threshold float64
	)

	cmd := &cobra.Command{
		Use:   "query [text]",
		Short: "Search articles by text query",
		Long:  `Search for articles semantically similar to a text query`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearchQuery(args, limit, threshold)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of results")
	cmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.5, "Minimum similarity threshold (0.0-1.0)")

	return cmd
}

// NewSearchSimilarCmd creates the similar subcommand
func NewSearchSimilarCmd() *cobra.Command {
	var (
		limit     int
		threshold float64
	)

	cmd := &cobra.Command{
		Use:   "similar [article-id]",
		Short: "Find articles similar to a specific article",
		Long:  `Find articles semantically similar to a specific article by ID`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearchSimilar(args[0], limit, threshold)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of results")
	cmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.5, "Minimum similarity threshold (0.0-1.0)")

	return cmd
}

// NewSearchStatsCmd creates the stats subcommand
func NewSearchStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show vector store statistics",
		Long:  `Display statistics about the vector store (embeddings, index, etc.)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearchStats()
		},
	}
}

func runSearchQuery(args []string, limit int, threshold float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Load config
	_, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	// Get database connection string
	dbConnStr := cfg.Database.ConnectionString
	if dbConnStr == "" {
		dbConnStr = os.Getenv("DATABASE_URL")
		if dbConnStr == "" {
			return fmt.Errorf("database connection string not configured")
		}
	}

	// Connect to database
	db, err := persistence.NewPostgresDB(dbConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Initialize LLM client
	llmClient, err := llm.NewClient(cfg.AI.Gemini.APIKey)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	// Initialize vector store
	vectorStore := vectorstore.NewPgVectorAdapter(db.GetDB())

	// Generate embedding for query
	queryText := strings.Join(args, " ")

	fmt.Printf("🔍 Searching for: \"%s\"\n", queryText)
	fmt.Printf("   Generating embedding...\n")

	embedding, err := llmClient.GenerateEmbedding(queryText)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	fmt.Printf("   ✓ Embedding generated (%d dimensions)\n", len(embedding))
	fmt.Printf("   Searching vector store (threshold: %.2f, limit: %d)...\n\n", threshold, limit)

	// Search for similar articles
	query := vectorstore.SearchQuery{
		Embedding:           embedding,
		Limit:               limit,
		SimilarityThreshold: threshold,
		IncludeArticle:      true,
	}

	results, err := vectorStore.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	// Display results
	if len(results) == 0 {
		fmt.Println("❌ No similar articles found")
		fmt.Printf("   Try lowering the threshold (current: %.2f)\n", threshold)
		return nil
	}

	fmt.Printf("✨ Found %d similar articles:\n\n", len(results))

	// Display results in a readable format
	for i, result := range results {
		fmt.Printf("[%d] %.3f similarity - %s\n", i+1, result.Similarity, result.Article.Title)
		fmt.Printf("    ID: %s\n", result.ArticleID)
		if result.Article.URL != "" {
			fmt.Printf("    URL: %s\n", result.Article.URL)
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Printf("💡 Use 'briefly read <url>' to view full article\n")
	fmt.Printf("💡 Use 'briefly search similar <article-id>' to find related articles\n")

	return nil
}

func runSearchSimilar(articleID string, limit int, threshold float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Load config
	_, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	// Get database connection string
	dbConnStr := cfg.Database.ConnectionString
	if dbConnStr == "" {
		dbConnStr = os.Getenv("DATABASE_URL")
		if dbConnStr == "" {
			return fmt.Errorf("database connection string not configured")
		}
	}

	// Connect to database
	db, err := persistence.NewPostgresDB(dbConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Initialize vector store
	vectorStore := vectorstore.NewPgVectorAdapter(db.GetDB())

	fmt.Printf("🔍 Finding articles similar to: %s\n", articleID)
	fmt.Printf("   Loading article embedding...\n")

	// Get article repository
	articleRepo := db.Articles()

	// Get article to retrieve embedding
	article, err := articleRepo.Get(ctx, articleID)
	if err != nil {
		return fmt.Errorf("failed to get article: %w", err)
	}

	if article.Embedding == nil || len(article.Embedding) == 0 {
		return fmt.Errorf("article %s has no embedding", articleID)
	}

	fmt.Printf("   ✓ Article: \"%s\"\n", article.Title)
	fmt.Printf("   ✓ Embedding loaded (%d dimensions)\n", len(article.Embedding))
	fmt.Printf("   Searching vector store (threshold: %.2f, limit: %d)...\n\n", threshold, limit)

	// Search for similar articles
	query := vectorstore.SearchQuery{
		Embedding:           article.Embedding,
		Limit:               limit + 1, // +1 to account for self-match
		SimilarityThreshold: threshold,
		IncludeArticle:      true,
		ExcludeIDs:          []string{articleID}, // Exclude the query article itself
	}

	results, err := vectorStore.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	// Display results
	if len(results) == 0 {
		fmt.Println("❌ No similar articles found")
		fmt.Printf("   Try lowering the threshold (current: %.2f)\n", threshold)
		return nil
	}

	fmt.Printf("✨ Found %d similar articles:\n\n", len(results))

	// Display results in a readable format
	for i, result := range results {
		fmt.Printf("[%d] %.3f similarity - %s\n", i+1, result.Similarity, result.Article.Title)
		fmt.Printf("    ID: %s\n", result.ArticleID)
		if result.Article.URL != "" {
			fmt.Printf("    URL: %s\n", result.Article.URL)
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Printf("💡 Use 'briefly read <url>' to view full article\n")

	return nil
}

func runSearchStats() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load config
	_, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	// Get database connection string
	dbConnStr := cfg.Database.ConnectionString
	if dbConnStr == "" {
		dbConnStr = os.Getenv("DATABASE_URL")
		if dbConnStr == "" {
			return fmt.Errorf("database connection string not configured")
		}
	}

	// Connect to database
	db, err := persistence.NewPostgresDB(dbConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Initialize vector store
	vectorStore := vectorstore.NewPgVectorAdapter(db.GetDB())

	fmt.Println("📊 Vector Store Statistics")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	stats, err := vectorStore.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("Total Embeddings:     %d\n", stats.TotalEmbeddings)
	fmt.Printf("Embedding Dimensions: %d\n", stats.EmbeddingDimensions)
	fmt.Printf("Index Type:           %s\n", stats.IndexType)
	fmt.Printf("Index Size:           %s\n", stats.IndexSize)

	if stats.AvgSearchLatency > 0 {
		fmt.Printf("Avg Search Latency:   %.2fms\n", stats.AvgSearchLatency)
	}

	fmt.Println()

	return nil
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
