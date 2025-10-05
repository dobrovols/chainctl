//go:build tools

package tools

// The following packages are imported with blank identifiers to ensure they are included in go.mod
// and available for tooling or side effects, even though they are not used directly in this file.
// That keeps those binaries in go.mod/go.sum so go mod tidy doesn’t drop them and teammates can run
// the same tooling after go install. Without such a file, build-time tools aren’t tracked and 
// environments drift.
import (
	_ "go.opentelemetry.io/otel/sdk/metric"
	_ "helm.sh/helm/v3/pkg/action"
	_ "sigs.k8s.io/controller-runtime/pkg/envtest"
)
