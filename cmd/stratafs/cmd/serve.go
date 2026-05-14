package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/monitor"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the StrataFS daemon",
	Long:  `Starts the StrataFS daemon including file monitoring, REST API, and MCP server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if configDir != "" {
			if err := os.Setenv("STRATAFS_GLOBAL_DIR", configDir); err != nil {
				return fmt.Errorf("failed to set config directory: %w", err)
			}
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		m, err := monitor.NewMonitor(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize monitor: %w", err)
		}

		if err := m.Start(); err != nil {
			return fmt.Errorf("failed to start monitor: %w", err)
		}

		fmt.Println("StrataFS daemon started. Press Ctrl+C to stop.")

		// Wait for interrupt signal
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\nShutting down StrataFS daemon...")
		m.Stop()
		fmt.Println("StrataFS daemon stopped.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
