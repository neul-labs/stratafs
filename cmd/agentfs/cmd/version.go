package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print AgentFS version",
	Long:  `Displays the current version and build time of AgentFS.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("AgentFS version %s (built: %s)\n", Version, BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
