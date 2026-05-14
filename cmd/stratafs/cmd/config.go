package cmd

import (
	"fmt"
	"os"

	"github.com/neul-labs/stratafs/pkg/config"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage StrataFS configuration",
	Long:  `Initialize or inspect StrataFS configuration.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize StrataFS configuration",
	Long:  `Creates the default configuration file and directories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if configDir != "" {
			if err := os.Setenv("STRATAFS_GLOBAL_DIR", configDir); err != nil {
				return fmt.Errorf("failed to set config directory: %w", err)
			}
		}

		cfg := config.DefaultConfig()
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("Configuration initialized at: %s\n", cfg.GlobalDir)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}
