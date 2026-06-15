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
	db           *sql.DB
	tmpl         *template.Template
	oddsKey      string
	brand        string
	startBalance int64
	loc          *time.Location
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
		db:           db,
		oddsKey:      os.Getenv("ODDS_API_KEY"),
		brand:        env("AEWC_BRAND", "AEWC26"),
		startBalance: startBalance,
		loc:          loc,
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

// startJobs runs odds + results sync and settlement on a timer.
func (a *App) startJobs() {
	run := func() {
		if a.oddsKey != "" {
			if err := a.syncOdds(context.Background()); err != nil {
				log.Printf("[odds] %v", err)
			}
			if err := a.syncScores(context.Background()); err != nil {
				log.Printf("[scores] %v", err)
			}
		}
		a.settleDueMatches()
	}
	go func() {
		time.Sleep(3 * time.Second)
		run()
		t := time.NewTicker(15 * time.Minute)
		for range t.C {
			run()
		}
	}()
}
