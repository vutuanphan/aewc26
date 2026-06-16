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

// applyMatchScore records a score update for a match: finalize when completed
// (goal-based settlement runs separately), otherwise mark it live. A change in
// the live score (a goal) voids every still-open bet on the match. Returns
// (justFinished, openBetsVoided).
func (a *App) applyMatchScore(mid int64, completed bool, hs, as int) (bool, int) {
	var status string
	var oldH, oldA int
	if a.db.QueryRow(`SELECT status,ft_home,ft_away FROM matches WHERE id=?`, mid).Scan(&status, &oldH, &oldA) != nil {
		return false, 0
	}
	if status == "finished" {
		return false, 0
	}
	if completed {
		a.db.Exec(`UPDATE matches SET ft_home=?,ft_away=?,status='finished',settled=0 WHERE id=?`, hs, as, mid)
		return true, 0
	}
	changed := hs != oldH || as != oldA
	if status != "live" || changed {
		a.db.Exec(`UPDATE matches SET status='live',ft_home=?,ft_away=? WHERE id=?`, hs, as, mid)
	}
	if changed {
		return false, a.refundOpenBets(mid, "live_voided", "có bàn thắng — huỷ kèo chưa khớp")
	}
	return false, 0
}

// ---- ESPN scoreboard: free, no key, no quota (the default results source) ----

const espnBase = "https://site.api.espn.com/apis/site/v2/sports/soccer/fifa.world/scoreboard"

type espnResp struct {
	Events []struct {
		Status struct {
			Type struct {
				State     string `json:"state"` // pre | in | post
				Completed bool   `json:"completed"`
			} `json:"type"`
		} `json:"status"`
		Competitions []struct {
			Competitors []struct {
				HomeAway string `json:"homeAway"`
				Score    string `json:"score"`
				Team     struct {
					DisplayName  string `json:"displayName"`
					Abbreviation string `json:"abbreviation"`
				} `json:"team"`
			} `json:"competitors"`
		} `json:"competitions"`
	} `json:"events"`
}

func (a *App) fetchESPN(ctx context.Context, date string) (*espnResp, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, espnBase+"?dates="+date, nil)
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("espn status %d", resp.StatusCode)
	}
	var r espnResp
	return &r, json.NewDecoder(resp.Body).Decode(&r)
}

// syncESPN pulls live + finished scores from ESPN for the days around now.
func (a *App) syncESPN(ctx context.Context) error {
	pair, normToFifa := a.pairIndex()
	fifaSet := map[string]bool{}
	rows, _ := a.db.Query(`SELECT fifa FROM teams`)
	for rows.Next() {
		var f string
		rows.Scan(&f)
		fifaSet[f] = true
	}
	rows.Close()

	now := time.Now().UTC()
	finished, voided := 0, 0
	seen := map[int64]bool{}
	for _, off := range []int{-1, 0, 1} {
		date := now.AddDate(0, 0, off).Format("20060102")
		r, err := a.fetchESPN(ctx, date)
		if err != nil {
			log.Printf("[espn] %s: %v", date, err)
			continue
		}
		for _, ev := range r.Events {
			if ev.Status.Type.State == "pre" || len(ev.Competitions) == 0 {
				continue
			}
			var hName, aName, hAbbr, aAbbr string
			var hs, as int
			for _, c := range ev.Competitions[0].Competitors {
				sc, _ := strconv.Atoi(strings.TrimSpace(c.Score))
				if c.HomeAway == "home" {
					hName, hAbbr, hs = c.Team.DisplayName, c.Team.Abbreviation, sc
				} else if c.HomeAway == "away" {
					aName, aAbbr, as = c.Team.DisplayName, c.Team.Abbreviation, sc
				}
			}
			mid, ok := resolveESPN(hName, aName, hAbbr, aAbbr, pair, normToFifa, fifaSet)
			if !ok {
				log.Printf("[espn] unmatched %q vs %q", hName, aName)
				continue
			}
			if seen[mid] {
				continue
			}
			seen[mid] = true
			fin, v := a.applyMatchScore(mid, ev.Status.Type.Completed, hs, as)
			if fin {
				finished++
			}
			voided += v
		}
	}
	if finished > 0 || voided > 0 {
		log.Printf("[espn] %d finished, %d open bets voided on goals", finished, voided)
	}
	return nil
}

// resolveESPN maps an ESPN event to a match id by team-name pair, falling back
// to ESPN's 3-letter abbreviation (often the FIFA code).
func resolveESPN(hName, aName, hAbbr, aAbbr string, pair map[string]int64, normToFifa map[string]string, fifaSet map[string]bool) (int64, bool) {
	hf := normToFifa[canon(hName)]
	if hf == "" && fifaSet[strings.ToUpper(hAbbr)] {
		hf = strings.ToUpper(hAbbr)
	}
	af := normToFifa[canon(aName)]
	if af == "" && fifaSet[strings.ToUpper(aAbbr)] {
		af = strings.ToUpper(aAbbr)
	}
	if hf == "" || af == "" {
		return 0, false
	}
	id, ok := pair[hf+"|"+af]
	return id, ok
}
