package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"agentfs/pkg/config"
	"agentfs/pkg/database"
	"agentfs/pkg/embeddings"
	"agentfs/pkg/search"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search indexed files",
	Long:  `Performs a hybrid search across all indexed files.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if configDir != "" {
			if err := os.Setenv("AGENTFS_GLOBAL_DIR", configDir); err != nil {
				return fmt.Errorf("failed to set config directory: %w", err)
			}
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize embedder
		embedder, err := embeddings.NewEmbedder(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize embedder: %w", err)
		}

		// Open databases for all enabled sources
		databases := make(map[string]*database.DB)
		for _, source := range cfg.GetEnabledSources() {
			dbPath := cfg.GetDBPathForSource(source)
			db, err := database.NewDB(dbPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to open database for source %s: %v\n", source.Name, err)
				continue
			}
			defer db.Close()
			databases[source.Path] = db
		}

		if len(databases) == 0 {
			return fmt.Errorf("no databases available for search")
		}

		// Initialize search engine
		engine, err := search.NewEngine(databases, embedder)
		if err != nil {
			return fmt.Errorf("failed to initialize search engine: %w", err)
		}

		mode := search.SearchModeHybrid
		modeFlag, _ := cmd.Flags().GetString("mode")
		switch modeFlag {
		case "fulltext":
			mode = search.SearchModeFullText
		case "vector":
			mode = search.SearchModeVector
		}

		limit, _ := cmd.Flags().GetInt("limit")
		includeContent, _ := cmd.Flags().GetBool("content")

		req := &search.SearchRequest{
			Query:          args[0],
			Mode:           mode,
			Limit:          limit,
			IncludeContent: includeContent,
			Weights:        search.DefaultWeights(),
		}

		resp, err := engine.Search(req)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		fmt.Printf("Found %d results in %s (query: %s, mode: %s)\n\n", resp.Total, resp.TimeTaken, resp.Query, resp.Mode)
		for i, r := range resp.Results {
			fmt.Printf("%d. %s (score: %.4f)\n", i+1, r.FilePath, r.Score)
			if r.Snippet != "" {
				fmt.Printf("   %s\n", r.Snippet)
			}
			if includeContent && r.Content != "" {
				fmt.Printf("   Content: %s\n", r.Content)
			}
			fmt.Println()
		}

		if cmd.Flags().Changed("json") {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		}

		return nil
	},
}

func init() {
	searchCmd.Flags().StringP("mode", "m", "hybrid", "Search mode: hybrid, fulltext, vector")
	searchCmd.Flags().IntP("limit", "n", 10, "Maximum number of results")
	searchCmd.Flags().BoolP("content", "c", false, "Include full content in results")
	searchCmd.Flags().Bool("json", false, "Output results as JSON")
	rootCmd.AddCommand(searchCmd)
}
