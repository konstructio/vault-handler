/*
Copyright (C) 2021-2024, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func getInitCommand() *cobra.Command {
	// initCmd represents the init command
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a vault instance",
		Long:  `Initialize a vault instance`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("init called")
		},
	}

	return initCmd
}
