package app

import (
	"errors"
	"strings"
	"time"
)

// createPlayer adds a non-admin user with an optional starting balance.
func (a *App) createPlayer(username, name, password string, balance int64) error {
	username = strings.ToLower(strings.TrimSpace(username))
	name = strings.TrimSpace(name)
	if username == "" || name == "" {
		return errors.New("thông tin không hợp lệ")
	}
	if len(password) < 4 {
		return errors.New("mật khẩu tối thiểu 4 ký tự")
	}
	if balance < 0 {
		balance = 0
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	res, err := a.db.Exec(`INSERT INTO users(username,name,pass_hash,balance,is_admin,created) VALUES(?,?,?,?,0,?)`,
		username, name, hash, balance, now)
	if err != nil {
		return errors.New("tên đăng nhập đã tồn tại")
	}
	if balance > 0 {
		id, _ := res.LastInsertId()
		a.db.Exec(`INSERT INTO txns(user_id,amount,kind,balance_after,memo,created) VALUES(?,?,?,?,?,?)`,
			id, balance, "grant", balance, "Starting balance", now)
	}
	return nil
}

// resetPassword sets a new password for a non-admin player.
func (a *App) resetPassword(userID int64, password string) error {
	if len(password) < 4 {
		return errors.New("mật khẩu tối thiểu 4 ký tự")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = a.db.Exec(`UPDATE users SET pass_hash=? WHERE id=? AND is_admin=0`, hash, userID)
	return err
}
