package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// parseScore parses a correct-score pick like "2-1" into goals.
func parseScore(s string) (h, a int, ok bool) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	a, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if e1 != nil || e2 != nil || h < 0 || a < 0 || h > 99 || a > 99 {
		return 0, 0, false
	}
	return h, a, true
}

const eps = 1e-9

func sign3(x float64) float64 {
	if x > eps {
		return 1
	}
	if x < -eps {
		return -1
	}
	return 0
}

func isHalfStep(line float64) bool { return math.Abs(math.Mod(line*2, 1)) < eps }

// isQuarterStep reports whether a line is a valid betting line (multiple of 0.25).
func isQuarterStep(line float64) bool { return math.Abs(math.Mod(line*4, 1)) < eps }

// quarterSplit evaluates a possibly-quarter line by averaging the two adjacent
// half lines (eval returns +1/0/-1 for a concrete half/whole line).
func quarterSplit(line float64, eval func(float64) float64) float64 {
	if isHalfStep(line) {
		return eval(line)
	}
	return (eval(line-0.25) + eval(line+0.25)) / 2
}

// creatorFraction returns the creator's net result multiplier in
// {-1,-0.5,0,0.5,1} for a finished match.
func creatorFraction(betType, pick string, line float64, ftH, ftA int) (float64, error) {
	switch betType {
	case "wdl":
		outcome := "draw"
		if ftH > ftA {
			outcome = "home"
		} else if ftA > ftH {
			outcome = "away"
		}
		if outcome == pick {
			return 1, nil
		}
		return -1, nil
	case "ah":
		var diff float64
		switch pick {
		case "home":
			diff = float64(ftH - ftA)
		case "away":
			diff = float64(ftA - ftH)
		default:
			return 0, fmt.Errorf("bad ah pick")
		}
		return quarterSplit(line, func(l float64) float64 { return sign3(diff + l) }), nil
	case "ou":
		total := float64(ftH + ftA)
		switch pick {
		case "over":
			return quarterSplit(line, func(l float64) float64 { return sign3(total - l) }), nil
		case "under":
			return quarterSplit(line, func(l float64) float64 { return sign3(l - total) }), nil
		default:
			return 0, fmt.Errorf("bad ou pick")
		}
	case "cs":
		h, a, ok := parseScore(pick)
		if !ok {
			return 0, fmt.Errorf("bad cs pick")
		}
		if ftH == h && ftA == a {
			return 1, nil
		}
		return -1, nil
	}
	return 0, fmt.Errorf("bad bet type")
}

// payouts converts the creator fraction + stake into integer point returns.
// The pot (2*stake) is always fully distributed.
func payouts(fraction float64, stake int64) (creatorReturn, takerReturn int64, outcome string) {
	pot := 2 * stake
	creatorReturn = int64(math.Round(float64(stake) * (1 + fraction)))
	if creatorReturn < 0 {
		creatorReturn = 0
	}
	if creatorReturn > pot {
		creatorReturn = pot
	}
	takerReturn = pot - creatorReturn
	switch {
	case fraction > 0.75:
		outcome = "creator"
	case fraction > 0.25:
		outcome = "creator_half"
	case fraction > -0.25:
		outcome = "push"
	case fraction > -0.75:
		outcome = "taker_half"
	default:
		outcome = "taker"
	}
	return
}

func validPick(betType, pick string) bool {
	switch betType {
	case "wdl":
		return pick == "home" || pick == "draw" || pick == "away"
	case "ah":
		return pick == "home" || pick == "away"
	case "ou":
		return pick == "over" || pick == "under"
	case "cs":
		_, _, ok := parseScore(pick)
		return ok
	}
	return false
}
