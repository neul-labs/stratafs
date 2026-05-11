package cmd

import (
	"fmt"
	"os"

	"agentfs/pkg/config"
	"agentfs/pkg/database"
	"agentfs/pkg/fsbridge"

	"github.com/spf13/cobra"
)

var fsCmd = &cobra.Command{
	Use:   "fs",
	Short: "Filesystem operations",
	Long:  `Export or mount the AgentFS virtual filesystem.`,
}

var fsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export virtual filesystem to a directory",
	Long:  `Exports the indexed virtual filesystem into a local directory with metadata and chunk files.`,
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

		outputDir, _ := cmd.Flags().GetString("output")
		if outputDir == "" {
			outputDir = cfg.GlobalDir + "/export"
		}

		sources := cfg.GetEnabledSources()
		if len(sources) == 0 {
			return fmt.Errorf("no enabled sources found")
		}

		for _, source := range sources {
			dbPath := cfg.GetDBPathForSource(source)
			db, err := database.NewDB(dbPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to open database for source %s: %v\n", source.Name, err)
				continue
			}

			sourceOutputDir := outputDir + "/" + source.ID
			if err := fsbridge.ExportVirtualFS(db, source, sourceOutputDir); err != nil {
				db.Close()
				fmt.Fprintf(os.Stderr, "Warning: failed to export source %s: %v\n", source.Name, err)
				continue
			}
			db.Close()
			fmt.Printf("Exported source %s to %s\n", source.Name, sourceOutputDir)
		}

		return nil
	},
}

func init() {
	fsExportCmd.Flags().StringP("output", "o", "", "Output directory for export")
	fsCmd.AddCommand(fsExportCmd)
	rootCmd.AddCommand(fsCmd)
}
