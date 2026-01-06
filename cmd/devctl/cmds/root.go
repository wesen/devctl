package cmds

import "github.com/spf13/cobra"

func AddCommands(root *cobra.Command) error {
	root.AddCommand(newSmokeTestCmd())
	return nil
}
