package validation_test

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

	expectedIssues := []string{
		"require >= 4 cpu cores, detected 2",
		"require >= 8 GiB memory, detected 4 GiB",
		"missing kernel module br_netfilter",
		"requires sudo privileges",
		"path missing: /nonexistent",
	}
	for _, expected := range expectedIssues {
		found := false
		for _, actual := range result.Issues {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected issue %q not found in actual issues: %v", expected, result.Issues)
		}
	}
}
