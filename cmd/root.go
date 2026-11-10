/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package cmd

import (
	"bufio"
	"fmt"
	"github.com/ScienceGuns/FormsFiller/apis/atf_forms"
	"github.com/ScienceGuns/FormsFiller/apis/internal_funcs"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command, which runs the form filling flow when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "formsfiller",
	Short: functionHelpShort,
	Long: fmt.Sprintf(`%s

Version: %s`, functionHelpLong, shortVersion),
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Load and validate the settings file before asking the user anything
		settings := pullSettings()
		// Ask which form to fill out
		reader := bufio.NewReader(os.Stdin)
		selectedForm, err := atf_forms.SelectForm(reader, settings)
		if err != nil {
			log.Fatalf("Unable to continue: %v", err)
		}
		// Ask for the details unique to the selected form and fill it out
		if err := selectedForm.Fill(reader, settings); err != nil {
			log.Fatalf("Failed to fill out %s: %v", selectedForm.Name, err)
		}
	},
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
	// Skip timestamps as we don't need them for CLI output
	log.SetFlags(0)
	rootCmd.PersistentFlags().StringVar(&SettingsFilePath, "settings-path", SettingsFilePath,
		"OPTIONAL: Path to the settings file")
}

// pullSettings Check and read the settings file, exiting on any problem
func pullSettings() *internal_funcs.FFConfig {
	// Make sure the file exists and only the owner can read it
	internal_funcs.CheckForSettingsFile(SettingsFilePath)
	// Read and validate the contents
	settings, err := internal_funcs.ReadSettingsFile(SettingsFilePath)
	if err != nil {
		log.Fatalf("Failed to load %s: %v", SettingsFilePath, err)
	}
	return settings
}
