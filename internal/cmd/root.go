package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Tar-Mairon24/vslauncher/internal/platform"
)

var rootCmd = &cobra.Command{
	Use:   "vsl",
	Short: "Vintage Story Launcher CLI",
	Long:  `A command-line interface for the Vintage Story Launcher`,
	SilenceErrors: true,
	SilenceUsage:  true,
	DisableAutoGenTag: true,
	Version: "0.1.0",
}

var currentPlatform string

func Execute() {
	p, err := platform.Detect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting platform: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Detected platform: %s\n", p)
	currentPlatform = p

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}
