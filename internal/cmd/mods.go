package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Tar-Mairon24/vslauncher/internal/instance"
	"github.com/Tar-Mairon24/vslauncher/internal/mods"
)

var modsCmd = &cobra.Command{
	Use:     "mods",
	Short:   "Manage mods for Vintage Story instances",
	Aliases: []string{"mod"},
	Long:    "Manage mods for Vintage Story instances, including listing installed mods and checking for updates.",
}

func listModsCmd() *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:   "list <instance_name>",
		Short: "List installed mods for an instance",
		Long:  "List all installed mods for a specified instance, including their names, versions, and authors.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, err := instance.FindByName(args[0])
			if err != nil {
				return err
			}

			installed, err := mods.ScanInstalled(inst.DataPath)
			if err != nil {
				return fmt.Errorf("scanning mods for instance %q: %w", inst.Name, err)
			}
			if len(installed) == 0 {
				fmt.Printf("No mods installed for instance %q.\n", inst.Name)
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			if verbose {
				fmt.Fprintln(w, "MOD NAME\tVERSION\tMOD IDENTIFIER\tAUTHORS\tSIDE\tREQUIRED ON CLIENT\tREQUIRED ON SERVER")
				for _, mod := range installed {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%t\t%t\n",
						mod.Info.Name, mod.Info.Version, mod.Info.QueryID(), strings.Join(mod.Info.Authors, ", "),
						mod.Info.Side, mod.Info.RequiredOnClient, mod.Info.RequiredOnServer)
				}
			} else {
				fmt.Fprintln(w, "MOD NAME\tVERSION\tAUTHORS\tSIDE\tENABLED")
				for _, mod := range installed {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\n",
						mod.Info.Name, mod.Info.Version, strings.Join(mod.Info.Authors, ", "), mod.Info.Side, mod.Enabled)
				}
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information about each mod")

	return cmd
}

func checkModsCmd() *cobra.Command {
	var verbose, all bool
	cmd := &cobra.Command{
		Use:   "check <instance-name>",
		Short: "Check install info for an instance's mods",
		Long:  "Check install info for an instance's mods, including whether updates are available.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, err := instance.FindByName(args[0])
			if err != nil {
				return err
			}
			installed, err := mods.ScanInstalled(inst.DataPath)
			if err != nil {
				return fmt.Errorf("scanning installed mods: %w", err)
			}

			ids := make([]string, len(installed))
			for i, mod := range installed {
				ids[i] = mod.Info.QueryID()
			}

			result, err := mods.FetchInstallInfo(cmd.Context(), ids, inst.Version)
			if err != nil {
				return fmt.Errorf("fetching install info: %w", err)
			}
			if len(result) == 0 {
				fmt.Println("No install info available for the installed mods.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			if verbose {
				fmt.Fprintln(w, "MOD NAME\tFILE NAME\tFILE URL\tRECOMMENDED UPGRADE")
				for _, info := range result {
					if info.RecommendedUpgrade != "" {
						fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.Name, info.FileName, info.FileURL, info.RecommendedUpgrade)
					} else if all {
						fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.Name, "Up to date", info.FileURL, info.RecommendedUpgrade)
					}
				}
			} else {
				fmt.Fprintln(w, "MOD NAME\tRECOMMENDED UPGRADE")
				for _, info := range result {
					if info.RecommendedUpgrade != "" {
						fmt.Fprintf(w, "%s\t%s\n", info.Name, info.RecommendedUpgrade)
					} else if all {
						fmt.Fprintf(w, "%s\t%s\n", info.Name, "Up to date")
					}
				}
			}

			return w.Flush()
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information about each mod")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show all mods, including those without updates")

	return cmd
}

func init() {
	rootCmd.AddCommand(modsCmd)
	modsCmd.AddCommand(listModsCmd())
	modsCmd.AddCommand(checkModsCmd())
}
