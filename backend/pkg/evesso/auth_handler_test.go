package evesso

import (
	"os"
	"testing"
)

func TestCookieSecure_FailClosed(t *testing.T) {
	cases := []struct {
		env  string
		set  bool
		want bool
	}{
		{set: false, want: true},
		{env: "true", set: true, want: true},
		{env: "false", set: true, want: false},
		{env: "anything", set: true, want: true},
	}
	for _, c := range cases {
		os.Unsetenv("COOKIE_SECURE")
		if c.set {
			os.Setenv("COOKIE_SECURE", c.env)
		}
		if got := cookieSecure(); got != c.want {
			t.Errorf("COOKIE_SECURE=%q (set=%v): cookieSecure()=%v, want %v", c.env, c.set, got, c.want)
		}
	}
	os.Unsetenv("COOKIE_SECURE")
}
