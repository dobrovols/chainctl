//go:build tools

package tools

import (
	_ "go.opentelemetry.io/otel/sdk/metric"
	_ "helm.sh/helm/v3/pkg/action"
	_ "sigs.k8s.io/controller-runtime/pkg/envtest"
)
