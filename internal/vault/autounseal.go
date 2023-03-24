package vault

import (
	"context"
	"errors"
	"fmt"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	kubernetesinternal "github.com/kubefirst/vault-handler/internal/kubernetes"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func (conf *VaultConfiguration) AutoUnseal() (*vaultapi.InitResponse, error) {
	vaultClient, err := vaultapi.NewClient(&conf.Config)
	if err != nil {
		return &vaultapi.InitResponse{}, err
	}
	vaultClient.CloneConfig().ConfigureTLS(&vaultapi.TLSConfig{
		Insecure: true,
	})
	log.Infof("created vault client, initializing vault with auto unseal")

	initResponse, err := vaultClient.Sys().Init(&vaultapi.InitRequest{
		RecoveryShares:    RecoveryShares,
		RecoveryThreshold: RecoveryThreshold,
		SecretShares:      SecretShares,
		SecretThreshold:   SecretThreshold,
	})
	if err != nil {
		return &vaultapi.InitResponse{}, err
	}
	log.Infof("vault initialization complete")

	return initResponse, nil
}

// UnsealRaftLeader initializes and unseals a vault leader when using raft for ha and storage
func (conf *VaultConfiguration) UnsealRaftLeader(clientset *kubernetes.Clientset, restConfig *rest.Config) error {
	node := "vault-0"

	pod, err := kubernetesinternal.ReturnPodObject(clientset, "statefulset.kubernetes.io/pod-name", node, "vault", 60)
	if err != nil {
		return err
	}

	// Vault api client
	vaultClient, err := vaultapi.NewClient(&vaultapi.Config{
		Address: fmt.Sprintf("http://%s:8200", pod.Status.PodIP),
	})
	if err != nil {
		return err
	}

	vaultClient.CloneConfig().ConfigureTLS(&vaultapi.TLSConfig{
		Insecure: true,
	})

	// Determine vault health
	health, err := vaultClient.Sys().Health()
	if err != nil {
		return err
	}

	switch health.Initialized {
	case false:
		log.Infof("initializing vault raft leader")

		initResponse, err := vaultClient.Sys().Init(&vaultapi.InitRequest{
			RecoveryShares:    RecoveryShares,
			RecoveryThreshold: RecoveryThreshold,
			SecretShares:      SecretShares,
			SecretThreshold:   SecretThreshold,
		})
		if err != nil {
			return err
		}

		// Write secret containing init data
		dataToWrite := make(map[string][]byte)
		dataToWrite["root-token"] = []byte(initResponse.RootToken)
		for i, value := range initResponse.Keys {
			dataToWrite[fmt.Sprintf("root-unseal-key-%v", i+1)] = []byte(value)
		}
		secret := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      VaultSecretName,
				Namespace: VaultNamespace,
			},
			Data: dataToWrite,
		}

		log.Infof("creating secret %s containing vault initialization data", VaultSecretName)
		err = kubernetesinternal.CreateSecretV2(clientset, &secret)
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second * 3)

		// Unseal raft leader
		sealStatusTracking := 0
		for i, shard := range initResponse.Keys {
			if i < SecretThreshold {
				log.Infof("passing unseal shard %v to %s", i+1, node)
				deadline := time.Now().Add(60 * time.Second)
				ctx, cancel := context.WithDeadline(context.Background(), deadline)
				defer cancel()
				// Try 5 times to pass unseal shard
				for i := 0; i < 5; i++ {
					_, err := vaultClient.Sys().UnsealWithContext(ctx, shard)
					if err != nil {
						if errors.Is(err, context.DeadlineExceeded) {
							continue
						}
					}
					if i == 5 {
						return fmt.Errorf("error passing unseal shard %v to %s: %s", i+1, node, err)
					}
				}
				// Wait for key acceptance
				for i := 0; i < 10; i++ {
					sealStatus, err := vaultClient.Sys().SealStatus()
					if err != nil {
						return fmt.Errorf("error retrieving health of %s: %s", node, err)
					}
					if sealStatus.Progress > sealStatusTracking || !sealStatus.Sealed {
						log.Infof("shard accepted")
						sealStatusTracking += 1
						break
					}
					log.Infof("waiting for node %s to accept unseal shard", node)
					time.Sleep(time.Second * 6)
				}
			}
		}
	case true:
		log.Infof("%s is already initialized", "vault-0")

		// Determine vault health
		health, err = vaultClient.Sys().Health()
		if err != nil {
			return err
		}

		switch health.Sealed {
		case true:
			existingInitResponse, err := parseExistingVaultInitSecret(clientset)
			if err != nil {
				return err
			}

			// Unseal raft leader
			sealStatusTracking := 0
			node := "vault-0"
			for i, shard := range existingInitResponse.Keys {
				if i < SecretThreshold {
					log.Infof("passing unseal shard %v to %s", i+1, node)
					deadline := time.Now().Add(60 * time.Second)
					ctx, cancel := context.WithDeadline(context.Background(), deadline)
					defer cancel()
					// Try 5 times to pass unseal shard
					for i := 0; i < 5; i++ {
						_, err := vaultClient.Sys().UnsealWithContext(ctx, shard)
						if err != nil {
							if errors.Is(err, context.DeadlineExceeded) {
								continue
							}
						}
						if i == 5 {
							return fmt.Errorf("error passing unseal shard %v to %s: %s", i+1, node, err)
						}
					}
					// Wait for key acceptance
					for i := 0; i < 10; i++ {
						sealStatus, err := vaultClient.Sys().SealStatus()
						if err != nil {
							return fmt.Errorf("error retrieving health of %s: %s", node, err)
						}
						if sealStatus.Progress > sealStatusTracking || !sealStatus.Sealed {
							log.Infof("shard accepted")
							sealStatusTracking += 1
							break
						}
						log.Infof("waiting for node %s to accept unseal shard", node)
						time.Sleep(time.Second * 6)
					}
				}
			}
		case false:
			log.Infof("%s is already unsealed", "vault-0")
		}
	}

	return nil
}

