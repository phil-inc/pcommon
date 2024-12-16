package semver

import (
	"testing"
)

func TestMajor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.2.3", "v1"},
		{"v2.0.0", "v2"},
		{"v0.1.0", "v0"},
		{"v10.20.30", "v10"},
		{"v1.2", "v1"},
		{"v1.2.x", ""},
		{"1.2.3", ""},
		{"invalid", ""},
	}

	for _, test := range tests {
		result := Major(test.input)
		if result != test.expected {
			t.Errorf("Major(%s) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestMinor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.2.3", "2"},
		{"v2.0.0", "0"},
		{"v0.1.0", "1"},
		{"v10.20.30", "20"},
		{"v1.2", "2"},
		{"v1", "0"},
		{"v1.2.x", ""},
		{"1.2.3", ""},
		{"invalid", ""},
	}

	for _, test := range tests {
		result := Minor(test.input)
		if result != test.expected {
			t.Errorf("Minor(%s) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestPatch(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.2.3", "3"},
		{"v2.0.0", "0"},
		{"v0.1.0", "0"},
		{"v10.20.30", "30"},
		{"v1.2", "0"},
		{"v1", "0"},
		{"v1.2.x", ""},
		{"1.2.3", ""},
		{"invalid", ""},
	}

	for _, test := range tests {
		result := Patch(test.input)
		if result != test.expected {
			t.Errorf("Patch(%s) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestIncrement(t *testing.T) {
	tests := []struct {
		inputVersion     string
		inputReleaseType string
		expected         string
	}{
		{"v1.2.3", "patch", "v1.2.4"},
		{"v1.2.9", "patch", "v1.2.10"},
		{"v1.2.99", "patch", "v1.2.100"},
		{"v1.2.3", "minor", "v1.3.0"},
		{"v1.2.3", "major", "v2.0.0"},
		{"v1", "patch", "v1.0.1"},
		{"v1", "minor", "v1.1.0"},
		{"v1.2.x", "patch", ""},
		{"invalid", "patch", ""},
	}

	for _, test := range tests {
		result, err := Increment(test.inputVersion, test.inputReleaseType)
		if result != test.expected {
			t.Errorf("Increment(%s, %s) = %s; expected %s", test.inputVersion, test.inputReleaseType, result, test.expected)
		}
		if test.expected == "" && err == nil { // all invalid cases should return an error
			t.Errorf("Increment(%s, %s) did not return an error", test.inputVersion, test.inputReleaseType)
		}
	}
}
