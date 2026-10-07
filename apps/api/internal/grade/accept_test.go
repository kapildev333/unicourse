package grade

import "testing"

func TestWantsCSV(t *testing.T) {
	for accept, want := range map[string]bool{
		"":                                     false,
		"text/csv":                             true,
		"application/json":                     false,
		"text/csv;q=0":                         false,
		"text/csv;q=0.1, application/json":     false,
		"application/json;q=0.2, text/csv;q=1": true,
		"text/csv, application/json":           true, // tie goes to the one the caller named for the export
		"TEXT/CSV; q=0.5":                      true,
		"*/*":                                  false,
		"text/csv;q=bogus":                     true, // unparsable q counts as 1
	} {
		if got := wantsCSV(accept); got != want {
			t.Errorf("%q: got %v want %v", accept, got, want)
		}
	}
}
