package vault

import (
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

// UnsealCoreTransit initializes and unseals a core instance used to provide transit unseal
func (conf *VaultConfiguration) UnsealCoreTransit(clientset *kubernetes.Clientset, restConfig *rest.Config) error {
	//* vault port-forward
	log.Infof("starting port-forward for vault-0")
	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()

	// Vault api client
	vaultClient, err := vaultapi.NewClient(&conf.Config)
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
		log.Info("initializing vault raft leader")

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
		for i, shard := range initResponse.Keys {
			if i < 3 {
				log.Infof("passing unseal shard %v to %s", i+1, "vault-0")
				_, err := vaultClient.Sys().Unseal(shard)
				if err != nil {
					return err
				}
			} else {
				break
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
			for i, shard := range existingInitResponse.Keys {
				if i < 3 {
					retries := 10
					for r := 0; r < retries; r++ {
						if r > 0 {
							log.Warnf("encountered an error during unseal, retrying (%d/%d)", r+1, retries)
						}
						time.Sleep(5 * time.Second)

						log.Infof("passing unseal shard %v to %s", i+1, "vault-0")
						_, err := vaultClient.Sys().Unseal(shard)
						if err != nil {
							continue
						} else {
							break
						}
					}
					time.Sleep(time.Second * 2)
				} else {
					break
				}

			}
		case false:
			log.Infof("%s is already unsealed", "vault-0")
		}
	}

	log.Infof("closing port-forward for vault-0")
	time.Sleep(time.Second * 3)

	return nil
}
