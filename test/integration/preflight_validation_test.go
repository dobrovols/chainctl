package integration

import (
	"os"
	"testing"

	"github.com/dobrovols/chainctl/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestValidateClusterWithEnvtest(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("skipping envtest preflight integration; KUBEBUILDER_ASSETS not set")
	}
	env := &envtest.Environment{}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	t.Cleanup(func() {
		if stopErr := env.Stop(); stopErr != nil {
			t.Fatalf("stop envtest: %v", stopErr)
		}
	})

	if err := validation.ValidateCluster(cfg); err != nil {
		t.Fatalf("ValidateCluster: %v", err)
	}
}
