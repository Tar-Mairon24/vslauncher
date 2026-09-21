package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Tar-Mairon24/vslauncher/internal/instance"
	"github.com/Tar-Mairon24/vslauncher/internal/launch"
)

func launchInstanceCmd() *cobra.Command {
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
			if err := launch.LaunchInstance(cmd.Context(), inst); err != nil {
				return fmt.Errorf("launching instance %q: %w", name, err)
			}
			return nil
		},
	}
	return cmd
}

func init() {
	rootCmd.AddCommand(launchInstanceCmd())
}