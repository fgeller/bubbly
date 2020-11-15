/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/verifa/bubbly/bubbly"
	cmdutil "github.com/verifa/bubbly/cmd/util"
	"github.com/verifa/bubbly/config"
	normalise "github.com/verifa/bubbly/util/normalise"
)

var (
	_         cmdutil.Options = (*ApplyOptions)(nil)
	applyLong                 = normalise.LongDesc(`
		Apply a Bubbly configuration (collection of 1 or more Bubbly Resources) to a Bubbly server

		    $ bubbly apply (-f (FILENAME | DIRECTORY)) [flags]

		will first check for an exact match on FILENAME. If no such filename
		exists, it will instead search for a directory.`)

	applyExample = normalise.Examples(`
		# Apply the configuration in the file ./main.bubbly
		bubbly apply -f ./main.bubbly

		# Apply the configuration in the directory ./config
		bubbly apply -f ./config`)
)

// ApplyOptions -
type ApplyOptions struct {
	o        cmdutil.Options //embedding
	Config   *config.Config
	Filename string

	// sc ServerConfig

	Command string
	Args    []string

	// Result from o.Run() - success / failure for the apply
	Result bool
}

// NewCmdApply creates a new cobra.Command representing "bubbly apply"
func NewCmdApply() (*cobra.Command, *ApplyOptions) {
	o := &ApplyOptions{
		Command: "apply",
		Config:  config.NewDefaultConfig(),
		Result:  false,
	}

	// cmd represents the apply command
	cmd := &cobra.Command{
		Use:     "apply (-f (FILENAME | DIRECTORY)) [flags]",
		Short:   "Apply a Bubbly configuration to a Bubbly server",
		Long:    applyLong + "\n\n" + cmdutil.SuggestBubblyResources(),
		Example: applyExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Debug().Strs("arguments", args).
				Msg("apply arguments")
			config, err := config.SetupConfigs()

			if err != nil {
				return fmt.Errorf("failed to set up configs: %w", err)
			}

			o.Config = config

			log.Debug().Interface("configuration_merged", o.Config).Msg("merged bubbly configuration")
			o.Args = args

			validationError := o.Validate(cmd)

			if validationError != nil {
				return validationError
			}

			resolveError := o.Resolve(cmd)

			if resolveError != nil {
				return resolveError
			}
			runError := o.Run()

			if runError != nil {
				return runError
			}

			o.Print(cmd)
			return nil
		},
		PreRun: func(cmd *cobra.Command, _ []string) {
			viper.BindPFlags(rootCmd.PersistentFlags())
			viper.BindPFlags(cmd.PersistentFlags())
			log.Debug().Interface("configuration", viper.AllSettings()).Msg("bubbly configuration")
		},
	}

	f := cmd.Flags()

	f.StringVarP(&o.Filename, "filename", "f", o.Filename, "filename or directory that contains the configuration to apply")
	cmd.MarkFlagRequired("filename")
	viper.BindPFlags(f)

	return cmd, o
}

// Validate checks the ApplyOptions to see if there is sufficient information run the command.
func (o *ApplyOptions) Validate(cmd *cobra.Command) error {
	if len(o.Args) != 0 {
		return cmdutil.UsageErrorf(cmd, "Unexpected args: %v", o.Args)
	}
	if o.Filename == "" {
		return fmt.Errorf("you must specify the filename or directory with -f %s", cmdutil.SuggestBubblyResources())
	}

	// TODO: validation of a given o.Filename. It might be sufficient to delegate this to parser (as is currently implemented).
	return nil
}

// Resolve resolves various ApplyOptions attributes from the provided arguments to cmd
func (o *ApplyOptions) Resolve(cmd *cobra.Command) error {
	return nil
}

// Run runs the apply command over the validated ApplyOptions configuration
func (o *ApplyOptions) Run() error {
	if err := bubbly.Apply(o.Filename, *o.Config.ServerConfig); err != nil {
		o.Result = false
		return fmt.Errorf("failed to apply configuration: %w", err)
	}
	o.Result = true
	return nil
}

// Print formats and prints the ApplyOptions.Result from o.Run()
func (o *ApplyOptions) Print(cmd *cobra.Command) {
	fmt.Fprintf(cmd.OutOrStdout(), "Apply result: %t\n", o.Result)
}

func init() {
	applyCmd, _ := NewCmdApply()
	rootCmd.AddCommand(applyCmd)
}