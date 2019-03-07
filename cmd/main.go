// Package main provides cmd stas application
package main

import (
	"encoding/json"
	"fmt"

	"github.com/BurntSushi/toml"
	assembly "github.com/molecule-man/stack-assembly"
	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/cli/color"
	"github.com/molecule-man/stack-assembly/conf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	rootCmd := rootCmd()

	rootCmd.AddCommand(
		infoCmd(),
		syncCmd(),
		diffCmd(),
		deleteCmd(),
		dumpConfigCmd(),
	)

	assembly.MustSucceed(rootCmd.Execute())
}

func rootCmd() *cobra.Command {
	var nocolor bool

	rootCmd := &cobra.Command{
		Use: "stas",
	}
	rootCmd.PersistentFlags().StringP("profile", "p", "default", "AWS named profile")
	rootCmd.PersistentFlags().StringP("region", "r", "", "AWS region")

	rootCmd.PersistentFlags().StringSliceP("configs", "c", []string{},
		"Alternative config file(s). Default: stack-assembly.yaml")
	rootCmd.PersistentFlags().BoolVar(&nocolor, "nocolor", false,
		"Disables color output")
	rootCmd.PersistentFlags().BoolP("no-interaction", "n", false,
		"Do not ask any interactive questions")

	err := viper.BindPFlag("aws.profile", rootCmd.PersistentFlags().Lookup("profile"))
	assembly.MustSucceed(err)

	err = viper.BindPFlag("aws.region", rootCmd.PersistentFlags().Lookup("region"))
	assembly.MustSucceed(err)

	cobra.OnInitialize(func() {
		color.NoColor = nocolor
		err := viper.BindPFlags(rootCmd.PersistentFlags())
		assembly.MustSucceed(err)
	})

	return rootCmd
}

func infoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [stack name]",
		Short: "Show info about the stack",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			assembly.MustSucceed(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			assembly.MustSucceed(err)

			ss, err := cfg.StackConfigsSortedByExecOrder()
			assembly.MustSucceed(err)

			cf := conf.Cf(cfg)

			for _, s := range ss {
				if len(args) > 0 && args[0] != s.Name {
					continue
				}

				assembly.Info(awscf.NewStack(cf, s.Name))
			}
		},
	}
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "sync [ID]",
		Aliases: []string{"deploy"},
		Short:   "Synchronize (deploy) stacks",
		Long: `Creates or updates stacks specified in the config file(s).

By default sync command deploys all the stacks described in the config file(s).
To deploy a particular stack, ID argument has to be provided. ID is an
identifier of a stack within the config file. For example, ID is tpl1 in the
following yaml config:

    stacks:
      tpl1: # <--- this is ID
        name: mystack
        path: path/to/tpl.json`,

		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			assembly.MustSucceed(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			assembly.MustSucceed(err)

			if len(args) > 0 {
				id := args[0]
				stack, ok := cfg.Stacks[id]
				if !ok {
					foundIds := make([]string, 0, len(cfg.Stacks))
					for id := range cfg.Stacks {
						foundIds = append(foundIds, id)
					}

					assembly.MustSucceed(fmt.Errorf("ID %s is not found in the config. Found IDs: %v", id, foundIds))
				}
				cfg.Stacks = map[string]conf.Config{
					id: stack,
				}
			}

			nonInteractive, err := cmd.Parent().PersistentFlags().GetBool("no-interaction")
			assembly.MustSucceed(err)

			assembly.Sync(cfg, nonInteractive)
		},
	}
}

func diffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff [stack name]",
		Short: "Show diff of the stacks to be deployed",
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			assembly.MustSucceed(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			assembly.MustSucceed(err)

			cc, err := cfg.ChangeSets()
			assembly.MustSucceed(err)

			for _, c := range cc {
				diff, err := awscf.Diff(c)
				assembly.MustSucceed(err)

				cli.Print(diff)
			}
		},
	}
}

func deleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [stack name]",
		Short: "Deletes deployed stacks",
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			assembly.MustSucceed(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			assembly.MustSucceed(err)

			nonInteractive, err := cmd.Parent().PersistentFlags().GetBool("no-interaction")
			assembly.MustSucceed(err)

			assembly.Delete(cfg, nonInteractive)
		},
	}
}

func dumpConfigCmd() *cobra.Command {
	var format string
	dumpCmd := &cobra.Command{
		Use:   "dump-config",
		Short: "Dump loaded config into stdout",
		Run: func(cmd *cobra.Command, _ []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			assembly.MustSucceed(err)

			cfg, err := conf.LoadConfig(cfgFiles)
			assembly.MustSucceed(err)

			out := cli.Output

			switch format {
			case "yaml", "yml":
				assembly.MustSucceed(yaml.NewEncoder(out).Encode(cfg))
			case "json":
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				assembly.MustSucceed(enc.Encode(cfg))
			case "toml":
				assembly.MustSucceed(toml.NewEncoder(out).Encode(cfg))
			default:
				assembly.Terminate("unknown format: " + format)
			}
		},
	}
	dumpCmd.Flags().StringVarP(&format, "format", "f", "yaml", "One of: yaml, toml, json")

	return dumpCmd
}
