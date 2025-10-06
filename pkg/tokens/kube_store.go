package tokens

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// DefaultNamespace is where token secrets are stored when no override is provided.
	DefaultNamespace = "chain-system"

	secretPrefix          = "chainctl-node-token-"
	dataKeyRecord         = "record"
	annotationConsumedAt  = "chainctl.io/consumedAt"
	annotationCreatedBy   = "chainctl.io/createdBy"
	annotationDescription = "chainctl.io/description"
	annotationExpiresAt   = "chainctl.io/expiresAt"
	labelManagedBy        = "app.kubernetes.io/managed-by"
	managedByValue        = "chainctl"
	labelTokenScope       = "chainctl.io/token-scope"
	requestTimeout        = 5 * time.Second
)

// KubeStore persists join tokens in Kubernetes secrets.
type KubeStore struct {
	client    kubernetes.Interface
	namespace string
	clock     func() time.Time
}

// KubeStoreOption configures a kube-backed store instance.
type KubeStoreOption func(*KubeStore)

// WithClock overrides the wall clock used for TTL and consumption timestamps (tests).
func WithClock(clock func() time.Time) KubeStoreOption {
	return func(s *KubeStore) {
		if clock != nil {
			s.clock = clock
		}
	}
}

// NewKubeStore constructs a Kubernetes-backed token store.
func NewKubeStore(client kubernetes.Interface, namespace string, opts ...KubeStoreOption) *KubeStore {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = DefaultNamespace
	}
	store := &KubeStore{
		client:    client,
		namespace: ns,
		clock:     time.Now,
	}
	for _, opt := range opts {
		opt(store)
	}
	return store
}

// Create stores a freshly generated token as a Kubernetes secret.
func (s *KubeStore) Create(opts CreateOptions) (*CreatedToken, error) {
	if s.client == nil {
		return nil, fmt.Errorf("kube store not initialised")
	}

	record, created, err := generateToken(opts, s.clock())
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshal token record: %w", err)
	}

	annotations := map[string]string{
		annotationExpiresAt: record.ExpiresAt.UTC().Format(time.RFC3339),
	}
	if record.CreatedBy != "" {
		annotations[annotationCreatedBy] = record.CreatedBy
	}
	if record.Description != "" {
		annotations[annotationDescription] = record.Description
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tokenSecretName(record.ID),
			Namespace: s.namespace,
			Labels: map[string]string{
				labelManagedBy:  managedByValue,
				labelTokenScope: string(record.Scope),
			},
			Annotations: annotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			dataKeyRecord: payload,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	if _, err := s.client.CoreV1().Secrets(s.namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("create token secret: %w", err)
	}

	return created, nil
}

// Consume validates and marks a token as used within the Kubernetes secret store.
func (s *KubeStore) Consume(composite string, expected Scope) error {
	if s.client == nil {
		return fmt.Errorf("kube store not initialised")
	}

	id, secret, err := splitToken(composite)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	stored, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, tokenSecretName(id), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return errTokenNotFound
	}
	if err != nil {
		return fmt.Errorf("get token secret: %w", err)
	}

	raw, ok := stored.Data[dataKeyRecord]
	if !ok {
		return fmt.Errorf("token secret malformed: record payload missing")
	}

	var record Token
	if err := json.Unmarshal(raw, &record); err != nil {
		return fmt.Errorf("decode token record: %w", err)
	}

	now := s.clock()

	if record.Consumed {
		return errTokenConsumed
	}
	if now.After(record.ExpiresAt) {
		return errTokenExpired
	}
	if record.Scope != expected {
		return errScopeMismatch
	}
	if !compareSecret(&record, id, secret) {
		return errTokenNotFound
	}

	record.Consumed = true
	updatedPayload, err := json.Marshal(&record)
	if err != nil {
		return fmt.Errorf("encode token record: %w", err)
	}

	if stored.Annotations == nil {
		stored.Annotations = map[string]string{}
	}
	stored.Annotations[annotationConsumedAt] = now.UTC().Format(time.RFC3339)
	stored.Data[dataKeyRecord] = updatedPayload

	if _, err := s.client.CoreV1().Secrets(s.namespace).Update(ctx, stored, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("mark token consumed: %w", err)
	}

	return nil
}

func tokenSecretName(id string) string {
	return secretPrefix + id
}
