package validation

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ValidateCluster ensures the Kubernetes control plane is reachable and responsive.
func ValidateCluster(cfg *rest.Config) error {
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	if _, err := clientset.Discovery().ServerVersion(); err != nil {
		return fmt.Errorf("discover server version: %w", err)
	}

	return nil
}
