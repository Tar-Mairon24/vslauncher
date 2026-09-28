package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Tar-Mairon24/vslauncher/internal/instance"
	"github.com/Tar-Mairon24/vslauncher/internal/mods"
	"github.com/Tar-Mairon24/vslauncher/internal/utils/cmdprogressbar"
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

func updateModsCmd() *cobra.Command {
	var excludeFlag string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "update <instance-name> [mod-ids...]",
		Short: "Update mods installed for an instance (default: all mods)",
		Long:  "Update mods installed for an instance, optionally specifying which mods to update by their mod IDs (default: all mods).",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, err := instance.FindByName(args[0])
			if err != nil {
				return err
			}

			targetModsIDs := args[1:]

			excluded := inst.ExcludedFromUpdate
			if excludeFlag != "" {
				excluded = append(excluded, strings.Split(excludeFlag, ",")...)
			}

			installed, err := mods.ScanInstalled(inst.DataPath)
			if err != nil {
				return fmt.Errorf("scanning installed mods: %w", err)
			}

			availabeUpdates, err := mods.CheckForUpdates(cmd.Context(), installed, inst.Version, targetModsIDs, excluded)
			if err != nil {
				return fmt.Errorf("checking for updates: %w", err)
			}
			if len(availabeUpdates) == 0 {
				fmt.Println("No updates available for the installed mods.")
				return nil
			}

			if dryRun {
				for _, r := range availabeUpdates {
					if r.Available() && r.RecommendedUpgrade != "" {
						fmt.Printf("Mod %s has an available update: %s\n", r.Name, r.RecommendedUpgrade)
					}
				}
				return nil
			}

			modsDir := filepath.Join(inst.DataPath, "Mods")
			options := mods.UpdateModParams{
				Installed:      installed,
				GameVersion:    inst.Version,
				TargetModIDs:   targetModsIDs,
				ExcludedModIDs: excluded,
				ModsDir:        modsDir,
				DataPath:       inst.DataPath,
				InstName:       inst.Name,
				BackupDir:      defaultBackupDir(inst),
				MaxBackups:     5,
				NewProgress:    cmdprogressbar.NewMultiProgress(),
			}

			results, err := mods.UpdateMods(cmd.Context(), options)
			if err != nil {
				return fmt.Errorf("updating mods: %w", err)
			}
			if len(results) == 0 {
				fmt.Println("No updates available for the installed mods.")
				return nil
			}
			for _, res := range results {
				if res.Error != "" {
					fmt.Printf("%s: FAILED - %v\n", res.ModID, res.Error)
					continue
				}
				fmt.Printf("%s: %s -> %s\n", res.ModID, res.OldVersion, res.NewVersion)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&excludeFlag, "exclude", "", "Comma-separated list of mod IDs to exclude from updates")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run without actually updating the mods")

	return cmd
}

func excludeModsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exclude <instance-name> <mod-ids...>",
		Short: "Permanently exclude mods from being updated for an instance",
		Long:  "Permanently exclude specified mods from being updated for a given instance. This will add the mod IDs to the instance's exclusion list.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return instance.SetExcludedFromUpdate(args[0], args[1:])
		},
	}

	return cmd
}

func defaultBackupDir(inst *instance.Instance) string {
	return filepath.Join(inst.Path, "backups")
}

func init() {
	rootCmd.AddCommand(modsCmd)
	modsCmd.AddCommand(listModsCmd())
	modsCmd.AddCommand(checkModsCmd())
	modsCmd.AddCommand(updateModsCmd())
	modsCmd.AddCommand(excludeModsCmd())
}
