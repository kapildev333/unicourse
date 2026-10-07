package tenant

import (
	"encoding/json"
	"testing"
)

func TestDecimal(t *testing.T) {
	for _, c := range []struct {
		in       string
		places   int
		min, max string
		want     string // value, or the error message
	}{
		{"87.5", 2, "0", "100", "87.50"},
		{"3", 1, "0", "20", "3.0"},
		{"0", 2, "0", "100", "0.00"},
		{"100.00", 2, "0", "100", "100.00"},
		{"100.01", 2, "0", "100", "must be between 0 and 100"},
		{"-1", 2, "0", "100", "must be between 0 and 100"},
		{"1.234", 2, "0", "100", "must have at most 2 decimals"},
		{"1.25", 1, "0", "20", "must have at most 1 decimal"},
		{"1.20", 1, "0", "20", "1.2"},
		{"", 1, "0", "20", "is required"},
		{" ", 1, "0", "20", "is required"},
		{"1e2", 2, "0", "100", "must be a number"},
		{"0x10", 2, "0", "100", "must be a number"},
		{"1/2", 2, "0", "100", "must be a number"},
		{"abc", 2, "0", "100", "must be a number"},
		{".5", 2, "0", "100", "must be a number"},
	} {
		got, fe := Decimal("f", json.Number(c.in), c.places, c.min, c.max)
		if fe != nil {
			got = fe.Message
		}
		if got != c.want {
			t.Errorf("Decimal(%q, %d) = %q, want %q", c.in, c.places, got, c.want)
		}
	}
}

func TestOpt(t *testing.T) {
	var in struct {
		A, B, C Opt[int]
	}
	if err := json.Unmarshal([]byte(`{"B":null,"C":5}`), &in); err != nil {
		t.Fatal(err)
	}
	if in.A.Set || !in.B.Set || !in.B.Null || !in.C.Set || in.C.Null || in.C.V != 5 {
		t.Errorf("got %+v", in)
	}
	if err := json.Unmarshal([]byte(`{"A":"x"}`), &in); err == nil {
		t.Error("string into Opt[int] accepted")
	}
}
