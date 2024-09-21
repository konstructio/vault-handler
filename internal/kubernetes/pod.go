package kubernetes

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (k *Kubernetes) GetPodWhenReady(ctx context.Context, timeoutSeconds int, matchLabel string, namespace string) (*v1.Pod, error) {
	podListOptions := metav1.ListOptions{
		LabelSelector: matchLabel,
	}

	timeout := time.Duration(timeoutSeconds) * time.Second
	var foundPod v1.Pod

	// fetch the pod and ensure at least one is running
	if err := wait.PollUntilContextTimeout(ctx, 1*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		pods, err := k.clientset.CoreV1().Pods(namespace).List(ctx, podListOptions)
		if err != nil {
			return false, fmt.Errorf("error listing pods: %w", err)
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase == v1.PodRunning {
				foundPod = pod
				return true, nil
			}
		}

		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("error waiting for pod to be ready: %w", err)
	}

	return &foundPod, nil
}
