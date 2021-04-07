package cmd

import (
	"errors"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var debug bool

// path to the config file
var config string

// paco deployment name
var name string

// output path
var output string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "paco-parser",
	Short: "Parse paco deployment file and generate the necessary k8s objects",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if debug {
			log.SetLevel(log.DebugLevel)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug mode")
	rootCmd.PersistentFlags().StringVarP(&config, "config", "c", "", "path to the file with paco deployment information")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "out", "path to the output path")
	rootCmd.PersistentFlags().StringVarP(&name, "name", "n", "", "paco deployment name")
}

// returns an error if config path is not provided
func configSet() error {
	if config == "" {
		return errors.New("path to the paco configuration definition file must be provided with --conf/-c flag")
	}
	return nil
}
