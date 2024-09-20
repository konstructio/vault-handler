package vault

import (
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/kubefirst/vault-handler/internal/kubernetes"
)

type Configuration struct {
	Config     *vaultapi.Config
	Kubernetes *kubernetes.Kubernetes
}

func New(kubernetes *kubernetes.Kubernetes) *Configuration {
	return &Configuration{
		Config:     vaultapi.DefaultConfig(),
		Kubernetes: kubernetes,
	}
}
