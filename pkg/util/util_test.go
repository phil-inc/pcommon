package util

import (
	"testing"
)

func TestInjectEnvToTheDomainStr(t *testing.T) {
	tests := []struct {
		name      string
		domainStr string
		env       string
		want      string
	}{
		// --- prod should return as-is ---
		{
			name:      "prod env returns prod domain",
			domainStr: "https://my.jnjdirect.com/login",
			env:       "prod",
			want:      "https://my.prod.jnjdirect.com/login",
		},
		{
			name:      "prod with philrx.com",
			domainStr: "https://philrx.com/",
			env:       "prod",
			want:      "https://philrx.com/prod",
		},

		// --- philrx.com special rule ---
		{
			name:      "philrx.com root no slash",
			domainStr: "https://philrx.com",
			env:       "dev",
			want:      "https://philrx.com/dev",
		},
		{
			name:      "philrx.com with slash",
			domainStr: "https://philrx.com/",
			env:       "stage",
			want:      "https://philrx.com/stage",
		},

		// --- generic rule: insert env after first label ---
		{
			name:      "simple domain insert env",
			domainStr: "https://my.jnjdirect.com/",
			env:       "dev",
			want:      "https://my.dev.jnjdirect.com/",
		},
		{
			name:      "domain with query",
			domainStr: "https://my.jnjdirect.com?abc=1",
			env:       "stage",
			want:      "https://my.stage.jnjdirect.com?abc=1",
		},

		// --- env sanitization ---
		{
			name:      "env with spaces",
			domainStr: "https://my.jnjdirect.com/",
			env:       "   DeV ",
			want:      "https://my.dev.jnjdirect.com/",
		},

		// --- invalid URLs ---
		{
			name:      "broken url",
			domainStr: "://broken",
			env:       "dev",
			want:      "",
		},
		{
			name:      "single label host",
			domainStr: "https://localhost/",
			env:       "dev",
			want:      "",
		},
		{
			name:      "not a url",
			domainStr: "not-a-url",
			env:       "dev",
			want:      "",
		},
		{
			name:      "empty domain",
			domainStr: "",
			env:       "dev",
			want:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := InjectEnvToTheDomainStr(tc.domainStr, tc.env)
			if got != tc.want {
				t.Errorf("InjectEnvToTheDomainStr(%q, %q) = %q; want %q",
					tc.domainStr, tc.env, got, tc.want)
			}
		})
	}
}
