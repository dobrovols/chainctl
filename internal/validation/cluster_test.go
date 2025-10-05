package validation

import (
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
)

func TestValidateClusterSuccess(t *testing.T) {
	original := newKubeClient
	t.Cleanup(func() { newKubeClient = original })

	client := fake.NewSimpleClientset()
	discovery := client.Discovery().(*discoveryfake.FakeDiscovery)
	discovery.FakedServerVersion = &version.Info{Major: "1", Minor: "28"}

	newKubeClient = func(*rest.Config) (kubernetes.Interface, error) {
		return client, nil
	}

	if err := ValidateCluster(&rest.Config{}); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestValidateClusterNewClientError(t *testing.T) {
	original := newKubeClient
	t.Cleanup(func() { newKubeClient = original })

	newKubeClient = func(*rest.Config) (kubernetes.Interface, error) {
		return nil, errors.New("dial failed")
	}

	err := ValidateCluster(&rest.Config{})
	if err == nil || err.Error() != "create kubernetes client: dial failed" {
		t.Fatalf("expected create client error, got %v", err)
	}
}

func TestValidateClusterServerVersionError(t *testing.T) {
	original := newKubeClient
	t.Cleanup(func() { newKubeClient = original })

	client := fake.NewSimpleClientset()
	discovery := client.Discovery().(*discoveryfake.FakeDiscovery)
	discovery.Fake.PrependReactor("get", "version", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("boom")
	})

	newKubeClient = func(*rest.Config) (kubernetes.Interface, error) {
		return client, nil
	}

	err := ValidateCluster(&rest.Config{})
	if err == nil || err.Error() != "discover server version: boom" {
		t.Fatalf("expected server version error, got %v", err)
	}
}
