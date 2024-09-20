/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/vault-handler/internal/kubernetes"
	vault "github.com/kubefirst/vault-handler/internal/vault"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func getUnsealCommand() *cobra.Command {
	var opts vault.UnsealOptions

	unsealCmd := &cobra.Command{
		Use:   "unseal",
		Short: "Unseal a vault instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			kube, err := kubernetes.New(true)
			if err != nil {
				return fmt.Errorf("error creating kubernetes client: %s", err)
			}

			vault := vault.New(kube)

			if err := vault.UnsealRaftLeader(); err != nil {
				return fmt.Errorf("error unsealing vault raft leader: %s", err)
			}

			if !opts.UnsealLeaderOnly {
				if err := vault.UnsealRaftFollowers(); err != nil {
					return fmt.Errorf("error unsealing vault raft followers: %s", err)
				}
			}

			log.Info("vault initialized and unsealed successfully!")
			return nil
		},
	}

	unsealCmd.Flags().BoolVar(&opts.UnsealLeaderOnly, "leader-only", false, "unseal only the raft leader - false (default) - true to only init and unseal vault-0")
	unsealCmd.Flags().BoolVar(&opts.KubeInClusterConfig, "use-kubeconfig-in-cluster", true, "kube config type - in-cluster (default), set to false to use local")

	return unsealCmd
}
