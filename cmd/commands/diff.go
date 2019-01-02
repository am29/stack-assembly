package commands

import (
	"fmt"

	"github.com/molecule-man/stack-assembly/cmd/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/cobra"
)

func diffCmd() *cobra.Command {
	diffCmd := &cobra.Command{
		Use:   "diff [stack name]",
		Short: "Show diff of the stacks to be deployed",
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			handleError(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			handleError(err)

			diffS := stackassembly.DiffService{
				Dp: conf.Aws(cfg),
			}

			stacks, err := cfg.GetStacks()
			handleError(err)

			for _, stack := range stacks {
				diff, err := diffS.Diff(stack)
				handleError(err)

				fmt.Println(diff)
			}
		},
	}

	return diffCmd
}
