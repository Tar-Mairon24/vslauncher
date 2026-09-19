package cmd

import 
(
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vsl",
	Short: "Vintage Story Launcher CLI",
	Long:  `A command-line interface for the Vintage Story Launcher`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}