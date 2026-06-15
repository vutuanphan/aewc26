package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type ctxKey int

const userKey ctxKey = 1

const sessionCookie = "aewc_sess"

func hashPassword(p string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, p string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)) == nil
}

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// createSession issues a session token for a user.
func (a *App) createSession(userID int64) (string, error) {
	tok := randToken()
	exp := time.Now().Add(30 * 24 * time.Hour).Unix()
	_, err := a.db.Exec(`INSERT INTO sessions(token,user_id,expires) VALUES(?,?,?)`, tok, userID, exp)
	return tok, err
}

func (a *App) deleteSession(tok string) {
	a.db.Exec(`DELETE FROM sessions WHERE token=?`, tok)
}

// userFromRequest resolves the logged-in user from the session cookie.
func (a *App) userFromRequest(r *http.Request) *User {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	var uid, exp int64
	if err := a.db.QueryRow(`SELECT user_id,expires FROM sessions WHERE token=?`, c.Value).Scan(&uid, &exp); err != nil {
		return nil
	}
	if exp < time.Now().Unix() {
		a.deleteSession(c.Value)
		return nil
	}
	return a.getUser(uid)
}

// currentUser pulls the user that auth middleware stored on the context.
func currentUser(r *http.Request) *User {
	u, _ := r.Context().Value(userKey).(*User)
	return u
}

// requireAuth wraps a handler, redirecting anonymous visitors to /login.
func (a *App) requireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := a.userFromRequest(r)
		if u == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), userKey, u)
		h(w, r.WithContext(ctx))
	}
}

// requireAdmin is like requireAuth but also demands the admin flag.
func (a *App) requireAdmin(h http.HandlerFunc) http.HandlerFunc {
	return a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if u := currentUser(r); u == nil || !u.IsAdmin {
			http.Error(w, "Chỉ admin", http.StatusForbidden)
			return
		}
		h(w, r)
	})
}

// csrfToken returns the per-session CSRF token (the session cookie value).
func csrfToken(r *http.Request) string {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

// checkCSRF validates the double-submit token on a POST.
func checkCSRF(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	return c.Value != "" && r.FormValue("csrf") == c.Value
}
