package cmds

import "github.com/spf13/cobra"

func AddCommands(root *cobra.Command) error {
	root.AddCommand(newSmokeTestCmd())
	root.AddCommand(newSmokeTestSuperviseCmd())
	root.AddCommand(newSmokeTestE2ECmd())
	root.AddCommand(newSmokeTestLogsCmd())
	root.AddCommand(newSmokeTestFailuresCmd())
	root.AddCommand(newPlanCmd())
	root.AddCommand(newPluginsCmd())

	root.AddCommand(newUpCmd())
	root.AddCommand(newDownCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newLogsCmd())
	root.AddCommand(newTuiCmd())
	return nil
}
