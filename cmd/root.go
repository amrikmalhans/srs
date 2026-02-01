// Package cmd implements the CLI commands for the SRS tool.
// It provides commands for managing flashcards, reviewing cards, syncing the database,
// and other operations. All commands use Cobra for command-line interface handling.
package cmd

import (
	"fmt"
	"os"

	"github.com/amrikmalhans/srs/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "srs",
	Short: "A CLI spaced repetition system (SRS) tool",
	Long: `SRS is a command-line tool for managing flashcards using
spaced repetition. Cards are stored as markdown files with scheduling
state tracked in SQLite.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is resolved automatically)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Resolve config path
		configPath, err := config.ResolveConfigPath("")
		if err != nil {
			// Non-fatal error during initialization, just log and continue
			fmt.Fprintf(os.Stderr, "Warning: failed to resolve config path: %v\n", err)
			return
		}

		// Set config directory
		viper.AddConfigPath(configPath)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// Config file found and parsed successfully
		// This is optional, so we don't error if it doesn't exist
	}
}
