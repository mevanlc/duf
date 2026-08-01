package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDereferenceMountPointPatterns(t *testing.T) {
	volumeDir := t.TempDir()
	targetDir := t.TempDir()
	firstVolume := filepath.Join(volumeDir, "first")
	secondVolume := filepath.Join(volumeDir, "second")
	for _, path := range []string{firstVolume, secondVolume} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatalf("create test volume %q: %v", path, err)
		}
	}
	if err := os.Symlink(targetDir, filepath.Join(volumeDir, "linked")); err != nil {
		t.Skipf("create test symlink: %v", err)
	}
	resolvedTargetDir, err := filepath.EvalSymlinks(targetDir)
	if err != nil {
		t.Fatalf("resolve test target %q: %v", targetDir, err)
	}

	mounts := []Mount{
		{Mountpoint: firstVolume},
		{Mountpoint: secondVolume},
		{Mountpoint: resolvedTargetDir},
	}
	pattern := filepath.Join(volumeDir, "*")
	patterns := map[string]struct{}{pattern: {}}
	countMatches := func() int {
		count := 0
		for _, mount := range mounts {
			if findInKey(mount.Mountpoint, patterns) {
				count++
			}
		}
		return count
	}
	if got := countMatches(); got != 2 {
		t.Fatalf("mount-point pattern matched %d volumes before dereferencing, want 2", got)
	}

	patterns, err = dereferenceMountPointPatterns(mounts, patterns)
	if err != nil {
		t.Fatalf("dereferenceMountPointPatterns() error = %v", err)
	}

	want := map[string]struct{}{
		pattern:           {},
		resolvedTargetDir: {},
	}
	if !reflect.DeepEqual(patterns, want) {
		t.Errorf("dereferenceMountPointPatterns() = %#v, want %#v", patterns, want)
	}
	if got := countMatches(); got != 3 {
		t.Errorf("mount-point pattern matched %d volumes after dereferencing, want 3", got)
	}
}
