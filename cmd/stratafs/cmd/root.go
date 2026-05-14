package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version   string
	BuildTime string
	configDir string
)

var rootCmd = &cobra.Command{
	Use:   "stratafs",
	Short: "StrataFS - A semantic filesystem for AI agents",
	Long: `StrataFS transforms passive file storage into an intelligent,
searchable knowledge base with REST API and Model Context Protocol support.`,
}

func Execute(version, buildTime string) {
	Version = version
	BuildTime = buildTime
	rootCmd.Version = fmt.Sprintf("%s (built: %s)", version, buildTime)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "Configuration directory (default: ~/.stratafs)")
}
