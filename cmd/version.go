/*
Copyright © 2026 Mad Scientist Research LLC.
This file is part of Forms Filler.
*/

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Tell you the version details.",
	Long:  `Tell you the version details of Forms Filler.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(functionHelpLong + "\n")
		detailedVersion := fmt.Sprintf(
			`Version: %s
Build Arch: %v
Maintainer: %v
License: %v`,
			shortVersion, buildArch, projectMaintainer, projectLicense)
		fmt.Println(detailedVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
