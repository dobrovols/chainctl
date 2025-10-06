package node

import (
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/dobrovols/chainctl/pkg/tokens"
)

func kubeTokenStore() *tokens.KubeStore {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		overrides := &clientcmd.ConfigOverrides{}
		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
		cfg, err = clientConfig.ClientConfig()
		if err != nil {
			return nil
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil
	}

	return tokens.NewKubeStore(clientset, os.Getenv("CHAINCTL_TOKEN_NAMESPACE"))
}

func defaultStore() tokenStore {
	if store := kubeTokenStore(); store != nil {
		return store
	}
	return tokens.NewMemoryStore()
}

func joinStore() tokenConsumer {
	if store := kubeTokenStore(); store != nil {
		return store
	}
	return tokens.NewMemoryStore()
}
