package tomlsawyer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// wantGolden compares got and want, printing a unified diff on mismatch.
func wantGolden(t *testing.T, got, want string) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}
