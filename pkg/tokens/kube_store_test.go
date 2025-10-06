package tokens

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestKubeStoreCreatePersistsSecret(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: DefaultNamespace}})
	now := time.Unix(1_700_000_000, 0).UTC()

	store := NewKubeStore(client, "", WithClock(func() time.Time { return now }))

	created, err := store.Create(CreateOptions{
		Scope:       ScopeWorker,
		TTL:         time.Hour,
		CreatedBy:   "tester",
		Description: "provision worker",
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	if created.Scope != ScopeWorker {
		t.Fatalf("expected scope worker, got %s", created.Scope)
	}
	if created.Token == "" {
		t.Fatalf("expected composite token in response")
	}

	secrets, err := client.CoreV1().Secrets(DefaultNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("list secrets: %v", err)
	}
	if len(secrets.Items) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets.Items))
	}

	secret := secrets.Items[0]
	if secret.Labels[labelTokenScope] != string(ScopeWorker) {
		t.Fatalf("expected scope label %q, got %q", string(ScopeWorker), secret.Labels[labelTokenScope])
	}

	var record Token
	if err := json.Unmarshal(secret.Data[dataKeyRecord], &record); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	if record.Scope != ScopeWorker {
		t.Fatalf("expected record scope worker, got %s", record.Scope)
	}
	if !record.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("expected expiry %s, got %s", now.Add(time.Hour), record.ExpiresAt)
	}
	if record.Consumed {
		t.Fatalf("expected record not consumed")
	}
}

func TestKubeStoreConsumeSuccess(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: DefaultNamespace}})
	clock := time.Unix(1_700_000_000, 0).UTC()
	store := NewKubeStore(client, "", WithClock(func() time.Time { return clock }))

	created, err := store.Create(CreateOptions{Scope: ScopeWorker, TTL: 2 * time.Hour})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	clock = clock.Add(30 * time.Minute)

	if err := store.Consume(created.Token, ScopeWorker); err != nil {
		t.Fatalf("consume token: %v", err)
	}

	secret, err := client.CoreV1().Secrets(DefaultNamespace).Get(context.Background(), tokenSecretName(created.ID), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}

	if secret.Annotations[annotationConsumedAt] == "" {
		t.Fatalf("expected consumed annotation to be set")
	}

	var record Token
	if err := json.Unmarshal(secret.Data[dataKeyRecord], &record); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	if !record.Consumed {
		t.Fatalf("expected record marked consumed")
	}

	if err := store.Consume(created.Token, ScopeWorker); !errors.Is(err, errTokenConsumed) {
		t.Fatalf("expected errTokenConsumed, got %v", err)
	}
}

func TestKubeStoreConsumeScopeAndExpiry(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: DefaultNamespace}})
	clock := time.Unix(1_700_000_000, 0).UTC()
	store := NewKubeStore(client, "", WithClock(func() time.Time { return clock }))

	created, err := store.Create(CreateOptions{Scope: ScopeWorker, TTL: time.Minute})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	if err := store.Consume(created.Token, ScopeControlPlane); !errors.Is(err, errScopeMismatch) {
		t.Fatalf("expected scope mismatch error, got %v", err)
	}

	clock = clock.Add(2 * time.Minute)

	if err := store.Consume(created.Token, ScopeWorker); !errors.Is(err, errTokenExpired) {
		t.Fatalf("expected expiry error, got %v", err)
	}
}
