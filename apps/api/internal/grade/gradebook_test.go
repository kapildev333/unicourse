package grade

import (
	"encoding/json"
	"math/big"
	"testing"
)

func pt(w, max, p int64) part {
	return part{big.NewRat(w, 1), big.NewRat(max, 1), big.NewRat(p, 1)}
}

func TestPercentOf(t *testing.T) {
	eq := func(name string, parts []part, want int64) {
		t.Helper()
		got, ok := percentOf(parts)
		if !ok || got.Cmp(big.NewRat(want, 1)) != 0 {
			t.Errorf("%s: got %v %v, want %d", name, got, ok, want)
		}
	}
	eq("only the scored 40", []part{pt(40, 100, 80)}, 80)
	eq("both scored", []part{pt(40, 100, 80), pt(60, 100, 90)}, 86)
	eq("0-weight part", []part{pt(40, 100, 80), pt(0, 100, 0)}, 80)
	eq("bonus", []part{pt(100, 100, 150)}, 150)
	third := []part{{big.NewRat(1, 3), big.NewRat(50, 1), big.NewRat(25, 1)}, {big.NewRat(2, 3), big.NewRat(50, 1), big.NewRat(25, 1)}}
	eq("thirds", third, 50)
	if _, ok := percentOf([]part{pt(0, 100, 50)}); ok {
		t.Error("sum of weights 0 must be false")
	}
}

var testScale = func() []ScaleRow {
	var s []ScaleRow
	for _, r := range [][3]string{{"A", "93.00", "4.00"}, {"A-", "90.00", "3.70"}, {"B+", "87.00", "3.30"}, {"B", "83.00", "3.00"}, {"B-", "80.00", "2.70"},
		{"C+", "77.00", "2.30"}, {"C", "73.00", "2.00"}, {"C-", "70.00", "1.70"}, {"D+", "67.00", "1.30"}, {"D", "63.00", "1.00"}, {"D-", "60.00", "0.70"}, {"F", "0.00", "0.00"}} {
		s = append(s, ScaleRow{r[0], json.Number(r[1]), json.Number(r[2])})
	}
	return s
}()

func TestLetterBoundaries(t *testing.T) {
	r := func(s string) *big.Rat { v, _ := new(big.Rat).SetString(s); return v }
	for pct, want := range map[string]string{"92.99": "A-", "93": "A", "92.995": "A-", "59.99": "F", "60": "D-", "0": "F", "150": "A"} {
		if got, ok := letterFor(testScale, r(pct)); !ok || got.Letter != want {
			t.Errorf("%s: got %q %v, want %s", pct, got.Letter, ok, want)
		}
	}
	if _, ok := letterFor(nil, r("50")); ok {
		t.Error("empty scale must be false")
	}
	if round2(r("92.995")) != "93.00" {
		t.Error("92.995 must round to 93.00")
	}
}
