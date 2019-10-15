package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	assembly "github.com/molecule-man/stack-assembly"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/conf"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

type Commands struct {
	SA             *assembly.SA
	Cli            *cli.CLI
	CfgLoader      *conf.Loader
	cfg            *conf.Config
	nonInteractive *bool
}

func (c *Commands) RootCmd() *cobra.Command {
	nonInteractive := false
	c.nonInteractive = &nonInteractive

	c.cfg = &conf.Config{
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Capabilities: []string{},
	}

	defaultProfile := "default"
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		defaultProfile = profile
	}

	rootCmd := &cobra.Command{
		Use: "stas <stack name> <template path>",
	}
	rootCmd.PersistentFlags().StringVarP(&c.cfg.Settings.Aws.Profile, "profile", "p", defaultProfile, "AWS named profile")
	rootCmd.PersistentFlags().StringVarP(&c.cfg.Settings.Aws.Region, "region", "r", os.Getenv("AWS_REGION"), "AWS region")

	// rootCmd.PersistentFlags().StringSliceVarP(&c.cfgFiles, "configs", "c", []string{},
	// 	"Alternative config file(s). Default: stack-assembly.yaml")
	rootCmd.PersistentFlags().BoolVar(&c.Cli.Color.Disabled, "nocolor", false,
		"Disables color output")
	rootCmd.PersistentFlags().BoolVarP(c.nonInteractive, "no-interaction", "n", false,
		"Do not ask any interactive questions")

	rootCmd.PersistentFlags().StringToStringVarP(&c.cfg.Parameters, "var", "v", map[string]string{},
		"Additional variables to use as parameters in config.\nExample: -v myParam=someValue")

	rootCmd.AddCommand(
		c.infoCmd(),
		c.syncCmd(),
		c.deployCmd(),
		c.diffCmd(),
		c.deleteCmd(),
		c.dumpConfigCmd(),
	)

	return rootCmd
}

func (c Commands) infoCmd() *cobra.Command {
	cfgFiles := []string{}
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show info about the stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.CfgLoader.LoadConfig(cfgFiles, c.cfg); err != nil {
				return err
			}

			return c.SA.InfoAll(*c.cfg)
		},
	}

	addConfigFlag(cmd, &cfgFiles)

	return cmd
}

func (c Commands) deployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy <stack name> <template path>",
		Args:  cobra.ExactArgs(2),
		Short: "Deploys single cloudformation template",
		RunE: func(cmd *cobra.Command, args []string) error {
			c.cfg.Name = args[0]
			c.cfg.Path = args[1]

			if err := c.CfgLoader.InitConfig(c.cfg); err != nil {
				return err
			}
			return c.SA.Sync(*c.cfg, *c.nonInteractive)
		},
	}

	cmd.Flags().StringSliceVar(&c.cfg.Capabilities, "capabilities", c.cfg.Capabilities,
		"A list of capabilities that you must specify before AWS\nCloudformation can create certain stacks. E.g. CAPABILITY_IAM")
	cmd.Flags().StringToStringVar(&c.cfg.Tags, "tags", c.cfg.Tags, "A list of tags to associate with the stack that is deployed")

	return cmd
}

func (c Commands) syncCmd() *cobra.Command {
	cfgFiles := []string{}
	cmd := &cobra.Command{
		Use:   "sync [<ID> [<ID> ...]]",
		Short: "Deploy stacks using the config file(s)",
		Long: `Creates or updates stacks specified in the config file(s).

By default sync command deploys all the stacks described in the config file(s).
To deploy a particular stack, ID argument has to be provided. ID is an
identifier of a stack within the config file. For example, ID is tpl1 in the
following yaml config:

    stacks:
      tpl1: # <--- this is ID
        name: mystack
        path: path/to/tpl.json

The config can be nested:
    stacks:
      parent_tpl:
        name: my-parent-stack
        path: path/to/tpl.json
        stacks:
          child_tpl: # <--- this is ID of the stack we want to deploy
            name: my-child-stack
            path: path/to/tpl.json

In this case specifying ID of only wanted stack is not enough all the parent IDs
have to be specified as well:

  stas sync parent_tpl child_tpl`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.CfgLoader.LoadConfig(cfgFiles, c.cfg); err != nil {
				return err
			}

			for _, id := range args {
				stack, ok := c.cfg.Stacks[id]
				if !ok {
					foundIds := make([]string, 0, len(c.cfg.Stacks))
					for id := range c.cfg.Stacks {
						foundIds = append(foundIds, id)
					}

					assembly.MustSucceed(fmt.Errorf("ID %s is not found in the config. Found IDs: %v", id, foundIds))
				}

				*c.cfg = stack
			}

			return c.SA.Sync(*c.cfg, *c.nonInteractive)
		},
	}

	addConfigFlag(cmd, &cfgFiles)

	return cmd
}

func (c Commands) diffCmd() *cobra.Command {
	cfgFiles := []string{}
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diff of the stacks to be deployed",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.CfgLoader.LoadConfig(cfgFiles, c.cfg); err != nil {
				return err
			}
			c.SA.Diff(*c.cfg)
			return nil
		},
	}

	addConfigFlag(cmd, &cfgFiles)

	return cmd
}

func (c Commands) deleteCmd() *cobra.Command {
	cfgFiles := []string{}
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes deployed stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.CfgLoader.LoadConfig(cfgFiles, c.cfg); err != nil {
				return err
			}

			return c.SA.Delete(*c.cfg, *c.nonInteractive)
		},
	}

	addConfigFlag(cmd, &cfgFiles)

	return cmd
}

func (c Commands) dumpConfigCmd() *cobra.Command {
	var format string

	cfgFiles := []string{}

	dumpCmd := &cobra.Command{
		Use:   "dump-config",
		Short: "Dump loaded config into stdout",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := c.CfgLoader.LoadConfig(cfgFiles, c.cfg); err != nil {
				return err
			}

			c.dumpCfg(format)
			return nil
		},
	}
	dumpCmd.Flags().StringVarP(&format, "format", "f", "yaml", "One of: yaml, toml, json")
	addConfigFlag(dumpCmd, &cfgFiles)

	return dumpCmd
}

func (c Commands) dumpCfg(format string) {
	out := c.Cli.Writer

	switch format {
	case "yaml", "yml":
		assembly.MustSucceed(yaml.NewEncoder(out).Encode(c.cfg))
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		assembly.MustSucceed(enc.Encode(c.cfg))
	case "toml":
		assembly.MustSucceed(toml.NewEncoder(out).Encode(c.cfg))
	default:
		assembly.Terminate("unknown format: " + format)
	}
}

func addConfigFlag(cmd *cobra.Command, val *[]string) {
	cmd.Flags().StringSliceVarP(val, "configs", "c", []string{},
		"Alternative config file(s). Default: stack-assembly.yaml")
}