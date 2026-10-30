package util

import "testing"

// BRUN-7078: an email stored with a zero-width space fails IsEmailValid and blocked sign-up.
func TestCleanUpEmailStripsInvisibleChars(t *testing.T) {
	cases := map[string]string{
		"jdoe\u200bsmith@example.com":    "jdoesmith@example.com",
		"\ufeff JDoe@Example.com\u2060 ": "jdoe@example.com",
		"j\u200cd\u200do\u00ade@ex.com":  "jdoe@ex.com",
		"jdoe@example.con":               "jdoe@example.com",
		"jdoe@example.com":               "jdoe@example.com",
		"":                               "",
	}
	for in, want := range cases {
		got := CleanUpEmail(in)
		if got != want {
			t.Errorf("CleanUpEmail(%q) = %q, want %q", in, got, want)
		}
		if want != "" && !IsEmailValid(got) {
			t.Errorf("IsEmailValid(%q) = false after CleanUpEmail", got)
		}
	}
	if IsEmailValid("jdoe\u200bsmith@example.com") {
		t.Error("expected raw zero-width email to fail IsEmailValid")
	}
}
