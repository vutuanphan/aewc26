package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const oddsBase = "https://api.the-odds-api.com/v4/sports/soccer_fifa_world_cup"

type apiOutcome struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Point float64 `json:"point"`
}
type apiMarket struct {
	Key      string       `json:"key"`
	Outcomes []apiOutcome `json:"outcomes"`
}
type apiBook struct {
	Key     string      `json:"key"`
	Markets []apiMarket `json:"markets"`
}
type apiScore struct {
	Name  string `json:"name"`
	Score string `json:"score"`
}
type apiEvent struct {
	ID         string     `json:"id"`
	HomeTeam   string     `json:"home_team"`
	AwayTeam   string     `json:"away_team"`
	Completed  bool       `json:"completed"`
	Scores     []apiScore `json:"scores"`
	Bookmakers []apiBook  `json:"bookmakers"`
}

// ---- team name matching ----

var accentFold = strings.NewReplacer(
	"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a", "å", "a",
	"è", "e", "é", "e", "ê", "e", "ë", "e",
	"ì", "i", "í", "i", "î", "i", "ï", "i",
	"ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o",
	"ù", "u", "ú", "u", "û", "u", "ü", "u",
	"ñ", "n", "ç", "c", "ý", "y", "ÿ", "y",
)

func normalizeName(s string) string {
	s = accentFold.Replace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "&", "and")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// aliases map normalized Odds-API names to normalized seed names.
var aliases = map[string]string{
	"unitedstates":     "usa",
	"korearepublic":    "southkorea",
	"congodr":          "drcongo",
	"czechia":          "czechrepublic",
	"iriran":           "iran",
	"capeverdeislands": "capeverde",
	"turkiye":          "turkey",
	"cotedivoire":      "ivorycoast",
}

func canon(name string) string {
	n := normalizeName(name)
	if a, ok := aliases[n]; ok {
		return a
	}
	return n
}

// pairIndex maps "homeFifa|awayFifa" -> matchID plus a normalized-name -> fifa map.
func (a *App) pairIndex() (map[string]int64, map[string]string) {
	normToFifa := map[string]string{}
	rows, _ := a.db.Query(`SELECT fifa,name FROM teams`)
	for rows.Next() {
		var f, n string
		rows.Scan(&f, &n)
		normToFifa[normalizeName(n)] = f
	}
	rows.Close()
	pair := map[string]int64{}
	mr, _ := a.db.Query(`SELECT id,home_fifa,away_fifa FROM matches WHERE home_fifa<>'' AND away_fifa<>''`)
	for mr.Next() {
		var id int64
		var h, aw string
		mr.Scan(&id, &h, &aw)
		pair[h+"|"+aw] = id
	}
	mr.Close()
	return pair, normToFifa
}

func resolveMatch(ev apiEvent, pair map[string]int64, normToFifa map[string]string) (int64, bool) {
	h := normToFifa[canon(ev.HomeTeam)]
	aw := normToFifa[canon(ev.AwayTeam)]
	if h == "" || aw == "" {
		return 0, false
	}
	id, ok := pair[h+"|"+aw]
	return id, ok
}

func (a *App) fetchEvents(ctx context.Context, path string, params map[string]string) ([]apiEvent, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, oddsBase+path, nil)
	q := req.URL.Query()
	q.Set("apiKey", a.oddsKey)
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if rem := resp.Header.Get("x-requests-remaining"); rem != "" {
		log.Printf("[odds] requests remaining: %s", rem)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var evs []apiEvent
	return evs, json.NewDecoder(resp.Body).Decode(&evs)
}

func pickMarket(ev apiEvent, key string) (apiMarket, bool) {
	var fb *apiMarket
	for _, bk := range ev.Bookmakers {
		for _, m := range bk.Markets {
			if m.Key != key {
				continue
			}
			if bk.Key == "pinnacle" {
				mm := m
				return mm, true
			}
			if fb == nil {
				mm := m
				fb = &mm
			}
		}
	}
	if fb != nil {
		return *fb, true
	}
	return apiMarket{}, false
}

