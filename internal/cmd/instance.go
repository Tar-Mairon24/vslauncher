package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Tar-Mairon24/vslauncher/internal/instance"
	"github.com/Tar-Mairon24/vslauncher/internal/versions"
)

var instanceCmd = &cobra.Command{
	Use:     "instance",
	Short:   "Manage Vintage Story instances",
	Aliases: []string{"inst", "i"},
	Long:    "Manage Vintage Story instances, including creating, listing, updating, and removing instances.",
}

func createInstanceCmd() *cobra.Command {
	var channel, version, customPath string
	var desktopFile bool

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new instance",
		Long:  "Create a new instance with the specified name, version, channel, and platform.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if channel == "" {
				channel = "stable"
			}

			releases, err := resolveReleases(channel, "asc", currentPlatform)
			if err != nil {
				return err
			}
			if version == "" {
				release, err := versions.FindLatestRelease(releases, currentPlatform)
				if err != nil {
					return err
				}
				version = release.Version
			}
			release, err := versions.FindRelease(releases, version, currentPlatform)
			if err != nil {
				return err
			}

			fmt.Printf("installing %s (%s) into instance %q \n", release.Version, release.Channel, name)

			inst, err := instance.Create(cmd.Context(), name, release, customPath, desktopFile)
			if err != nil {
				return err
			}

			fmt.Printf("instance %q created at %s\n", inst.Name, inst.Path)
			return nil
		},
	}

	cmd.Flags().StringVarP(&channel, "channel", "c", "", "Release channel (stable, unstable), defaults to stable")
	cmd.Flags().StringVarP(&version, "version", "v", "", "Version to install (optional, defaults to latest in channel)")
	cmd.Flags().StringVarP(&customPath, "path", "p", "", "Custom path for this particular instance outside the default location (optional)")
	cmd.Flags().BoolVarP(&desktopFile, "desktop-file", "d", false, "Create a desktop file for this instance (Linux only for now) (optional)")

	return cmd
}

func listInstancesCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all instances",
		Long:    "List all instances, including those in the default location and any custom paths.",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			instances, err := instance.List()
			if err != nil {
				return fmt.Errorf("listing instances: %w", err)
			}

			if len(instances) == 0 {
				fmt.Println("No instances found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			if verbose {
				fmt.Fprintln(w, "NAME\tVERSION\tCHANNEL\tPATH\tINSTALLED")
				for _, inst := range instances {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						inst.Name, inst.Version, inst.Channel, inst.Path,
						inst.InstalledAt.Format("2006-01-02 15:04"))
				}
			} else {
				fmt.Fprintln(w, "NAME\tVERSION\tCHANNEL")
				for _, inst := range instances {
					fmt.Fprintf(w, "%s\t%s\t%s\n", inst.Name, inst.Version, inst.Channel)
				}
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information about each instance")

	return cmd
}

func removeInstanceCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Short:   "Remove an instance",
		Long:    "Remove an instance by name, including its directory and registry entry.",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var confirmed bool
			var err error

			if !force {
				confirmed, err = confirm(fmt.Sprintf("Are you sure you want to remove the instance and all its data %q? This action cannot be undone ", name))
				if err != nil {
					return fmt.Errorf("confirming removal of instance %q: %w", name, err)
				}
				if !confirmed {
					fmt.Printf("instance %q removal cancelled.\n", name)
					return nil
				}
			}

			if err := instance.Remove(name); err != nil {
				return fmt.Errorf("removing instance %q: %w", name, err)
			}

			fmt.Printf("instance %q removed successfully\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal without confirmation")
	return cmd
}

func updateInstanceCmd() *cobra.Command {
	var channel, version string

	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update an instance to a new version",
		Long:  "Update an instance to a new version, leave empty to update to the latest version in the specified channel.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			releases, err := resolveReleases(channel, "asc", currentPlatform)
			if err != nil {
				return err
			}

			if version == "" {
				release, err := versions.FindLatestRelease(releases, currentPlatform)
				if err != nil {
					return err
				}
				version = release.Version
			}
			if channel == "" {
				channel = "stable"
			}

			release, err := versions.FindRelease(releases, version, currentPlatform)
			if err != nil {
				return fmt.Errorf("finding release %q for platform %q: %w", version, currentPlatform, err)
			}

			fmt.Printf("updating instance %q to version %s (%s)\n", name, release.Version, release.Channel)

			if err := instance.Update(name, release); err != nil {
				return fmt.Errorf("updating instance %q: %w", name, err)
			}

			fmt.Printf("instance %q updated successfully to version %s\n", name, release.Version)
			return nil
		},
	}

	cmd.Flags().StringVarP(&channel, "channel", "c", "", "Release channel (stable, unstable), defaults to stable")
	cmd.Flags().StringVarP(&version, "version", "v", "", "Version to update to (optional, defaults to latest in channel)")

	return cmd
}

func confirm(prompt string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s[y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading confirmation %w", err)
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes", nil
}

func init() {
	instanceCmd.AddCommand(createInstanceCmd())
	instanceCmd.AddCommand(listInstancesCmd())
	instanceCmd.AddCommand(updateInstanceCmd())
	instanceCmd.AddCommand(removeInstanceCmd())
	rootCmd.AddCommand(instanceCmd)
}
