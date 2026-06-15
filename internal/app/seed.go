package app

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"
)

//go:embed seeddata/teams.json
var teamsJSON []byte

//go:embed seeddata/matches.json
var matchesJSON []byte

type seedTeam struct {
	FifaCode      string `json:"fifaCode"`
	Name          string `json:"name"`
	ISO2          string `json:"iso2"`
	Confederation string `json:"confederation"`
}

type seedMatch struct {
	ExtID       string `json:"extId"`
	Stage       string `json:"stage"`
	GroupLetter string `json:"groupLetter"`
	RoundLabel  string `json:"roundLabel"`
	Num         int    `json:"num"`
	Kickoff     string `json:"kickoff"`
	HomeFifa    string `json:"homeFifa"`
	AwayFifa    string `json:"awayFifa"`
	HomeLabel   string `json:"homeLabel"`
	AwayLabel   string `json:"awayLabel"`
	Status      string `json:"status"`
	FtHome      int    `json:"ftHome"`
	FtAway      int    `json:"ftAway"`
}

func parseKickoff(s string) int64 {
	for _, layout := range []string{"2006-01-02 15:04:05.000Z", time.RFC3339, "2006-01-02 15:04:05Z"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Unix()
		}
	}
	return 0
}

// seed populates teams, matches and the initial admin on first boot.
func (a *App) seed() error {
	db := a.db
	// teams
	var nTeams int
	db.QueryRow(`SELECT COUNT(*) FROM teams`).Scan(&nTeams)
	if nTeams == 0 {
		var ts []seedTeam
		if err := json.Unmarshal(teamsJSON, &ts); err != nil {
			return err
		}
		for _, t := range ts {
			if _, err := db.Exec(`INSERT INTO teams(fifa,name,iso2,confederation) VALUES(?,?,?,?)`,
				t.FifaCode, t.Name, t.ISO2, t.Confederation); err != nil {
				return err
			}
		}
		log.Printf("[seed] %d teams", len(ts))
	}

	// matches
	var nMatches int
	db.QueryRow(`SELECT COUNT(*) FROM matches`).Scan(&nMatches)
	if nMatches == 0 {
		var ms []seedMatch
		if err := json.Unmarshal(matchesJSON, &ms); err != nil {
			return err
		}
		for _, m := range ms {
			status := m.Status
			if status == "" {
				status = "scheduled"
			}
			if _, err := db.Exec(`INSERT INTO matches
				(ext_id,stage,group_letter,round_label,num,kickoff,home_fifa,away_fifa,home_label,away_label,status,ft_home,ft_away,settled)
				VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,0)`,
				m.ExtID, m.Stage, m.GroupLetter, m.RoundLabel, m.Num, parseKickoff(m.Kickoff),
				m.HomeFifa, m.AwayFifa, m.HomeLabel, m.AwayLabel, status, m.FtHome, m.FtAway); err != nil {
				return err
			}
		}
		log.Printf("[seed] %d matches", len(ms))
	}

	// users — create an admin on first boot; players are added later in /admin.
	var nUsers int
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&nUsers)
	if nUsers == 0 {
		now := time.Now().Unix()
		adminPass := os.Getenv("AEWC_ADMIN_PASSWORD")
		if adminPass == "" {
			adminPass = randToken()[:12]
			log.Printf("[seed] ADMIN PASSWORD (save this): admin / %s", adminPass)
		}
		createUser(db, "admin", "Admin", adminPass, true, 0, now)

		// Optional demo roster for a quick try (AEWC_SEED_DEMO=1).
		if os.Getenv("AEWC_SEED_DEMO") == "1" {
			for i := 1; i <= 4; i++ {
				u := "demo" + strconv.Itoa(i)
				id := createUser(db, u, "Demo "+strconv.Itoa(i), "demo1234", false, a.startBalance, now)
				db.Exec(`INSERT INTO txns(user_id,amount,kind,balance_after,memo,created) VALUES(?,?,?,?,?,?)`,
					id, a.startBalance, "grant", a.startBalance, "Starting balance", now)
			}
			log.Printf("[seed] created admin + 4 demo players (password: demo1234)")
		} else {
			log.Printf("[seed] created admin only — add players in /admin")
		}
	}
	return nil
}

func createUser(db *sql.DB, username, name, password string, admin bool, balance, now int64) int64 {
	hash, _ := hashPassword(password)
	ad := 0
	if admin {
		ad = 1
	}
	res, err := db.Exec(`INSERT INTO users(username,name,pass_hash,balance,is_admin,created) VALUES(?,?,?,?,?,?)`,
		username, name, hash, balance, ad, now)
	if err != nil {
		log.Printf("[seed] createUser %s: %v", username, err)
		return 0
	}
	id, _ := res.LastInsertId()
	return id
}
