package knowledge

import "math"

// clamp returns v clamped to the range [lo, hi].
func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

// WilsonScore computes the Wilson score lower bound for a binomial proportion
// confidence interval at 95% confidence (z=1.96).
//
// use_count is the number of successful retrievals; miss_count is the number of
// times the entry was flagged as wrong or unhelpful.
//
// Returns 0.5 when there are no observations (n=0).
func WilsonScore(useCount, missCount int) float64 {
	n := useCount + missCount
	if n == 0 {
		return 0.5
	}

	p := float64(useCount) / float64(n)
	z := 1.96
	nf := float64(n)
	z2 := z * z

	numerator := p + z2/(2*nf) - z*math.Sqrt(p*(1-p)/nf+z2/(4*nf*nf))
	denominator := 1 + z2/nf

	return clamp(numerator/denominator, 0.0, 1.0)
}
