package ai

import (
	"testing"
)

func TestBedrockResponds(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"Hello"},
	}

	for _, test := range tests {
		result, err := InvokeAWSBedrockAgent(test.input)
		if result == "" || err != nil {
			t.Errorf("InvokeAWSBedrockAgent(%s) returned nil; expected not nil", test.input)
		}
	}
}
