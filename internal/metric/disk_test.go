package metric

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestResolveDMNameFromMapperWithRoot(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sysfs dm tests are linux-only")
	}

	root := t.TempDir()
	name := "ubuntu--vg-ubuntu--lv"

	p := filepath.Join(root, "block", "dm-0", "dm")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p, "name"), []byte(name+"\n"), 0o600); err != nil {
		t.Fatalf("write name: %v", err)
	}

	dm, ok := resolveDMNameFromMapperWithRoot(name, root)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if dm != "dm-0" {
		t.Fatalf("expected dm-0, got %q", dm)
	}
}

func TestBuildDeviceKeyCandidates_AddsDMForMapper(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sysfs dm tests are linux-only")
	}

	old := sysfsRoot
	t.Cleanup(func() { sysfsRoot = old })

	root := t.TempDir()
	sysfsRoot = root

	name := "ubuntu--vg-ubuntu--lv"
	p := filepath.Join(root, "block", "dm-0", "dm")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p, "name"), []byte(name), 0o600); err != nil {
		t.Fatalf("write name: %v", err)
	}

	candidates := buildDeviceKeyCandidates("/dev/mapper/" + name)
	if !slices.Contains(candidates, "dm-0") {
		t.Fatalf("expected candidates to include dm-0, got %v", candidates)
	}
}
