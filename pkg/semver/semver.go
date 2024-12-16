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

func Build(v string) string {
	return semver.Build(v)
}

func Canonical(v string) string {
	return semver.Canonical(v)
}

func Compare(v1, v2 string) int {
	return semver.Compare(v1, v2)
}

func IsValid(v string) bool {
	return semver.IsValid(v)
}

func Major(v string) string {
	return semver.Major(v)
}

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

func MajorMinor(v string) string {
	return semver.MajorMinor(v)
}

// releaseType is one of three values: "major", "minor", or "patch"
// if releaseType is not one of the three values, the function will return an error
func Increment(v, releaseType string) (string, error) {
	if !IsValid(v) {
		return "", errors.New("invalid version format")
	}

	parts := strings.Split(v, ".")
	switch len(parts) { // handle the cases where minor or patch version wasn't included
	case 1:
		parts = append(parts, "0", "0")
	case 2:
		parts = append(parts, "0")
	}

	if RELEASE_TYPES[releaseType] == MAJOR_RELEASE {
		parts[0] = parts[0][1:] // strip the 'v' if major version
	}

	toIncrement, err := strconv.Atoi(parts[RELEASE_TYPES[releaseType]])
	if err != nil {
		return "", err
	}
	toIncrement++

	parts[RELEASE_TYPES[releaseType]] = strconv.Itoa(toIncrement)

	if RELEASE_TYPES[releaseType] == PATCH {
		return strings.Join(parts, "."), nil
	} else if RELEASE_TYPES[releaseType] == MINOR_RELEASE {
		return fmt.Sprintf("%s.%s.0", parts[MAJOR_RELEASE], parts[MINOR_RELEASE]), nil
	}
	return fmt.Sprintf("v%s.0.0", parts[MAJOR_RELEASE]), nil // add the 'v' back for major version
}

// We don't use this for now, but it's included just in case
func Prerelease(v string) string {
	return semver.Prerelease(v)
}

func Sort(vs []string) {
	semver.Sort(vs)
}