// UnsealRaftFollowers initializes, unseals, and joins raft followers when using raft for ha and storage
func (conf *VaultConfiguration) UnsealRaftFollowers(clientset *kubernetes.Clientset, restConfig *rest.Config) error {
	// With the current iteration of the Vault helm chart, we create 3 nodes
	// vault-0 is unsealed as leader, vault-1 and vault-2 are unsealed here
	raftNodes := []string{"vault-1", "vault-2"}
	existingInitResponse, err := parseExistingVaultInitSecret(clientset)
	if err != nil {
		return err
	}

	for _, node := range raftNodes {
		pod, err := kubernetesinternal.ReturnPodObject(clientset, "statefulset.kubernetes.io/pod-name", node, "vault", 60)
		if err != nil {
			return err
		}

		// Vault api client
		vaultClient, err := vaultapi.NewClient(&vaultapi.Config{
			Address: fmt.Sprintf("http://%s:8200", pod.Status.PodIP),
		})
		if err != nil {
			return err
		}
		vaultClient.CloneConfig().ConfigureTLS(&vaultapi.TLSConfig{
			Insecure: true,
		})
		log.Infof("created vault client for %s", node)

		// Determine vault health
		health, err := vaultClient.Sys().Health()
		if err != nil {
			return err
		}

		switch health.Initialized {
		case false:
			// Join to raft cluster
			log.Infof("joining raft follower %s to vault cluster", node)
			_, err = vaultClient.Sys().RaftJoin(&vaultapi.RaftJoinRequest{
				//AutoJoin:         "",
				//AutoJoinScheme:   "",
				//AutoJoinPort:     0,
				LeaderAPIAddr: fmt.Sprintf("%s:8200", vaultRaftPrimaryAddress),
				// LeaderCACert:     "",
				// LeaderClientCert: "",
				// LeaderClientKey:  "",
				Retry: true,
			})
			if err != nil {
				return err
			}
			time.Sleep(time.Second * 5)
		case true:
			log.Infof("raft follower %s is already initialized", node)
		}

		// Determine vault health
		health, err = vaultClient.Sys().Health()
		if err != nil {
			return err
		}

		switch health.Sealed {
		case true:
			// Unseal raft followers
			sealStatusTracking := 0
			for i, shard := range existingInitResponse.Keys {
				if i < SecretThreshold {
					log.Infof("passing unseal shard %v to %s", i+1, node)
					deadline := time.Now().Add(60 * time.Second)
					ctx, cancel := context.WithDeadline(context.Background(), deadline)
					defer cancel()
					// Try 5 times to pass unseal shard
					for i := 0; i < 5; i++ {
						_, err := vaultClient.Sys().UnsealWithContext(ctx, shard)
						if err != nil {
							if errors.Is(err, context.DeadlineExceeded) {
								continue
							}
						}
						if i == 5 {
							return fmt.Errorf("error passing unseal shard %v to %s: %s", i+1, node, err)
						}
					}
					// Wait for key acceptance
					for i := 0; i < 10; i++ {
						sealStatus, err := vaultClient.Sys().SealStatus()
						if err != nil {
							return fmt.Errorf("error retrieving health of %s: %s", node, err)
						}
						if sealStatus.Progress > sealStatusTracking || !sealStatus.Sealed {
							log.Infof("shard accepted")
							sealStatusTracking += 1
							break
						}
						log.Infof("waiting for node %s to accept unseal shard", node)
						time.Sleep(time.Second * 6)
					}
				}
			}
		case false:
			log.Infof("raft follower %s is already unsealed", node)
		}
	}

	return nil
}
