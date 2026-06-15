package app

import (
	"database/sql"
	"time"
)

func (a *App) getUser(id int64) *User {
	u := &User{}
	var admin int
	err := a.db.QueryRow(`SELECT id,username,name,balance,is_admin FROM users WHERE id=?`, id).
		Scan(&u.ID, &u.Username, &u.Name, &u.Balance, &admin)
	if err != nil {
		return nil
	}
	u.IsAdmin = admin == 1
	return u
}

func (a *App) authUser(username string) (id int64, hash string, ok bool) {
	err := a.db.QueryRow(`SELECT id,pass_hash FROM users WHERE username=?`, username).Scan(&id, &hash)
	return id, hash, err == nil
}

const matchCols = `m.id,m.ext_id,m.stage,m.group_letter,m.round_label,m.num,m.kickoff,
 m.home_fifa,m.away_fifa,m.home_label,m.away_label,m.status,m.ft_home,m.ft_away,m.settled,
 COALESCE(th.name,''),COALESCE(ta.name,''),COALESCE(th.iso2,''),COALESCE(ta.iso2,'')`

const matchFrom = ` FROM matches m
 LEFT JOIN teams th ON th.fifa=m.home_fifa
 LEFT JOIN teams ta ON ta.fifa=m.away_fifa`

type scanner interface{ Scan(...any) error }

func scanMatch(s scanner) (Match, error) {
	var m Match
	var settled int
	err := s.Scan(&m.ID, &m.ExtID, &m.Stage, &m.GroupLetter, &m.RoundLabel, &m.Num, &m.KickoffUnix,
		&m.HomeFifa, &m.AwayFifa, &m.HomeLabel, &m.AwayLabel, &m.Status, &m.FtHome, &m.FtAway, &settled,
		&m.HomeName, &m.AwayName, &m.HomeISO, &m.AwayISO)
	m.Settled = settled == 1
	return m, err
}

func (a *App) getMatch(id int64) *Match {
	m, err := scanMatch(a.db.QueryRow(`SELECT `+matchCols+matchFrom+` WHERE m.id=?`, id))
	if err != nil {
		return nil
	}
	return &m
}

// upcomingMatches returns scheduled, both-teams-known matches not yet kicked off.
func (a *App) upcomingMatches() []Match {
	now := time.Now().Unix()
	rows, err := a.db.Query(`SELECT `+matchCols+matchFrom+
		` WHERE m.status='scheduled' AND m.home_fifa<>'' AND m.away_fifa<>'' AND m.kickoff>?
		  ORDER BY m.kickoff`, now)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Match
	for rows.Next() {
		if m, err := scanMatch(rows); err == nil {
			out = append(out, m)
		}
	}
	return out
}

func (a *App) getOdds(matchID int64) *Odds {
	o := &Odds{MatchID: matchID}
	err := a.db.QueryRow(`SELECT p_home,p_draw,p_away,home_odds,draw_odds,away_odds,
		ah_line,ah_home,ah_away,ou_line,ou_over,ou_under,source FROM match_odds WHERE match_id=?`, matchID).
		Scan(&o.PHome, &o.PDraw, &o.PAway, &o.HomeOdds, &o.DrawOdds, &o.AwayOdds,
			&o.AhLine, &o.AhHome, &o.AhAway, &o.OuLine, &o.OuOver, &o.OuUnder, &o.Source)
	if err != nil {
		return nil
	}
	return o
}

func (a *App) scanBets(rows *sql.Rows) []Bet {
	// Read all rows first and close the result set BEFORE running any further
	// queries — with a single DB connection, enriching inside the loop would
	// deadlock.
	var out []Bet
	for rows.Next() {
		var b Bet
		if err := rows.Scan(&b.ID, &b.MatchID, &b.BetType, &b.CreatorID, &b.TakerID, &b.Stake,
			&b.Pick, &b.Line, &b.Status, &b.Outcome, &b.CreatorReturn, &b.TakerReturn, &b.Note, &b.CreatedUnix); err != nil {
			continue
		}
		out = append(out, b)
	}
	rows.Close()
	for i := range out {
		if m := a.getMatch(out[i].MatchID); m != nil {
			out[i].Match = *m
		}
		if u := a.getUser(out[i].CreatorID); u != nil {
			out[i].CreatorName = u.Name
		}
		if out[i].TakerID != 0 {
			if u := a.getUser(out[i].TakerID); u != nil {
				out[i].TakerName = u.Name
			}
		}
	}
	return out
}

const betCols = `id,match_id,bet_type,creator_id,taker_id,stake,pick,line,status,outcome,creator_return,taker_return,note,created`

func (a *App) getBet(id int64) *Bet {
	rows, err := a.db.Query(`SELECT `+betCols+` FROM bets WHERE id=?`, id)
	if err != nil {
		return nil
	}
	bs := a.scanBets(rows)
	if len(bs) == 0 {
		return nil
	}
	return &bs[0]
}

func (a *App) listOpenBets() []Bet {
	rows, err := a.db.Query(`SELECT ` + betCols + ` FROM bets WHERE status='open' ORDER BY created DESC`)
	if err != nil {
		return nil
	}
	return a.scanBets(rows)
}

func (a *App) listMyBets(uid int64) []Bet {
	rows, err := a.db.Query(`SELECT `+betCols+` FROM bets WHERE creator_id=? OR taker_id=? ORDER BY created DESC`, uid, uid)
	if err != nil {
		return nil
	}
	return a.scanBets(rows)
}

func (a *App) leaderboard() []User {
	rows, err := a.db.Query(`SELECT id,username,name,balance,is_admin FROM users WHERE is_admin=0 ORDER BY balance DESC, name`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		var admin int
		if err := rows.Scan(&u.ID, &u.Username, &u.Name, &u.Balance, &admin); err == nil {
			out = append(out, u)
		}
	}
	return out
}

func (a *App) players() []User { return a.leaderboard() }

func (a *App) listTxns(uid int64, limit int) []Txn {
	rows, err := a.db.Query(`SELECT id,user_id,amount,kind,bet_id,balance_after,memo,created
		FROM txns WHERE user_id=? ORDER BY id DESC LIMIT ?`, uid, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Txn
	for rows.Next() {
		var t Txn
		if err := rows.Scan(&t.ID, &t.UserID, &t.Amount, &t.Kind, &t.BetID, &t.BalanceAfter, &t.Memo, &t.CreatedUnix); err == nil {
			out = append(out, t)
		}
	}
	return out
}

func (a *App) listChat(limit int) []ChatMsg {
	rows, err := a.db.Query(`SELECT c.id,c.user_id,u.name,c.body,c.created
		FROM chat c JOIN users u ON u.id=c.user_id ORDER BY c.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ChatMsg
	for rows.Next() {
		var m ChatMsg
		if err := rows.Scan(&m.ID, &m.UserID, &m.UserName, &m.Body, &m.CreatedUnix); err == nil {
			out = append(out, m)
		}
	}
	// reverse to chronological
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
