package vault

import (
	"context"
	"errors"
	"fmt"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (conf *Configuration) getVaultClientForNode(node string) (*vaultapi.Client, error) {
	pod, err := conf.Kubernetes.GetPodWhenReady(context.Background(), 60, "statefulset.kubernetes.io/pod-name="+node, "vault")
	if err != nil {
		return nil, fmt.Errorf("error getting pod %q: %w", node, err)
	}

	configCopy := conf.Config
	configCopy.Address = fmt.Sprintf("http://%s:8200", pod.Status.PodIP)
	client, err := vaultapi.NewClient(configCopy)
	if err != nil {
		return nil, fmt.Errorf("error creating vault client: %w", err)
	}

	insecureConfig := client.CloneConfig()
	insecureConfig.ConfigureTLS(&vaultapi.TLSConfig{
		Insecure: true,
	})

	return client, nil
}

func unsealVault(client *vaultapi.Client, keys []string) error {
	sealStatusTracking := 0
	for i, shard := range keys {
		if i >= SecretThreshold {
			break
		}

		deadline := time.Now().Add(60 * time.Second)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)

		var unsealed bool
		for attempt := 0; attempt < 5; attempt++ {
			_, err := client.Sys().UnsealWithContext(ctx, shard)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				} else {
					cancel()
					return fmt.Errorf("error unsealing shard %v: %w", i+1, err)
				}
			} else {
				unsealed = true
				break
			}
		}

		cancel()

		if !unsealed {
			return fmt.Errorf("error unsealing shard %v: context deadline exceeded", i+1)
		}

		for attempt := 0; attempt < 10; attempt++ {
			sealStatus, err := client.Sys().SealStatus()
			if err != nil {
				return fmt.Errorf("error getting seal status: %w", err)
			}

			if sealStatus.Progress > sealStatusTracking || !sealStatus.Sealed {
				sealStatusTracking++
				break
			}

			time.Sleep(time.Second * 6)
		}
	}
	return nil
}

func (conf *Configuration) initializeVault(client *vaultapi.Client) (*vaultapi.InitResponse, error) {
	initResponse, err := client.Sys().Init(&vaultapi.InitRequest{
		SecretShares:    SecretShares,
		SecretThreshold: SecretThreshold,
	})
	if err != nil {
		return nil, fmt.Errorf("error initializing vault: %w", err)
	}

	dataToWrite := map[string][]byte{
		"root-token": []byte(initResponse.RootToken),
	}

	for i, value := range initResponse.Keys {
		dataToWrite[fmt.Sprintf("root-unseal-key-%v", i+1)] = []byte(value)
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      VaultSecretName,
			Namespace: VaultNamespace,
		},
		Data: dataToWrite,
	}

	if err := conf.Kubernetes.CreateSecret(secret); err != nil {
		return nil, fmt.Errorf("error creating secret %q: %w", VaultSecretName, err)
	}

	time.Sleep(time.Second * 3)

	return initResponse, nil
}

func (conf *Configuration) unsealNode(node string, keys []string) error {
	client, err := conf.getVaultClientForNode(node)
	if err != nil {
		return err
	}

	health, err := client.Sys().Health()
	if err != nil {
		return fmt.Errorf("error getting vault health: %w", err)
	}

	if !health.Initialized {
		return fmt.Errorf("vault %q is not initialized", node)
	}

	if health.Sealed {
		if err := unsealVault(client, keys); err != nil {
			return err
		}
	}

	return nil
}

func (conf *Configuration) UnsealRaftLeader() error {
	node := "vault-0"
	client, err := conf.getVaultClientForNode(node)
	if err != nil {
		return err
	}

	health, err := client.Sys().Health()
	if err != nil {
		return fmt.Errorf("error getting vault health: %w", err)
	}

	if !health.Initialized {
		initResponse, err := conf.initializeVault(client)
		if err != nil {
			return err
		}
		if err := unsealVault(client, initResponse.Keys); err != nil {
			return err
		}
	} else if health.Sealed {
		existingInitResponse, err := conf.parseExistingVaultInitSecret()
		if err != nil {
			return fmt.Errorf("error parsing existing vault init secret: %w", err)
		}
		if err := unsealVault(client, existingInitResponse.Keys); err != nil {
			return err
		}
	}

	return nil
}

func (conf *Configuration) UnsealRaftFollowers() error {
	raftNodes := []string{"vault-1", "vault-2"}
	existingInitResponse, err := conf.parseExistingVaultInitSecret()
	if err != nil {
		return fmt.Errorf("error parsing existing vault init secret: %w", err)
	}

	for _, node := range raftNodes {
		if err := conf.unsealNode(node, existingInitResponse.Keys); err != nil {
			return err
		}
	}

	return nil
}
