package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Tar-Mairon24/vslauncher/internal/instance"
	"github.com/Tar-Mairon24/vslauncher/internal/versions"
)

var instanceCmd = &cobra.Command{
	Use:   "instance",
	Short: "Manage Vintage Story instances",
}

func createInstanceCmd() *cobra.Command {
	var channel, version, platform string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new instance",
		Long:  "Create a new instance with the specified name, version, channel, and platform.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			releases, err := resolveReleases(channel, "asc", platform)
			if err != nil {
				return err
			}
			release, err := versions.FindRelease(releases, version, platform)
			if err != nil {
				return err
			}

			instancesRoot, err := defaultInstancesRoot()
			if err != nil {
				return err
			}

			fmt.Printf("installing %s (%s) into instance %q \n", release.Version, release.Channel, name)

			inst, err := instance.Create(cmd.Context(), instancesRoot, name, release)
			if err != nil {
				return err
			}

			fmt.Printf("instance %q created at %s\n", inst.Name, inst.Path)
			return nil
		},
	}

	cmd.Flags().StringVarP(&channel, "channel", "c", "", "Release channel (stable, unstable)")
	cmd.Flags().StringVarP(&version, "version", "v", "", "Version to install (required)")
	cmd.Flags().StringVarP(&platform, "platform", "p", "", "Platform to install (default: current platform)")
	cmd.MarkFlagRequired("version")

	return cmd
}

func defaultInstancesRoot() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting user home directory: %w", err)
		}
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	return filepath.Join(dataHome, "vslauncher", "instances"), nil
}

func init() {
	instanceCmd.AddCommand(createInstanceCmd())
	rootCmd.AddCommand(instanceCmd)
}
