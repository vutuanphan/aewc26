package app

import (
	"context"
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// App holds shared dependencies.
type App struct {
	db            *sql.DB
	tmpl          *template.Template
	oddsKey       string
	resultsSource string
	brand         string
	startBalance  int64
	loc           *time.Location
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Run boots the app: open DB, seed, start sync jobs, serve HTTP.
func Run() error {
	db, err := openDB(env("AEWC_DB", "/data/aewc.db"))
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation(env("AEWC_TZ", "Asia/Ho_Chi_Minh"))
	if err != nil {
		loc = time.FixedZone("ICT", 7*3600)
	}

	startBalance := int64(100000)
	if v, err := strconv.ParseInt(os.Getenv("AEWC_START_BALANCE"), 10, 64); err == nil && v > 0 {
		startBalance = v
	}

	a := &App{
		db:            db,
		oddsKey:       os.Getenv("ODDS_API_KEY"),
		resultsSource: env("AEWC_RESULTS_SOURCE", "espn"),
		brand:         env("AEWC_BRAND", "AEWC26"),
		startBalance:  startBalance,
		loc:           loc,
	}
	if err := a.seed(); err != nil {
		return err
	}
	if err := a.parseTemplates(); err != nil {
		return err
	}

	a.startJobs()

	addr := ":" + env("PORT", "8090")
	log.Printf("[aewc] listening on %s", addr)
	return http.ListenAndServe(addr, a.router())
}

// startJobs settles locally every tick, polls live scores only while a match is
// on, and refreshes pre-match odds infrequently to conserve API quota.
func (a *App) startJobs() {
	tick := 120 * time.Second
	if v, err := strconv.Atoi(os.Getenv("AEWC_LIVE_POLL_SECONDS")); err == nil && v >= 20 {
		tick = time.Duration(v) * time.Second
	}
	oddsEvery := int((6 * time.Hour) / tick) // refresh odds ~every 6h
	if oddsEvery < 1 {
		oddsEvery = 1
	}
	ctx := context.Background()
	// Results/live source: ESPN (free, no key) by default, or the Odds API.
	syncResults := func() {
		if a.resultsSource == "oddsapi" {
			if a.oddsKey != "" {
				if err := a.syncScores(ctx); err != nil {
					log.Printf("[scores] %v", err)
				}
			}
			return
		}
		if err := a.syncESPN(ctx); err != nil {
			log.Printf("[espn] %v", err)
		}
	}
	go func() {
		time.Sleep(3 * time.Second)
		syncResults()
		if a.oddsKey != "" {
			a.syncOdds(ctx) // bookmaker reference odds (independent of results)
		}
		a.settleDueMatches()
		n := 0
		for range time.NewTicker(tick).C {
			n++
			a.settleDueMatches()
			if a.hasLiveWindow() {
				syncResults()
			}
			if a.oddsKey != "" && n%oddsEvery == 0 {
				if err := a.syncOdds(ctx); err != nil {
					log.Printf("[odds] %v", err)
				}
			}
		}
	}()
}

// hasLiveWindow reports whether any match is around kickoff (so live polling is
// worth the API call): kickoff within the last ~3h and not finished.
func (a *App) hasLiveWindow() bool {
	now := time.Now().Unix()
	var c int
	a.db.QueryRow(`SELECT COUNT(*) FROM matches WHERE status!='finished' AND kickoff<=? AND kickoff>=?`,
		now+120, now-3*3600).Scan(&c)
	return c > 0
}
