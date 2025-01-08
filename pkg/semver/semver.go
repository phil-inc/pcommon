package semver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
)

type ReleaseType int

const (
	MAJOR_RELEASE ReleaseType = iota
	MINOR_RELEASE
	PATCH
)

var RELEASE_TYPES = map[string]ReleaseType{
	"major": MAJOR_RELEASE,
	"minor": MINOR_RELEASE,
	"patch": PATCH,
}

// Build returns the build suffix of the semantic version v. For example, Build("v2.1.0+meta") == "+meta".
// If v is an invalid semantic version string, Build returns the empty string.
func Build(v string) string {
	return semver.Build(v)
}

// Canonical returns the canonical formatting of the semantic version v. It fills in any missing .MINOR or .PATCH and discards build metadata.
// Two semantic versions compare equal only if their canonical formattings are identical strings. The canonical invalid semantic version is the empty string.
func Canonical(v string) string {
	return semver.Canonical(v)
}

// Compare returns an integer comparing two versions according to semantic version precedence. The result will be 0 if v == w, -1 if v < w, or +1 if v > w.
// An invalid semantic version string is considered less than a valid one. All invalid semantic version strings compare equal to each other.
func Compare(v1, v2 string) int {
	return semver.Compare(v1, v2)
}

// IsValid reports whether v is a valid semantic version string.
func IsValid(v string) bool {
	return semver.IsValid(v)
}

// Major returns the major version prefix of the semantic version v. For example, Major("v2.1.0") == "v2".
// If v is an invalid semantic version string, Major returns the empty string.
func Major(v string) string {
	return semver.Major(v)
}

// Minor returns the minor version suffix of the semantic version v. For example, Minor("v2.1.0") == "1".
// If v is an invalid semantic version string, Minor returns the empty string.
// If v is a valid semantic version string but does not have a minor version, Minor returns "0".
func Minor(v string) string {
	if !IsValid(v) {
		return ""
	}
	parts := strings.Split(v, ".")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "0"
}

// Patch returns the patch version suffix of the semantic version v. For example, Patch("v1.2.3") == "3".
// If v is an invalid semantic version string, Patch returns the empty string.
// If v is a valid semantic version string but does not have a patch version, Patch returns "0".
func Patch(v string) string {
	if !IsValid(v) {
		return ""
	}
	parts := strings.Split(v, ".")
	if len(parts) == 3 {
		return parts[2]
	}
	return "0"
}

// MajorMinor returns the major.minor version prefix of the semantic version v. For example, MajorMinor("v2.1.0") == "v2.1".
// If v is an invalid semantic version string, MajorMinor returns the empty string.
func MajorMinor(v string) string {
	return semver.MajorMinor(v)
}

// Increment increments the specified part of the semantic version v.
// The releaseType parameter specifies which part to increment: "major", "minor", or "patch".
// If v is an invalid semantic version string, Increment returns an error.
// If releaseType is not one of "major", "minor", or "patch", Increment returns an error.
// If the minor or patch version is not included in v, they are assumed to be "0".
func Increment(v, releaseTypeStr string) (string, error) {
	if !IsValid(v) {
		return "", errors.New("invalid version format")
	}
	releaseType, ok := RELEASE_TYPES[releaseTypeStr]
	if !ok {
		return "", errors.New("invalid release type")
	}

	parts := strings.Split(v, ".")
	switch len(parts) { // handle the cases where minor or patch version wasn't included
	case 1:
		parts = append(parts, "0", "0")
	case 2:
		parts = append(parts, "0")
	}

	if releaseType == MAJOR_RELEASE {
		parts[0] = parts[0][1:] // strip the 'v' if major version
	}

	toIncrement, err := strconv.Atoi(parts[releaseType])
	if err != nil {
		return "", err
	}
	toIncrement++

	parts[releaseType] = strconv.Itoa(toIncrement)

	if releaseType == PATCH {
		return strings.Join(parts, "."), nil
	} else if releaseType == MINOR_RELEASE {
		return fmt.Sprintf("%s.%s.0", parts[MAJOR_RELEASE], parts[MINOR_RELEASE]), nil
	}
	return fmt.Sprintf("v%s.0.0", parts[MAJOR_RELEASE]), nil // add the 'v' back for major version
}

// Prerelease returns the prerelease suffix of the semantic version v. For example, Prerelease("v2.1.0-pre+meta") == "-pre".
// If v is an invalid semantic version string, Prerelease returns the empty string.
func Prerelease(v string) string {
	return semver.Prerelease(v)
}

// Sort sorts a list of semantic version strings using ByVersion (https://pkg.go.dev/golang.org/x/mod/semver#ByVersion).
func Sort(vs []string) {
	semver.Sort(vs)
}
