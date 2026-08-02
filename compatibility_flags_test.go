package main

import (
	"io"
	"testing"

	flag "github.com/spf13/pflag"
)

func TestDUCompatibilityFlags(t *testing.T) {
	flagSet := flag.NewFlagSet("du-compatibility", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	var threshold string
	var warnings bool
	var dereferenceSymlinks bool
	registerDUCompatibilityFlags(flagSet, &threshold, &warnings, &dereferenceSymlinks)

	args := []string{
		"-aAbcgklmnsSx",
		"-B", "1M",
		"-d3",
		"-I", "*.tmp",
		"-Xignore.txt",
		"-D",
		"-P",
		"-H",
		"-t", "10G",
		"-r",
		"path",
	}
	if err := flagSet.Parse(args); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if threshold != "10G" {
		t.Errorf("threshold = %q, want %q", threshold, "10G")
	}
	if !warnings {
		t.Error("warnings = false, want true")
	}
	if !dereferenceSymlinks {
		t.Error("dereferenceSymlinks = false after -D -P -H, want true")
	}
	if got := flagSet.Args(); len(got) != 1 || got[0] != "path" {
		t.Errorf("Args() = %q, want [path]", got)
	}

	for _, compatibilityFlag := range duNoopCompatibilityFlags {
		registeredFlag := flagSet.Lookup(compatibilityFlag.name)
		if registeredFlag == nil {
			t.Errorf("compatibility flag %q was not registered", compatibilityFlag.name)
			continue
		}
		if !registeredFlag.Hidden {
			t.Errorf("compatibility flag %q is visible", compatibilityFlag.name)
		}
	}

	if err := flagSet.Set("du-no-dereference", "true"); err != nil {
		t.Fatalf("Set(du-no-dereference) error = %v", err)
	}
	if dereferenceSymlinks {
		t.Error("dereferenceSymlinks = true after -P, want false")
	}

	for _, name := range []string{
		"du-threshold",
		"du-report-errors",
		"du-dereference-args",
		"du-dereference-command-line",
		"du-no-dereference",
	} {
		registeredFlag := flagSet.Lookup(name)
		if registeredFlag == nil {
			t.Errorf("mapped compatibility flag %q was not registered", name)
			continue
		}
		if !registeredFlag.Hidden {
			t.Errorf("mapped compatibility flag %q is visible", name)
		}
	}
}
