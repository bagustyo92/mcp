package tagversion

import (
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Semver
		wantErr bool
	}{
		{"simple", "v1.2.3", Semver{V: "v", Major: 1, Minor: 2, Patch: 3}, false},
		{"no-v", "1.2.3", Semver{Major: 1, Minor: 2, Patch: 3}, false},
		{"with-prefix", "release-v1.0.0", Semver{Prefix: "release-", V: "v", Major: 1, Minor: 0, Patch: 0}, false},
		{"with-suffix", "v1.0.0-rc1", Semver{V: "v", Major: 1, Minor: 0, Patch: 0, Suffix: "-rc1"}, false},
		{"large-numbers", "v10.20.300", Semver{V: "v", Major: 10, Minor: 20, Patch: 300}, false},
		{"prefix-and-suffix", "app-v2.1.0-beta", Semver{Prefix: "app-", V: "v", Major: 2, Minor: 1, Patch: 0, Suffix: "-beta"}, false},
		{"zeros", "v0.0.0", Semver{V: "v", Major: 0, Minor: 0, Patch: 0}, false},
		{"invalid", "not-a-version", Semver{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSemver(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSemver(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseSemver(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIncrementPatch(t *testing.T) {
	tests := []struct {
		name  string
		input Semver
		want  string
	}{
		{"simple", Semver{V: "v", Major: 1, Minor: 2, Patch: 3}, "v1.2.4"},
		{"no-v", Semver{Major: 1, Minor: 0, Patch: 0}, "1.0.1"},
		{"with-prefix", Semver{Prefix: "release-", V: "v", Major: 1, Minor: 0, Patch: 0}, "release-v1.0.1"},
		{"drops-suffix", Semver{V: "v", Major: 1, Minor: 0, Patch: 0, Suffix: "-rc1"}, "v1.0.1"},
		{"large-patch", Semver{V: "v", Major: 1, Minor: 0, Patch: 99}, "v1.0.100"},
		{"zeros", Semver{V: "v", Major: 0, Minor: 0, Patch: 0}, "v0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IncrementPatch(tt.input)
			if got != tt.want {
				t.Errorf("IncrementPatch(%+v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIncrementMinor(t *testing.T) {
	tests := []struct {
		name  string
		input Semver
		want  string
	}{
		{"simple", Semver{V: "v", Major: 1, Minor: 2, Patch: 3}, "v1.3.0"},
		{"no-v", Semver{Major: 1, Minor: 0, Patch: 5}, "1.1.0"},
		{"with-prefix", Semver{Prefix: "release-", V: "v", Major: 1, Minor: 0, Patch: 0}, "release-v1.1.0"},
		{"drops-suffix", Semver{V: "v", Major: 1, Minor: 0, Patch: 0, Suffix: "-rc1"}, "v1.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IncrementMinor(tt.input)
			if got != tt.want {
				t.Errorf("IncrementMinor(%+v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
