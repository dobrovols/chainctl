package validation

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var newKubeClient = func(cfg *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(cfg)
}

// ValidateCluster ensures the Kubernetes control plane is reachable and responsive.
func ValidateCluster(cfg *rest.Config) error {
	clientset, err := newKubeClient(cfg)
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	if _, err := clientset.Discovery().ServerVersion(); err != nil {
		return fmt.Errorf("discover server version: %w", err)
	}

	return nil
}
