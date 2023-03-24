package kubernetes

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ReturnPodObject returns a matching corev1.Pod object based on the filters
func ReturnPodObject(clientset *kubernetes.Clientset, matchLabel string, matchLabelValue string, namespace string, timeoutSeconds float64) (*corev1.Pod, error) {
	// Filter
	podListOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", matchLabel, matchLabelValue),
	}

	// Create watch operation
	objWatch, err := clientset.
		CoreV1().
		Pods(namespace).
		Watch(context.Background(), podListOptions)
	if err != nil {
		log.Fatalf("error when attempting to search for Pod: %s", err)
	}
	log.Infof("waiting for %s Pod to be created", matchLabelValue)

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 1)
			if !ok {
				// Error if the channel closes
				log.Fatalf("error waiting for %s Pod to be created: %s", matchLabelValue, err)
			}
			if event.
				Object.(*v1.Pod).Status.Phase == "Pending" {
				spec, err := clientset.CoreV1().Pods(namespace).List(context.Background(), podListOptions)
				if err != nil {
					log.Fatalf("error when searching for Pod: %s", err)
					return nil, err
				}
				return &spec.Items[0], nil
			}
			if event.
				Object.(*v1.Pod).Status.Phase == "Running" {
				spec, err := clientset.CoreV1().Pods(namespace).List(context.Background(), podListOptions)
				if err != nil {
					log.Fatalf("error when searching for Pod: %s", err)
					return nil, err
				}
				return &spec.Items[0], nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Errorf("the Pod was not created within the timeout period")
			return nil, fmt.Errorf("the Pod was not created within the timeout period")
		}
	}
}