// syncOdds pulls h2h + totals + spreads and stores reference lines.
func (a *App) syncOdds(ctx context.Context) error {
	evs, err := a.fetchEvents(ctx, "/odds", map[string]string{
		"regions": "eu", "markets": "h2h,totals,spreads", "oddsFormat": "decimal",
	})
	if err != nil {
		return err
	}
	pair, normToFifa := a.pairIndex()
	now := time.Now().Unix()
	n := 0
	for _, ev := range evs {
		mid, ok := resolveMatch(ev, pair, normToFifa)
		if !ok {
			continue
		}
		ho, dro, ao, ph, pd, pa := h2hConsensus(ev)
		ahLine, ahHome, ahAway := mainSpread(ev)
		ouLine, ouOver, ouUnder := mainTotals(ev)
		_, err := a.db.Exec(`INSERT INTO match_odds
			(match_id,p_home,p_draw,p_away,home_odds,draw_odds,away_odds,ah_line,ah_home,ah_away,ou_line,ou_over,ou_under,source,synced)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?, 'odds_api', ?)
			ON CONFLICT(match_id) DO UPDATE SET
			 p_home=excluded.p_home,p_draw=excluded.p_draw,p_away=excluded.p_away,
			 home_odds=excluded.home_odds,draw_odds=excluded.draw_odds,away_odds=excluded.away_odds,
			 ah_line=excluded.ah_line,ah_home=excluded.ah_home,ah_away=excluded.ah_away,
			 ou_line=excluded.ou_line,ou_over=excluded.ou_over,ou_under=excluded.ou_under,
			 source='odds_api',synced=excluded.synced`,
			mid, r4(ph), r4(pd), r4(pa), r2(ho), r2(dro), r2(ao),
			ahLine, r2(ahHome), r2(ahAway), ouLine, r2(ouOver), r2(ouUnder), now)
		if err == nil {
			n++
		}
	}
	log.Printf("[odds] synced %d matches", n)
	return nil
}

func h2hConsensus(ev apiEvent) (ho, dro, ao, ph, pd, pa float64) {
	var sh, sd, sa float64
	var c int
	for _, bk := range ev.Bookmakers {
		for _, m := range bk.Markets {
			if m.Key != "h2h" {
				continue
			}
			var h, d, aw float64
			for _, o := range m.Outcomes {
				switch canon(o.Name) {
				case canon(ev.HomeTeam):
					h = o.Price
				case canon(ev.AwayTeam):
					aw = o.Price
				default:
					d = o.Price
				}
			}
			if h > 0 && d > 0 && aw > 0 {
				sh += h
				sd += d
				sa += aw
				c++
			}
		}
	}
	if c == 0 {
		return
	}
	ho, dro, ao = sh/float64(c), sd/float64(c), sa/float64(c)
	ih, id, ia := 1/ho, 1/dro, 1/ao
	tot := ih + id + ia
	ph, pd, pa = ih/tot, id/tot, ia/tot
	return
}

func mainSpread(ev apiEvent) (line, home, away float64) {
	m, ok := pickMarket(ev, "spreads")
	if !ok {
		return
	}
	for _, o := range m.Outcomes {
		if canon(o.Name) == canon(ev.HomeTeam) {
			line, home = o.Point, o.Price
		} else if canon(o.Name) == canon(ev.AwayTeam) {
			away = o.Price
		}
	}
	return
}

func mainTotals(ev apiEvent) (line, over, under float64) {
	m, ok := pickMarket(ev, "totals")
	if !ok {
		return
	}
	for _, o := range m.Outcomes {
		if strings.EqualFold(o.Name, "Over") {
			over, line = o.Price, o.Point
		} else if strings.EqualFold(o.Name, "Under") {
			under = o.Price
		}
	}
	return
}

// syncScores pulls recent scores. Completed matches are finalized (and settle);
// in-play matches are marked live with their running score, and a goal voids
// every still-open (un-taken) bet on that match so stale offers aren't unfair.
func (a *App) syncScores(ctx context.Context) error {
	evs, err := a.fetchEvents(ctx, "/scores", map[string]string{"daysFrom": "3"})
	if err != nil {
		return err
	}
	pair, normToFifa := a.pairIndex()
	finished, voided := 0, 0
	for _, ev := range evs {
		if len(ev.Scores) < 2 {
			continue // not started yet (scores are null)
		}
		mid, ok := resolveMatch(ev, pair, normToFifa)
		if !ok {
			continue
		}
		var hs, as int
		for _, s := range ev.Scores {
			v, _ := strconv.Atoi(strings.TrimSpace(s.Score))
			if canon(s.Name) == canon(ev.HomeTeam) {
				hs = v
			} else if canon(s.Name) == canon(ev.AwayTeam) {
				as = v
			}
		}
		var status string
		var oldH, oldA int
		if a.db.QueryRow(`SELECT status,ft_home,ft_away FROM matches WHERE id=?`, mid).Scan(&status, &oldH, &oldA) != nil {
			continue
		}
		if status == "finished" {
			continue
		}
		if ev.Completed {
			a.db.Exec(`UPDATE matches SET ft_home=?,ft_away=?,status='finished',settled=0 WHERE id=?`, hs, as, mid)
			finished++
			continue
		}
		// in-play
		changed := hs != oldH || as != oldA
		if status != "live" || changed {
			a.db.Exec(`UPDATE matches SET status='live',ft_home=?,ft_away=? WHERE id=?`, hs, as, mid)
		}
		if changed {
			voided += a.refundOpenBets(mid, "live_voided", "có bàn thắng — huỷ kèo chưa khớp")
		}
	}
	if finished > 0 || voided > 0 {
		log.Printf("[scores] %d finished, %d open bets voided on goals", finished, voided)
	}
	return nil
}

func r2(f float64) float64 { v, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", f), 64); return v }
func r4(f float64) float64 { v, _ := strconv.ParseFloat(fmt.Sprintf("%.4f", f), 64); return v }
