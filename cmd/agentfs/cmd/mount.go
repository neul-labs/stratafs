package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"agentfs/pkg/config"
	"agentfs/pkg/database"
	"agentfs/pkg/fsbridge"

	"github.com/spf13/cobra"
)

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount AgentFS as a FUSE filesystem",
	Long:  `Mounts the AgentFS virtual filesystem at the specified mount point using FUSE.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("FUSE mount is not supported on Windows")
		}

		mountPoint, _ := cmd.Flags().GetString("mount-point")
		if mountPoint == "" {
			return fmt.Errorf("--mount-point is required")
		}

		if configDir != "" {
			if err := os.Setenv("AGENTFS_GLOBAL_DIR", configDir); err != nil {
				return fmt.Errorf("failed to set config directory: %w", err)
			}
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		sources := cfg.GetEnabledSources()
		if len(sources) == 0 {
			return fmt.Errorf("no enabled sources found")
		}

		// Use the first enabled source for mounting
		source := sources[0]
		dbPath := cfg.GetDBPathForSource(source)
		db, err := database.NewDB(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		readOnly, _ := cmd.Flags().GetBool("read-only")
		showChunks, _ := cmd.Flags().GetBool("show-chunks")
		showMetadata, _ := cmd.Flags().GetBool("show-metadata")

		opts := fsbridge.MountOptions{
			MountPoint:   mountPoint,
			ReadOnly:     readOnly,
			ShowChunks:   showChunks,
			ShowMetadata: showMetadata,
		}

		mount := fsbridge.NewFuseMount(db, source, opts)
		fmt.Printf("Mounting AgentFS at %s (Ctrl+C to unmount)...\n", mountPoint)
		if err := mount.Mount(); err != nil {
			return fmt.Errorf("failed to mount: %w", err)
		}

		// Wait for interrupt
		fmt.Println("Filesystem mounted. Press Ctrl+C to unmount.")
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch

		fmt.Println("\nUnmounting...")
		if err := mount.Unmount(); err != nil {
			return fmt.Errorf("failed to unmount: %w", err)
		}
		fmt.Println("Unmounted successfully.")
		return nil
	},
}

func init() {
	mountCmd.Flags().StringP("mount-point", "m", "", "Mount point directory (required)")
	mountCmd.Flags().Bool("read-only", true, "Mount as read-only")
	mountCmd.Flags().Bool("show-chunks", false, "Expose _chunks directories")
	mountCmd.Flags().Bool("show-metadata", false, "Expose metadata.json files")
	rootCmd.AddCommand(mountCmd)
}
