package knowledge

import (
	"math"
	"testing"
)

func TestWilsonScore_NoObservations(t *testing.T) {
	t.Parallel()

	got := WilsonScore(0, 0)
	if got != 0.5 {
		t.Errorf("WilsonScore(0, 0) = %v, want 0.5", got)
	}
}

func TestWilsonScore_AllSuccesses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		useCount   int
		missCount  int
		wantApprox float64
	}{
		// WilsonScore(1, 0): (1 + 3.8416/2 - 1.96*sqrt(0 + 3.8416/4)) / (1 + 3.8416)
		//   = (2.9208 - 1.96*0.9800) / 4.8416
		//   = 1.0 / 4.8416 ≈ 0.2066
		{useCount: 1, missCount: 0, wantApprox: 0.2066},
		// WilsonScore(3, 0): (1 + 3.8416/6 - 1.96*sqrt(0 + 3.8416/36)) / (1 + 3.8416/3)
		//   = (1.6403 - 1.96*0.3267) / 2.2805 ≈ 1.0 / 2.2805 ≈ 0.4385
		{useCount: 3, missCount: 0, wantApprox: 0.4385},
		// WilsonScore(10, 0): (1 + 3.8416/20 - 1.96*sqrt(0 + 3.8416/400)) / (1 + 3.8416/10)
		//   = 1.0 / 1.38416 ≈ 0.7225
		{useCount: 10, missCount: 0, wantApprox: 0.7225},
	}

	for _, tc := range tests {
		got := WilsonScore(tc.useCount, tc.missCount)
		if math.Abs(got-tc.wantApprox) > 0.001 {
			t.Errorf("WilsonScore(%d, %d) = %.4f, want ≈ %.4f",
				tc.useCount, tc.missCount, got, tc.wantApprox)
		}
	}
}

func TestWilsonScore_AllFailures(t *testing.T) {
	t.Parallel()

	// WilsonScore(0, 1): p̂=0, n=1
	// = (0 + 3.8416/2 - 1.96*sqrt(0 + 3.8416/4)) / (1 + 3.8416)
	// = (1.9208 - 1.9208) / 4.8416 = 0.0
	got := WilsonScore(0, 1)
	if math.Abs(got) > 0.001 {
		t.Errorf("WilsonScore(0, 1) = %.4f, want ≈ 0.0", got)
	}
}

func TestWilsonScore_Mixed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		useCount   int
		missCount  int
		wantApprox float64
	}{
		// WilsonScore(5, 5): p̂=0.5, n=10
		// = (0.5 + 3.8416/20 - 1.96*sqrt(0.025 + 3.8416/400)) / (1 + 3.8416/10)
		// = (0.69208 - 1.96*sqrt(0.034604)) / 1.38416
		// = (0.69208 - 1.96*0.18601) / 1.38416
		// = 0.32750 / 1.38416 ≈ 0.2366
		{useCount: 5, missCount: 5, wantApprox: 0.2366},
		// WilsonScore(1, 2): p̂=0.333, n=3
		// = (0.333 + 3.8416/6 - 1.96*sqrt(0.333*0.667/3 + 3.8416/36)) / (1 + 3.8416/3)
		// ≈ (0.973 - 1.96*0.425) / 2.281 ≈ 0.14 / 2.281 ≈ 0.0613
		{useCount: 1, missCount: 2, wantApprox: 0.0613},
		// WilsonScore(9, 1): p̂=0.9, n=10
		// = (0.9 + 3.8416/20 - 1.96*sqrt(0.9*0.1/10 + 3.8416/400)) / (1 + 3.8416/10)
		// = (1.09208 - 1.96*sqrt(0.009 + 0.009604)) / 1.38416
		// = (1.09208 - 1.96*0.13638) / 1.38416
		// = (1.09208 - 0.26730) / 1.38416 ≈ 0.82478 / 1.38416 ≈ 0.5959
		{useCount: 9, missCount: 1, wantApprox: 0.5959},
	}

	for _, tc := range tests {
		got := WilsonScore(tc.useCount, tc.missCount)
		if math.Abs(got-tc.wantApprox) > 0.002 {
			t.Errorf("WilsonScore(%d, %d) = %.4f, want ≈ %.4f",
				tc.useCount, tc.missCount, got, tc.wantApprox)
		}
	}
}

func TestWilsonScore_MonotonicallyIncreasing(t *testing.T) {
	t.Parallel()

	// With fixed miss_count=0, more uses → higher confidence
	prev := WilsonScore(1, 0)
	for uses := 2; uses <= 20; uses++ {
		curr := WilsonScore(uses, 0)
		if curr <= prev {
			t.Errorf("WilsonScore(%d, 0) = %.4f should be > WilsonScore(%d, 0) = %.4f",
				uses, curr, uses-1, prev)
		}
		prev = curr
	}
}

func TestWilsonScore_BoundedBetweenZeroAndOne(t *testing.T) {
	t.Parallel()

	cases := [][2]int{
		{0, 0}, {1, 0}, {0, 1}, {10, 0}, {0, 10},
		{5, 5}, {100, 1}, {1, 100}, {50, 50},
	}
	for _, c := range cases {
		got := WilsonScore(c[0], c[1])
		if got < 0 || got > 1 {
			t.Errorf("WilsonScore(%d, %d) = %.4f, want in [0, 1]", c[0], c[1], got)
		}
	}
}
