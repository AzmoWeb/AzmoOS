package updater

import "testing"

func TestNewer(t *testing.T) {
	cases := []struct {
		l, c string
		want bool
	}{
		{"v0.2.0", "v0.1.0", true},
		{"v0.1.0", "v0.1.0", false},
		{"v0.1.9", "v0.2.0", false},
		{"v1.0.0", "dev", true},
		{"v0.10.0", "v0.9.3", true},
		{"nightly", "v0.1.0", false},
	}
	for _, c := range cases {
		if got := Newer(c.l, c.c); got != c.want {
			t.Errorf("Newer(%q,%q)=%v", c.l, c.c, got)
		}
	}
}
