package app

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  username  TEXT UNIQUE NOT NULL,
  name      TEXT NOT NULL,
  pass_hash TEXT NOT NULL,
  balance   INTEGER NOT NULL DEFAULT 0,
  is_admin  INTEGER NOT NULL DEFAULT 0,
  created   INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
  token   TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL,
  expires INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS teams (
  fifa          TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  iso2          TEXT,
  confederation TEXT
);
CREATE TABLE IF NOT EXISTS matches (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  ext_id      TEXT UNIQUE,
  stage       TEXT,
  group_letter TEXT,
  round_label TEXT,
  num         INTEGER,
  kickoff     INTEGER,
  home_fifa   TEXT,
  away_fifa   TEXT,
  home_label  TEXT,
  away_label  TEXT,
  status      TEXT,
  ft_home     INTEGER,
  ft_away     INTEGER,
  settled     INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS match_odds (
  match_id  INTEGER PRIMARY KEY,
  p_home REAL, p_draw REAL, p_away REAL,
  home_odds REAL, draw_odds REAL, away_odds REAL,
  ah_line REAL, ah_home REAL, ah_away REAL,
  ou_line REAL, ou_over REAL, ou_under REAL,
  source TEXT, synced INTEGER
);
CREATE TABLE IF NOT EXISTS bets (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  match_id    INTEGER NOT NULL,
  bet_type    TEXT NOT NULL,
  creator_id  INTEGER NOT NULL,
  taker_id    INTEGER NOT NULL DEFAULT 0,
  stake       INTEGER NOT NULL,
  pick        TEXT NOT NULL,
  line        REAL NOT NULL DEFAULT 0,
  status      TEXT NOT NULL,
  outcome     TEXT NOT NULL DEFAULT '',
  creator_return INTEGER NOT NULL DEFAULT 0,
  taker_return   INTEGER NOT NULL DEFAULT 0,
  note        TEXT NOT NULL DEFAULT '',
  created     INTEGER NOT NULL,
  matched     INTEGER NOT NULL DEFAULT 0,
  settled_at  INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS txns (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id       INTEGER NOT NULL,
  amount        INTEGER NOT NULL,
  kind          TEXT NOT NULL,
  bet_id        INTEGER NOT NULL DEFAULT 0,
  balance_after INTEGER NOT NULL,
  memo          TEXT NOT NULL DEFAULT '',
  created       INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS chat (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  body    TEXT NOT NULL,
  created INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bets_status ON bets(status);
CREATE INDEX IF NOT EXISTS idx_bets_match ON bets(match_id);
CREATE INDEX IF NOT EXISTS idx_txns_user ON txns(user_id);
`

// openDB opens the SQLite database and ensures the schema exists.
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: serialize writes, simplest correct choice
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return db, nil
}
