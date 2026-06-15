package app

import "testing"

func TestCreatorFractionAndPayouts(t *testing.T) {
	const stake = 100
	cases := []struct {
		name     string
		betType  string
		pick     string
		line     float64
		ftH, ftA int
		wantF    float64
		wantCR   int64
		wantTR   int64
		wantOut  string
	}{
		{"1x2 home win", "wdl", "home", 0, 2, 0, 1, 200, 0, "creator"},
		{"1x2 home draw->lose", "wdl", "home", 0, 1, 1, -1, 0, 200, "taker"},
		{"AH -1 win by 2", "ah", "home", -1.0, 2, 0, 1, 200, 0, "creator"},
		{"AH -1 win by 1 push", "ah", "home", -1.0, 1, 0, 0, 100, 100, "push"},
		{"AH -1 draw lose", "ah", "home", -1.0, 1, 1, -1, 0, 200, "taker"},
		{"AH -0.25 draw half-lose", "ah", "home", -0.25, 0, 0, -0.5, 50, 150, "taker_half"},
		{"AH +0.25 draw half-win", "ah", "home", 0.25, 0, 0, 0.5, 150, 50, "creator_half"},
		{"OU over 2.5 total 3", "ou", "over", 2.5, 2, 1, 1, 200, 0, "creator"},
		{"OU under 2.5 total 3", "ou", "under", 2.5, 2, 1, -1, 0, 200, "taker"},
		{"OU over 3.0 push", "ou", "over", 3.0, 2, 1, 0, 100, 100, "push"},
		{"OU over 2.75 half-win", "ou", "over", 2.75, 2, 1, 0.5, 150, 50, "creator_half"},
		{"CS exact hit", "cs", "2-1", 0, 2, 1, 1, 200, 0, "creator"},
		{"CS miss", "cs", "2-1", 0, 1, 1, -1, 0, 200, "taker"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, err := creatorFraction(c.betType, c.pick, c.line, c.ftH, c.ftA)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f != c.wantF {
				t.Errorf("fraction = %v, want %v", f, c.wantF)
			}
			cr, tr, out := payouts(f, stake)
			if cr != c.wantCR || tr != c.wantTR {
				t.Errorf("payouts = (%d,%d), want (%d,%d)", cr, tr, c.wantCR, c.wantTR)
			}
			if out != c.wantOut {
				t.Errorf("outcome = %q, want %q", out, c.wantOut)
			}
			if cr+tr != 2*stake {
				t.Errorf("pot not conserved: %d+%d != %d", cr, tr, 2*stake)
			}
		})
	}
}

// The pot must always be fully distributed, even on odd stakes / half results.
func TestPayoutConservation(t *testing.T) {
	for _, stake := range []int64{1, 7, 99, 100, 333, 1000} {
		for _, f := range []float64{-1, -0.5, 0, 0.5, 1} {
			cr, tr, _ := payouts(f, stake)
			if cr+tr != 2*stake {
				t.Errorf("stake=%d f=%v: %d+%d != %d", stake, f, cr, tr, 2*stake)
			}
			if cr < 0 || tr < 0 {
				t.Errorf("stake=%d f=%v: negative payout (%d,%d)", stake, f, cr, tr)
			}
		}
	}
}

func TestValidPick(t *testing.T) {
	ok := []struct{ bt, p string }{{"wdl", "home"}, {"wdl", "draw"}, {"wdl", "away"}, {"ah", "home"}, {"ah", "away"}, {"ou", "over"}, {"ou", "under"}}
	for _, c := range ok {
		if !validPick(c.bt, c.p) {
			t.Errorf("validPick(%q,%q) = false, want true", c.bt, c.p)
		}
	}
	bad := []struct{ bt, p string }{{"wdl", "over"}, {"ah", "draw"}, {"ou", "home"}, {"xx", "home"}}
	for _, c := range bad {
		if validPick(c.bt, c.p) {
			t.Errorf("validPick(%q,%q) = true, want false", c.bt, c.p)
		}
	}
}

func TestQuarterStep(t *testing.T) {
	for _, v := range []float64{0, 0.25, -0.5, 1.75, -2.25} {
		if !isQuarterStep(v) {
			t.Errorf("isQuarterStep(%v) = false", v)
		}
	}
	for _, v := range []float64{0.1, 0.3, 1.2} {
		if isQuarterStep(v) {
			t.Errorf("isQuarterStep(%v) = true", v)
		}
	}
}
