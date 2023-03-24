/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	kubernetesinternal "github.com/kubefirst/vault-handler/internal/kubernetes"
	vault "github.com/kubefirst/vault-handler/internal/vault"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	vaultUnsealOpts *vault.VaultUnsealExecutionOptions = &vault.VaultUnsealExecutionOptions{}
)

// unsealCmd represents the unseal command
var unsealCmd = &cobra.Command{
	Use:   "unseal",
	Short: "Unseal a vault instance",
	Long:  `Unseal a vault instance`,
	Run: func(cmd *cobra.Command, args []string) {
		vaultClient := &vault.Conf
		restconfig, clientset, _ := kubernetesinternal.CreateKubeConfig(true)
		err := vaultClient.UnsealRaftLeader(clientset, restconfig)
		if err != nil {
			log.Fatalf("error unsealing vault raft leader: %s", err)
		}
		if !vaultUnsealOpts.UnsealLeaderOnly {
			err = vaultClient.UnsealRaftFollowers(clientset, restconfig)
			if err != nil {
				log.Fatalf("error unsealing vault raft followers: %s", err)
			}
		}
		log.Info("vault initialized and unsealed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(unsealCmd)

	unsealCmd.Flags().BoolVar(&vaultUnsealOpts.UnsealLeaderOnly, "leader-only", false, "unseal only the raft leader - false (default) - true to only init and unseal vault-0")
	unsealCmd.Flags().BoolVar(&vaultUnsealOpts.KubeInClusterConfig, "use-kubeconfig-in-cluster", true, "kube config type - in-cluster (default), set to false to use local")
}
