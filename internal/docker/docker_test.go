package docker

import "testing"

func TestDemux(t *testing.T) {
	b := []byte{1, 0, 0, 0, 0, 0, 0, 3, 'a', 'b', '\n', 2, 0, 0, 0, 0, 0, 0, 2, 'c', '\n'}
	if got := demux(b); got != "ab\nc\n" {
		t.Fatalf("demux = %q", got)
	}
	if got := demux([]byte("tty brut\n")); got != "tty brut\n" {
		t.Fatalf("tty = %q", got)
	}
}
