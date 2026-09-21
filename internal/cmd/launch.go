package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Tar-Mairon24/vslauncher/internal/instance"
	"github.com/Tar-Mairon24/vslauncher/internal/launch"
)

func launchInstanceCmd() *cobra.Command {
	var options launch.Options
	cmd := &cobra.Command{
		Use:   "launch <name>",
		Short: "Launch an instance",
		Long:  "Launch a Vintage Story instance by name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			inst, err := instance.FindByName(name)
			if err != nil {
				return err
			}

			fmt.Printf("launching instance %q (%s)", inst.Name, inst.Version)
			if err := launch.LaunchInstance(cmd.Context(), inst, options); err != nil {
				return fmt.Errorf("launching instance %q: %w", name, err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&options.World, "world", "o", "", "Opens given world. If it doesn't exist it will be created. (optional)")
	cmd.Flags().StringVarP(&options.Connect, "connect", "c", "", "Connect to given server (optional)")
	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "Password for the server (if any) (optional)")
	cmd.Flags().BoolVarP(&options.RandomWorld, "random-world", "r", false, "Creates a new world with a random name. Use -p modifier to set playstyle (optional)")
	cmd.Flags().StringVarP(&options.PlayStyle, "play-style", "s", "", "(Default: creativebuilding) Used when creating a new world. Possible values are \"creativebuilding\", \"preset-surviveandbuild\", \"preset-exploration\", \"preset-homosapiens\" and \"preset-wildernesssurvival\" (optional)")
	cmd.Flags().StringVarP(&options.InstallMod, "install-mod", "i", "", "Install given mod in the format: modid@version (optional)")
	return cmd
}

func init() {
	rootCmd.AddCommand(launchInstanceCmd())
}