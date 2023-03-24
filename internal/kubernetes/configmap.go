package kubernetes

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReadConfigMapV2
func ReadConfigMapV2(inCluster bool, namespace string, configMapName string) (map[string]string, error) {
	_, clientset, _ := CreateKubeConfig(inCluster)

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), configMapName, metav1.GetOptions{})
	if err != nil {
		return map[string]string{}, errors.New(fmt.Sprintf("error getting ConfigMap: %s\n", err))
	}

	parsedSecretData := make(map[string]string)
	for key, value := range configMap.Data {
		parsedSecretData[key] = string(value)
	}

	return parsedSecretData, nil
}

// UpdateConfigMapV2
func UpdateConfigMapV2(inCluster bool, namespace, configMapName string, key string, value string) error {
	_, clientset, _ := CreateKubeConfig(inCluster)

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), configMapName, metav1.GetOptions{})
	if err != nil {
		return errors.New(fmt.Sprintf("error getting ConfigMap: %s\n", err))
	}

	configMap.Data = map[string]string{key: value}
	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(
		context.Background(),
		configMap,
		metav1.UpdateOptions{},
	)

	log.Infof("updated ConfigMap %s in Namespace %s\n", configMap.Name, configMap.Namespace)

	return nil
}
