package kubernetes

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *Kubernetes) CreateSecret(secret *v1.Secret) error {
	_, err := k.clientset.CoreV1().Secrets(secret.Namespace).Create(
		context.Background(),
		secret,
		metav1.CreateOptions{},
	)
	if err != nil {
		return fmt.Errorf("error creating secret: %w", err)
	}

	log.Infof("created Secret %s in Namespace %s", secret.Name, secret.Namespace)
	return nil
}

func (k *Kubernetes) ReadSecret(name, namespace string) (map[string]string, error) {
	secret, err := k.clientset.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting secret: %w", err)
	}

	parsedSecretData := make(map[string]string)
	for key, value := range secret.Data {
		parsedSecretData[key] = string(value)
	}

	return parsedSecretData, nil
}
