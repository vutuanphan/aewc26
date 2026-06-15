package app

import (
	"database/sql"
	"errors"
	"log"
	"time"
)

var errInsufficient = errors.New("không đủ điểm")

// adjustBalance changes a wallet by delta and appends a ledger row. Runs in tx.
func adjustBalance(tx *sql.Tx, userID, delta int64, kind string, betID int64, memo string) error {
	var bal int64
	if err := tx.QueryRow(`SELECT balance FROM users WHERE id=?`, userID).Scan(&bal); err != nil {
		return err
	}
	nb := bal + delta
	if nb < 0 {
		return errInsufficient
	}
	if _, err := tx.Exec(`UPDATE users SET balance=? WHERE id=?`, nb, userID); err != nil {
		return err
	}
	_, err := tx.Exec(`INSERT INTO txns(user_id,amount,kind,bet_id,balance_after,memo,created) VALUES(?,?,?,?,?,?,?)`,
		userID, delta, kind, betID, nb, memo, time.Now().Unix())
	return err
}

func matchOpenForBets(m *Match) error {
	if m == nil {
		return errors.New("không tìm thấy trận")
	}
	if m.Status != "scheduled" {
		return errors.New("trận đã bắt đầu hoặc kết thúc")
	}
	if m.KickoffUnix > 0 && time.Now().Unix() > m.KickoffUnix {
		return errors.New("đã qua giờ bóng lăn")
	}
	return nil
}

// createBet validates and stores a new open kèo, locking the creator's stake.
func (a *App) createBet(creatorID, matchID int64, betType, pick string, line float64, stake int64, note string) error {
	if !validPick(betType, pick) {
		return errors.New("loại kèo / cửa không hợp lệ")
	}
	if stake < 1 {
		return errors.New("số điểm cược phải ≥ 1")
	}
	if betType == "ah" || betType == "ou" {
		if !isQuarterStep(line) {
			return errors.New("mức kèo phải là bội số 0.25")
		}
		if betType == "ou" && line <= 0 {
			return errors.New("mức tài/xỉu phải > 0")
		}
	} else {
		line = 0
	}
	m := a.getMatch(matchID)
	if err := matchOpenForBets(m); err != nil {
		return err
	}
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO bets(match_id,bet_type,creator_id,stake,pick,line,status,note,created)
		VALUES(?,?,?,?,?,?, 'open', ?, ?)`, matchID, betType, creatorID, stake, pick, line, note, time.Now().Unix())
	if err != nil {
		return err
	}
	betID, _ := res.LastInsertId()
	if err := adjustBalance(tx, creatorID, -stake, "stake_lock", betID, "đặt kèo"); err != nil {
		return err
	}
	return tx.Commit()
}

// takeBet matches the opposite side of an open kèo, locking the taker's stake.
// All reads happen before Begin: with a single DB connection, querying via
// a.db while a transaction is open would deadlock.
func (a *App) takeBet(uid, betID int64) error {
	var matchID, creatorID, stake int64
	var status string
	if err := a.db.QueryRow(`SELECT match_id,creator_id,stake,status FROM bets WHERE id=?`, betID).
		Scan(&matchID, &creatorID, &stake, &status); err != nil {
		return errors.New("không tìm thấy kèo")
	}
	if status != "open" {
		return errors.New("kèo đã được bắt hoặc đã đóng")
	}
	if creatorID == uid {
		return errors.New("không thể tự bắt kèo của mình")
	}
	if err := matchOpenForBets(a.getMatch(matchID)); err != nil {
		return err
	}

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Re-check status inside the tx to avoid a double-take race.
	var cur string
	if err := tx.QueryRow(`SELECT status FROM bets WHERE id=?`, betID).Scan(&cur); err != nil || cur != "open" {
		return errors.New("kèo đã được bắt hoặc đã đóng")
	}
	if err := adjustBalance(tx, uid, -stake, "stake_lock", betID, "bắt kèo"); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE bets SET taker_id=?,status='matched',matched=? WHERE id=?`,
		uid, time.Now().Unix(), betID); err != nil {
		return err
	}
	return tx.Commit()
}

