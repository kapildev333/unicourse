package grade

import (
	"encoding/json"
	"math/big"
)

type part struct{ weight, max, points *big.Rat }

// percentOf = 100 x sum(weight x points/max) / sum(weight), exact; false when the weights sum to 0.
// A 0-weight part contributes nothing.
func percentOf(parts []part) (*big.Rat, bool) {
	sum, w := new(big.Rat), new(big.Rat)
	for _, p := range parts {
		w.Add(w, p.weight)
		t := new(big.Rat).Quo(p.points, p.max)
		sum.Add(sum, t.Mul(t, p.weight))
	}
	if w.Sign() == 0 {
		return nil, false
	}
	sum.Quo(sum, w)
	return sum.Mul(sum, big.NewRat(100, 1)), true
}

// round2 rounds half away from zero, which is half up for the non-negative values used here.
func round2(r *big.Rat) json.Number { return json.Number(r.FloatString(2)) }
