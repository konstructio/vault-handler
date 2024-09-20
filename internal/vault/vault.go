package vault

import (
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
)

// parseExistingVaultInitSecret returns the value of a vault initialization secret if it exists
func (conf *Configuration) parseExistingVaultInitSecret() (*vaultapi.InitResponse, error) {
	// If vault has already been initialized, the response is formatted to contain the value
	// of the initialization secret
	secret, err := conf.Kubernetes.ReadSecret(VaultSecretName, VaultNamespace)
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
