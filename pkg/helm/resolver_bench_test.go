package helm

import (
	"context"
	"testing"

	"github.com/dobrovols/chainctl/pkg/bundle"
)

type benchPuller struct{}

func (benchPuller) Pull(ctx context.Context, ref string) (PullResult, error) {
	return PullResult{ChartPath: "/tmp/chart.tgz", Digest: "sha256:bench"}, nil
}

type benchBundleLoader struct{}

func (benchBundleLoader) Load(path, cache string) (*bundle.Bundle, error) {
	return &bundle.Bundle{}, nil
}

func BenchmarkResolverResolveOCI(b *testing.B) {
	resolver := NewResolver(benchPuller{}, benchBundleLoader{}.Load)
	opts := ResolveOptions{OCIReference: "oci://registry.example.com/apps/myapp:1.2.3"}
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		if _, err := resolver.Resolve(ctx, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolverResolveBundle(b *testing.B) {
	resolver := NewResolver(benchPuller{}, benchBundleLoader{}.Load)
	opts := ResolveOptions{BundlePath: "/tmp/bundle.tar", BundleCacheDir: "/tmp/cache"}
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		if _, err := resolver.Resolve(ctx, opts); err != nil {
			b.Fatal(err)
		}
	}
}
