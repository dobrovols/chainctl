package unit

import (
	"testing"

	"github.com/dobrovols/chainctl/internal/validation"
)

type fakeInspector struct {
	cpu       int
	memoryGiB int
	modules   map[string]bool
	sudo      bool
}

func (f fakeInspector) CPUCount() int                 { return f.cpu }
func (f fakeInspector) MemoryGiB() int                { return f.memoryGiB }
func (f fakeInspector) HasKernelModule(m string) bool { return f.modules[m] }
func (f fakeInspector) HasSudoPrivileges() bool       { return f.sudo }

func TestValidateHostSuccess(t *testing.T) {
	inspector := fakeInspector{
		cpu:       8,
		memoryGiB: 16,
		modules: map[string]bool{
			"br_netfilter": true,
		},
		sudo: true,
	}

	result := validation.ValidateHost(validation.HostConfig{
		RequireSudo:   true,
		KernelModules: []string{"br_netfilter"},
		MinCPU:        4,
		MinMemoryGiB:  8,
	}, inspector)

	if !result.Passed {
		t.Fatalf("expected validation to pass: %#v", result.Issues)
	}
}

func TestValidateHostFailureAggregatesIssues(t *testing.T) {
	inspector := fakeInspector{
		cpu:       2,
		memoryGiB: 4,
		modules:   map[string]bool{},
		sudo:      false,
	}

	result := validation.ValidateHost(validation.HostConfig{
		RequireSudo:     true,
		KernelModules:   []string{"br_netfilter"},
		MinCPU:          4,
		MinMemoryGiB:    8,
		FilesystemPaths: []string{"/nonexistent"},
	}, inspector)

	if result.Passed {
		t.Fatalf("expected validation to fail")
	}

	if len(result.Issues) < 4 {
		t.Fatalf("expected multiple issues, got %v", result.Issues)
	}
}