// cancelBet lets the creator withdraw an un-taken kèo and refunds the stake.
func (a *App) cancelBet(uid, betID int64) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var creatorID, stake int64
	var status string
	if err := tx.QueryRow(`SELECT creator_id,stake,status FROM bets WHERE id=?`, betID).
		Scan(&creatorID, &stake, &status); err != nil {
		return errors.New("không tìm thấy kèo")
	}
	if creatorID != uid {
		return errors.New("chỉ người tạo mới huỷ được")
	}
	if status != "open" {
		return errors.New("kèo đã được bắt, không huỷ được")
	}
	if err := adjustBalance(tx, uid, stake, "refund", betID, "huỷ kèo"); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE bets SET status='cancelled',outcome='cancelled',settled_at=? WHERE id=?`,
		time.Now().Unix(), betID); err != nil {
		return err
	}
	return tx.Commit()
}

// settleDueMatches settles bets on finished/cancelled matches not yet processed.
func (a *App) settleDueMatches() {
	rows, err := a.db.Query(`SELECT id FROM matches WHERE settled=0 AND status IN ('finished','cancelled','postponed')`)
	if err != nil {
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		a.settleMatch(id)
	}
}

// settleMatch settles every matched bet on a match and refunds open ones,
// then marks the match processed. Idempotent.
func (a *App) settleMatch(matchID int64) {
	m := a.getMatch(matchID)
	if m == nil {
		return
	}
	finished := m.Status == "finished"
	voided := m.Status == "cancelled" || m.Status == "postponed"
	if !finished && !voided {
		return
	}

	rows, err := a.db.Query(`SELECT id,bet_type,creator_id,taker_id,stake,pick,line,status FROM bets
		WHERE match_id=? AND (status='open' OR status='matched')`, matchID)
	if err != nil {
		return
	}
	type pend struct {
		id, creator, taker, stake int64
		betType, pick, status     string
		line                      float64
	}
	var ps []pend
	for rows.Next() {
		var p pend
		if rows.Scan(&p.id, &p.betType, &p.creator, &p.taker, &p.stake, &p.pick, &p.line, &p.status) == nil {
			ps = append(ps, p)
		}
	}
	rows.Close()

	now := time.Now().Unix()
	for _, p := range ps {
		tx, err := a.db.Begin()
		if err != nil {
			continue
		}
		ok := func() bool {
			if p.status == "open" {
				if err := adjustBalance(tx, p.creator, p.stake, "refund", p.id, "kèo không có người bắt"); err != nil {
					return false
				}
				_, err := tx.Exec(`UPDATE bets SET status='cancelled',outcome='no_taker',settled_at=? WHERE id=?`, now, p.id)
				return err == nil
			}
			if voided {
				if adjustBalance(tx, p.creator, p.stake, "refund", p.id, "trận huỷ/hoãn") != nil {
					return false
				}
				if adjustBalance(tx, p.taker, p.stake, "refund", p.id, "trận huỷ/hoãn") != nil {
					return false
				}
				_, err := tx.Exec(`UPDATE bets SET status='void',outcome='void',creator_return=?,taker_return=?,settled_at=? WHERE id=?`,
					p.stake, p.stake, now, p.id)
				return err == nil
			}
			f, ferr := creatorFraction(p.betType, p.pick, p.line, m.FtHome, m.FtAway)
			if ferr != nil {
				return false
			}
			cr, tr, outcome := payouts(f, p.stake)
			if cr > 0 {
				if adjustBalance(tx, p.creator, cr, "payout", p.id, "chung chi kèo") != nil {
					return false
				}
			}
			if tr > 0 {
				if adjustBalance(tx, p.taker, tr, "payout", p.id, "chung chi kèo") != nil {
					return false
				}
			}
			_, err := tx.Exec(`UPDATE bets SET status='settled',outcome=?,creator_return=?,taker_return=?,settled_at=? WHERE id=?`,
				outcome, cr, tr, now, p.id)
			return err == nil
		}()
		if ok {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}

	if _, err := a.db.Exec(`UPDATE matches SET settled=1 WHERE id=?`, matchID); err != nil {
		log.Printf("[settle] mark match %d: %v", matchID, err)
	}
}

// grant adds/sets points for one user or all players (admin action).
func (a *App) grant(targetID int64, all bool, amount int64, set bool, memo string) (int, error) {
	if memo == "" {
		memo = "admin cấp điểm"
	}
	var targets []int64
	if all {
		for _, u := range a.players() {
			targets = append(targets, u.ID)
		}
	} else {
		targets = []int64{targetID}
	}
	tx, err := a.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	n := 0
	for _, id := range targets {
		delta := amount
		if set {
			var bal int64
			if err := tx.QueryRow(`SELECT balance FROM users WHERE id=?`, id).Scan(&bal); err != nil {
				return 0, err
			}
			delta = amount - bal
		}
		if delta == 0 {
			continue
		}
		if err := adjustBalance(tx, id, delta, "grant", 0, memo); err != nil {
			return 0, err
		}
		n++
	}
	return n, tx.Commit()
}

// setResult manually records a final score (admin), triggering settlement.
func (a *App) setResult(matchID int64, ftH, ftA int) error {
	_, err := a.db.Exec(`UPDATE matches SET ft_home=?,ft_away=?,status='finished',settled=0 WHERE id=?`, ftH, ftA, matchID)
	if err != nil {
		return err
	}
	a.settleMatch(matchID)
	return nil
}
