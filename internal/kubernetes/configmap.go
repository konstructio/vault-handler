package kubernetes

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *Kubernetes) ReadConfigMap(name, namespace string) (map[string]string, error) {
	configMap, err := k.clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting ConfigMap: %w", err)
	}

	parsedSecretData := make(map[string]string)
	for key, value := range configMap.Data {
		parsedSecretData[key] = value
	}

	return parsedSecretData, nil
}

func (k *Kubernetes) UpdateConfigMap(name, namespace, key, value string) error {
	configMap, err := k.ReadConfigMap(name, namespace)
	if err != nil {
		return err
	}

	configMap[key] = value

	_, err = k.clientset.CoreV1().ConfigMaps(namespace).Update(
		context.Background(),
		&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: configMap,
		},
		metav1.UpdateOptions{},
	)
	if err != nil {
		return fmt.Errorf("error updating ConfigMap: %w", err)
	}

	return nil
}
