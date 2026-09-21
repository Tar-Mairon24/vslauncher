package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Tar-Mairon24/vslauncher/internal/platform"
)

var rootCmd = &cobra.Command{
	Use:   "vsl",
	Short: "Vintage Story Launcher CLI",
	Long:  `A command-line interface for the Vintage Story Launcher`,
	SilenceErrors: true,
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
	currentPlatform = p

	rootCmd.SetUsageTemplate(strings.ReplaceAll(rootCmd.UsageTemplate(), "Flags:", "Options:"))

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}
