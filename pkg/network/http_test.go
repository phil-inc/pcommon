package network

import (
	"errors"
	"fmt"
	"testing"
)

func TestGetCodeFromErrResp(t *testing.T) {
	tests := []struct {
		input string
		code  int
		err   bool
	}{
		{
			input: "Http response NOT_OK. Status: Error Status, Code:404",
			code:  404,
			err:   false,
		},
		{
			input: "Http response NOT_OK. Status: Error Status, Code:200",
			code:  200,
			err:   false,
		},
		{
			input: "No code in this response",
			code:  0,
			err:   true,
		},
		{
			input: "Invalid Code value. Code:invalid",
			code:  0,
			err:   true,
		},
	}

	for _, test := range tests {
		code := GetStatusCodeFromError(errors.New(test.input))
		if code != test.code {
			t.Errorf("Test failed for input %s", test.input)
		}
		fmt.Printf("Input: %s, Output: %d\n", test.input, code)
	}
}
