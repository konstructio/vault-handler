package cmd

import (
	"github.com/spf13/cobra"
)

func Run() error {
	rootCmd := &cobra.Command{
		Use:   "vault-handler",
		Short: "An application to assist with managing Vault",
		Long:  `An application to assist with managing Vault, especially useful in cases where there is no option for things like KMS auto unseal, etc.`,
	}

	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.AddCommand(getInitCommand())
	rootCmd.AddCommand(getUnsealCommand())

	return rootCmd.Execute()
}
