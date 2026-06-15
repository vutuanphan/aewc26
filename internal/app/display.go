package app

import (
	"fmt"
	"strconv"
	"strings"
)

// vnd formats an integer with thousands separators (e.g. 100.000).
func vnd(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, ".")
	if neg {
		return "-" + out
	}
	return out
}

func odd(f float64) string {
	if f <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.2f", f)
}

func pct(f float64) string {
	if f <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", f*100)
}

func fmtLineStr(n float64) string {
	s := strconv.FormatFloat(n, 'f', -1, 64)
	if n > 0 {
		return "+" + s
	}
	return s
}

func numStr(n float64) string { return strconv.FormatFloat(n, 'f', -1, 64) }

func typeLabel(lang, t string) string {
	switch t {
	case "wdl":
		return "1x2"
	case "ah":
		if lang == "vi" {
			return "Chấp"
		}
		return "AH"
	case "ou":
		if lang == "vi" {
			return "Tài/Xỉu"
		}
		return "O/U"
	case "cs":
		if lang == "vi" {
			return "Tỷ số"
		}
		return "CS"
	}
	return t
}

// creatorSide describes what the bet creator backs.
func creatorSide(lang string, b Bet) string {
	h, a := b.Match.HomeDisplay(), b.Match.AwayDisplay()
	switch b.BetType {
	case "wdl":
		switch b.Pick {
		case "home":
			return h + " " + tr(lang, "w.win")
		case "away":
			return a + " " + tr(lang, "w.win")
		default:
			return tr(lang, "w.draw")
		}
	case "ah":
		if b.Pick == "home" {
			return h + " " + fmtLineStr(b.Line)
		}
		return a + " " + fmtLineStr(b.Line)
	case "ou":
		if b.Pick == "over" {
			return tr(lang, "w.over") + " " + numStr(b.Line)
		}
		return tr(lang, "w.under") + " " + numStr(b.Line)
	case "cs":
		return tr(lang, "w.score") + " " + b.Pick
	}
	return ""
}

// takerSide describes the opposite side the taker backs.
func takerSide(lang string, b Bet) string {
	h, a := b.Match.HomeDisplay(), b.Match.AwayDisplay()
	win := tr(lang, "w.win")
	switch b.BetType {
	case "wdl":
		switch b.Pick {
		case "home":
			return tr(lang, "w.draw") + " / " + a + " " + win
		case "away":
			return h + " " + win + " / " + tr(lang, "w.draw")
		default:
			return h + " " + tr(lang, "w.or") + " " + a + " " + win
		}
	case "ah":
		if b.Pick == "home" {
			return a + " " + fmtLineStr(-b.Line)
		}
		return h + " " + fmtLineStr(-b.Line)
	case "ou":
		if b.Pick == "over" {
			return tr(lang, "w.under") + " " + numStr(b.Line)
		}
		return tr(lang, "w.over") + " " + numStr(b.Line)
	case "cs":
		return tr(lang, "w.notscore") + " " + b.Pick
	}
	return ""
}

func outcomeLabel(lang, o string) string {
	if o == "" {
		return ""
	}
	return tr(lang, "o."+o)
}

func txnKindLabel(lang, k string) string {
	return tr(lang, "k."+k)
}
