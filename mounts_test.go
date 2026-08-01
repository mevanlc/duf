package main

import (
	"reflect"
	"testing"
)

func TestParseMinimumTotalSize(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    uint64
		wantErr bool
	}{
		{name: "unset", value: "", want: 0},
		{name: "bytes", value: "512", want: 512},
		{name: "human readable", value: "10G", want: 10 << 30},
		{name: "invalid", value: "10GB", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseMinimumTotalSize(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseMinimumTotalSize(%q) error = %v, wantErr %v", test.value, err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("parseMinimumTotalSize(%q) = %d, want %d", test.value, got, test.want)
			}
		})
	}
}

func TestFilterMountsByTotalSize(t *testing.T) {
	mounts := []Mount{
		{Device: "smaller", Total: 99},
		{Device: "equal", Total: 100},
		{Device: "larger", Total: 101},
	}

	want := []Mount{
		{Device: "equal", Total: 100},
		{Device: "larger", Total: 101},
	}
	if got := filterMountsByTotalSize(mounts, 100); !reflect.DeepEqual(got, want) {
		t.Errorf("filterMountsByTotalSize() = %#v, want %#v", got, want)
	}

	if got := filterMountsByTotalSize(mounts, 0); !reflect.DeepEqual(got, mounts) {
		t.Errorf("filterMountsByTotalSize() with no minimum = %#v, want %#v", got, mounts)
	}
}
