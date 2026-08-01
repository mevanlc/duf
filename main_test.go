package main

import (
	"testing"

	flag "github.com/spf13/pflag"
)

func TestShortOptions(t *testing.T) {
	tests := []struct {
		name      string
		shorthand string
	}{
		{name: "dereference", shorthand: "L"},
		{name: "hide-fs", shorthand: "F"},
		{name: "hide-mp", shorthand: "U"},
		{name: "json", shorthand: "J"},
		{name: "only", shorthand: "i"},
		{name: "only-fs", shorthand: "f"},
		{name: "only-mp", shorthand: "u"},
		{name: "output", shorthand: "o"},
		{name: "sort", shorthand: "R"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registered := flag.Lookup(test.name)
			if registered == nil {
				t.Fatalf("flag %q is not registered", test.name)
			}
			if registered.Shorthand != test.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", test.name, registered.Shorthand, test.shorthand)
			}
		})
	}
}

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
