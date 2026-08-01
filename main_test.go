package main

import "testing"

func TestMountPointFilterPreservesCase(t *testing.T) {
	patterns := parseCommaSeparatedValues("/Volumes/D")

	if !findInKey("/Volumes/D", patterns) {
		t.Error("mount-point filter did not match a path with identical case")
	}
	if findInKey("/volumes/d", patterns) {
		t.Error("mount-point filter unexpectedly matched a path with different case")
	}
}

func TestCaseInsensitiveFilterNormalizesCase(t *testing.T) {
	values := parseCaseInsensitiveCommaSeparatedValues("LOCAL,UFSD_NTFS")

	for _, want := range []string{"local", "ufsd_ntfs"} {
		if _, ok := values[want]; !ok {
			t.Errorf("normalized filter values do not contain %q", want)
		}
	}
}
