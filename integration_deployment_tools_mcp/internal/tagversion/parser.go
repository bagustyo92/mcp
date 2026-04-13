package tagversion

import (
	"fmt"
	"regexp"
	"strconv"
)

var semverRegex = regexp.MustCompile(`^(.*?)(v?)(\d+)\.(\d+)\.(\d+)(.*)$`)

// Semver represents a parsed semantic version.
type Semver struct {
	Prefix string
	V      string
	Major  int
	Minor  int
	Patch  int
	Suffix string
}

// String returns the string representation of the semver.
func (s Semver) String() string {
	return fmt.Sprintf("%s%s%d.%d.%d%s", s.Prefix, s.V, s.Major, s.Minor, s.Patch, s.Suffix)
}

// ParseSemver parses a tag string into a Semver struct.
func ParseSemver(tag string) (Semver, error) {
	matches := semverRegex.FindStringSubmatch(tag)
	if matches == nil {
		return Semver{}, fmt.Errorf("tag %q does not match semver pattern", tag)
	}

	major, _ := strconv.Atoi(matches[3])
	minor, _ := strconv.Atoi(matches[4])
	patch, _ := strconv.Atoi(matches[5])

	return Semver{
		Prefix: matches[1],
		V:      matches[2],
		Major:  major,
		Minor:  minor,
		Patch:  patch,
		Suffix: matches[6],
	}, nil
}

// IncrementPatch returns the tag string with patch version incremented by 1.
// The suffix is dropped on increment.
func IncrementPatch(s Semver) string {
	return fmt.Sprintf("%s%s%d.%d.%d", s.Prefix, s.V, s.Major, s.Minor, s.Patch+1)
}

// IncrementMinor returns the tag string with minor version incremented by 1
// and patch reset to 0. The suffix is dropped on increment.
func IncrementMinor(s Semver) string {
	return fmt.Sprintf("%s%s%d.%d.%d", s.Prefix, s.V, s.Major, s.Minor+1, 0)
}
