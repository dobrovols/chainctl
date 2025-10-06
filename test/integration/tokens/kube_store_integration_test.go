package tokensintegration_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/dobrovols/chainctl/pkg/tokens"
)

func TestKubeStoreRoundTrip(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("skipping envtest token integration; KUBEBUILDER_ASSETS not set")
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

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("build clientset: %v", err)
	}

	ctx := context.Background()
	if _, err := clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tokens.DefaultNamespace}}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("ensure namespace: %v", err)
	}

	now := time.Unix(1_700_000_000, 0).UTC()
	store := tokens.NewKubeStore(clientset, "", tokens.WithClock(func() time.Time { return now }))

	created, err := store.Create(tokens.CreateOptions{Scope: tokens.ScopeWorker, TTL: time.Minute})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	now = now.Add(10 * time.Second)

	if err := store.Consume(created.Token, tokens.ScopeWorker); err != nil {
		t.Fatalf("consume token: %v", err)
	}

	secret, err := clientset.CoreV1().Secrets(tokens.DefaultNamespace).Get(ctx, createdSecretName(created), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("fetch secret: %v", err)
	}

	var record tokens.Token
	if err := json.Unmarshal(secret.Data["record"], &record); err != nil {
		t.Fatalf("decode record: %v", err)
	}

	if !record.Consumed {
		t.Fatalf("expected record consumed flag to be true")
	}
}

func createdSecretName(created *tokens.CreatedToken) string {
	// token id is before the dot in the composite token.
	return "chainctl-node-token-" + created.ID
}
