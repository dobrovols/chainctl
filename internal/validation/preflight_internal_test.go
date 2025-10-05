package validation

import "testing"

func TestSplitLines(t *testing.T) {
	lines := splitLines("a\nb\n")
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestDefaultInspectorAccessors(t *testing.T) {
	inst := DefaultInspector{}
	if inst.CPUCount() <= 0 {
		t.Fatalf("expected cpu count > 0")
	}
	if inst.MemoryGiB() < 0 {
		t.Fatalf("expected non-negative memory")
	}
	const stubKernelModule = "any-kernel-module" // Used to test stub implementation always returns true
	if !inst.HasKernelModule(stubKernelModule) {
		t.Fatalf("expected stub kernel module check to return true")
	}
	_ = inst.HasSudoPrivileges()
}
