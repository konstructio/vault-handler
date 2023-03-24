package vault

import (
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
	kubernetesinternal "github.com/kubefirst/vault-handler/internal/kubernetes"
	"k8s.io/client-go/kubernetes"
)

// parseExistingVaultInitSecret returns the value of a vault initialization secret if it exists
func parseExistingVaultInitSecret(clientset *kubernetes.Clientset) (*vaultapi.InitResponse, error) {
	// If vault has already been initialized, the response is formatted to contain the value
	// of the initialization secret
	secret, err := kubernetesinternal.ReadSecretV2(clientset, VaultNamespace, VaultSecretName)
	if err != nil {
		return &vaultapi.InitResponse{}, err
	}

	// Add root-unseal-key entries to slice
	var rkSlice []string
	for key, value := range secret {
		if strings.Contains(key, "root-unseal-key-") {
			rkSlice = append(rkSlice, value)
		}
	}

	existingInitResponse := &vaultapi.InitResponse{
		Keys:      rkSlice,
		RootToken: secret["root-token"],
	}
	return existingInitResponse, nil
}
