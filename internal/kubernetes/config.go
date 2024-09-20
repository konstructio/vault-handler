package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Kubernetes struct {
	clientset kubernetes.Interface
	config    *rest.Config
	inCluster bool
}

func New(inCluster bool) (*Kubernetes, error) {
	kube := &Kubernetes{}
	kube.inCluster = inCluster

	var kubeconfigLocation string

	if inCluster {
		kubeconfigLocation = ""
	} else {
		kubeconfigLocation = remoteKubernetes()

		if _, err := os.Stat(kubeconfigLocation); err != nil {
			return nil, fmt.Errorf("error reading kubeconfig: %s", err)
		}
	}

	// From the docs: If neither masterUrl or kubeconfigPath are passed in we
	// fallback to inClusterConfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigLocation)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %s", err)
	}

	kube.clientset = clientset
	kube.config = config

	return kube, nil
}

func remoteKubernetes() string {
	if loc := os.Getenv("KUBECONFIG"); loc != "" {
		return loc
	}
	return filepath.Join(homedir.HomeDir(), ".kube", "config")
}

func (k *Kubernetes) GetClientSet() kubernetes.Interface {
	return k.clientset
}

func (k *Kubernetes) GetConfig() *rest.Config {
	return k.config
}
