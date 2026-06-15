package app

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func secureReq(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func redirectMsg(w http.ResponseWriter, r *http.Request, path, okMsg, errMsg string) {
	q := ""
	if okMsg != "" {
		q = "?ok=" + url.QueryEscape(okMsg)
	} else if errMsg != "" {
		q = "?err=" + url.QueryEscape(errMsg)
	}
	http.Redirect(w, r, path+q, http.StatusSeeOther)
}

// ---- language ----

func (a *App) handleLang(w http.ResponseWriter, r *http.Request) {
	to := r.URL.Query().Get("to")
	if to != "vi" {
		to = "en"
	}
	http.SetCookie(w, &http.Cookie{Name: "lang", Value: to, Path: "/", MaxAge: 365 * 24 * 3600, SameSite: http.SameSiteLaxMode})
	ref := r.Header.Get("Referer")
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

// ---- auth ----

func (a *App) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	if a.userFromRequest(r) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	lang := langFromRequest(r)
	a.render(w, "login", pageData{Title: a.brand, Brand: a.brand, Lang: lang, FlashErr: r.URL.Query().Get("err")})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	lang := langFromRequest(r)
	username := r.FormValue("username")
	password := r.FormValue("password")
	id, hash, ok := a.authUser(username)
	if !ok || !checkPassword(hash, password) {
		redirectMsg(w, r, "/login", "", tr(lang, "err.login"))
		return
	}
	tok, err := a.createSession(id)
	if err != nil {
		redirectMsg(w, r, "/login", "", tr(lang, "err.session"))
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: tok, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: secureReq(r), MaxAge: 30 * 24 * 3600,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		a.deleteSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ---- home / market ----

type homeData struct {
	Balance   int64
	OpenBets  []Bet
	Upcoming  []Match
	FeedJSON  template.JS
}

func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	up := a.upcomingMatches()

	type oddsJS struct {
		Home, Away                       string
		HomeOdds, DrawOdds, AwayOdds     float64
		PHome, PDraw, PAway              float64
		AhLine, AhHome, AhAway           float64
		OuLine, OuOver, OuUnder          float64
		Has                              bool
	}
	feed := map[string]oddsJS{}
	for _, m := range up {
		oj := oddsJS{Home: m.HomeDisplay(), Away: m.AwayDisplay()}
		if o := a.getOdds(m.ID); o != nil {
			oj.HomeOdds, oj.DrawOdds, oj.AwayOdds = o.HomeOdds, o.DrawOdds, o.AwayOdds
			oj.PHome, oj.PDraw, oj.PAway = o.PHome, o.PDraw, o.PAway
			oj.AhLine, oj.AhHome, oj.AhAway = o.AhLine, o.AhHome, o.AhAway
			oj.OuLine, oj.OuOver, oj.OuUnder = o.OuLine, o.OuOver, o.OuUnder
			oj.Has = true
		}
		feed[strconv.FormatInt(m.ID, 10)] = oj
	}
	b, _ := json.Marshal(feed)

	pd := a.page(r, "Anh Em WC 2026", "home")
	pd.Data = homeData{
		Balance:  u.Balance,
		OpenBets: a.listOpenBets(),
		Upcoming: up,
		FeedJSON: template.JS(b),
	}
	a.render(w, "home", pd)
}

func (a *App) handleCreateBet(w http.ResponseWriter, r *http.Request) {
	lang := langFromRequest(r)
	if !checkCSRF(r) {
		redirectMsg(w, r, "/", "", tr(lang, "err.session"))
		return
	}
	u := currentUser(r)
	matchID, _ := strconv.ParseInt(r.FormValue("match"), 10, 64)
	betType := r.FormValue("betType")
	pick := r.FormValue("pick")
	line, _ := strconv.ParseFloat(r.FormValue("line"), 64)
	stake, _ := strconv.ParseInt(r.FormValue("stake"), 10, 64)
	note := r.FormValue("note")
	if err := a.createBet(u.ID, matchID, betType, pick, line, stake, note); err != nil {
		redirectMsg(w, r, "/", "", trMsg(lang, err.Error()))
		return
	}
	redirectMsg(w, r, "/", tr(lang, "ok.created"), "")
}

func (a *App) handleTakeBet(w http.ResponseWriter, r *http.Request) {
	lang := langFromRequest(r)
	if !checkCSRF(r) {
		redirectMsg(w, r, "/", "", tr(lang, "err.session"))
		return
	}
	u := currentUser(r)
	betID, _ := strconv.ParseInt(r.FormValue("bet"), 10, 64)
	if err := a.takeBet(u.ID, betID); err != nil {
		redirectMsg(w, r, "/", "", trMsg(lang, err.Error()))
		return
	}
	redirectMsg(w, r, "/", tr(lang, "ok.taken"), "")
}

func (a *App) handleCancelBet(w http.ResponseWriter, r *http.Request) {
	lang := langFromRequest(r)
	if !checkCSRF(r) {
		redirectMsg(w, r, "/", "", tr(lang, "err.session"))
		return
	}
	u := currentUser(r)
	betID, _ := strconv.ParseInt(r.FormValue("bet"), 10, 64)
	if err := a.cancelBet(u.ID, betID); err != nil {
		redirectMsg(w, r, "/", "", trMsg(lang, err.Error()))
		return
	}
	redirectMsg(w, r, "/", tr(lang, "ok.cancelled"), "")
}

// ---- my bets ----

type betView struct {
	Bet
	IsCreator  bool
	MySide     string
	Net        int64
	StatusText string
}

func (a *App) toView(lang string, b Bet, uid int64) betView {
	v := betView{Bet: b, IsCreator: b.CreatorID == uid}
	if v.IsCreator {
		v.MySide = creatorSide(lang, b)
	} else {
		v.MySide = takerSide(lang, b)
	}
	switch b.Status {
	case "open":
		if v.IsCreator {
			v.StatusText = tr(lang, "s.waiting")
		} else {
			v.StatusText = tr(lang, "s.open")
		}
	case "matched":
		v.StatusText = tr(lang, "s.matched")
	default:
		v.StatusText = outcomeLabel(lang, b.Outcome)
		ret := b.TakerReturn
		if v.IsCreator {
			ret = b.CreatorReturn
		}
		v.Net = ret - b.Stake
	}
	return v
}

func (a *App) handleMyBets(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	lang := langFromRequest(r)
	var active, done []betView
	for _, b := range a.listMyBets(u.ID) {
		v := a.toView(lang, b, u.ID)
		if b.Status == "open" || b.Status == "matched" {
			active = append(active, v)
		} else {
			done = append(done, v)
		}
	}
	pd := a.page(r, "Kèo của tôi", "mybets")
	pd.Data = map[string]any{"Active": active, "Done": done}
	a.render(w, "mybets", pd)
}

func (a *App) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	pd := a.page(r, "Bảng xếp hạng", "leaderboard")
	pd.Data = a.leaderboard()
	a.render(w, "leaderboard", pd)
}

func (a *App) handleWallet(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	pd := a.page(r, "Ví của tôi", "wallet")
	pd.Data = map[string]any{"Balance": u.Balance, "Txns": a.listTxns(u.ID, 200)}
	a.render(w, "wallet", pd)
}

// ---- chat ----

func (a *App) handleChat(w http.ResponseWriter, r *http.Request) {
	pd := a.page(r, "Chat", "chat")
	pd.Data = map[string]any{"Msgs": a.listChat(100), "Me": currentUser(r).ID}
	a.render(w, "chat", pd)
}

func (a *App) handleChatPost(w http.ResponseWriter, r *http.Request) {
	if !checkCSRF(r) {
		http.Redirect(w, r, "/chat", http.StatusSeeOther)
		return
	}
	body := r.FormValue("body")
	if len(body) > 500 {
		body = body[:500]
	}
	if body != "" {
		a.db.Exec(`INSERT INTO chat(user_id,body,created) VALUES(?,?,?)`, currentUser(r).ID, body, time.Now().Unix())
	}
	http.Redirect(w, r, "/chat", http.StatusSeeOther)
}

func (a *App) handleChatFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	pd := pageData{Lang: langFromRequest(r), Data: map[string]any{"Msgs": a.listChat(100), "Me": currentUser(r).ID}}
	a.render(w, "chat_feed", pd)
}

// ---- admin ----

func (a *App) handleAdmin(w http.ResponseWriter, r *http.Request) {
	// matches that can take a manual result: kicked off / live / finished recently
	now := time.Now().Unix()
	rows, _ := a.db.Query(`SELECT `+matchCols+matchFrom+
		` WHERE m.home_fifa<>'' AND m.away_fifa<>'' AND m.kickoff<=?
		  ORDER BY m.kickoff DESC LIMIT 40`, now)
	var ms []Match
	for rows.Next() {
		if m, err := scanMatch(rows); err == nil {
			ms = append(ms, m)
		}
	}
	rows.Close()
	pd := a.page(r, "Admin", "admin")
	pd.Data = map[string]any{"Players": a.players(), "Matches": ms, "StartBalance": a.startBalance}
	a.render(w, "admin", pd)
}

func (a *App) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	lang := langFromRequest(r)
	if !checkCSRF(r) {
		redirectMsg(w, r, "/admin", "", tr(lang, "err.session"))
		return
	}
	bal, _ := strconv.ParseInt(r.FormValue("balance"), 10, 64)
	if err := a.createPlayer(r.FormValue("username"), r.FormValue("name"), r.FormValue("password"), bal); err != nil {
		redirectMsg(w, r, "/admin", "", trMsg(lang, err.Error()))
		return
	}
	redirectMsg(w, r, "/admin", tr(lang, "ok.usercreated"), "")
}

func (a *App) handleResetPw(w http.ResponseWriter, r *http.Request) {
	lang := langFromRequest(r)
	if !checkCSRF(r) {
		redirectMsg(w, r, "/admin", "", tr(lang, "err.session"))
		return
	}
	uid, _ := strconv.ParseInt(r.FormValue("user"), 10, 64)
	if err := a.resetPassword(uid, r.FormValue("password")); err != nil {
		redirectMsg(w, r, "/admin", "", trMsg(lang, err.Error()))
		return
	}
	redirectMsg(w, r, "/admin", tr(lang, "ok.pwreset"), "")
}

func (a *App) handleGrant(w http.ResponseWriter, r *http.Request) {
	lang := langFromRequest(r)
	if !checkCSRF(r) {
		redirectMsg(w, r, "/admin", "", tr(lang, "err.session"))
		return
	}
	amount, _ := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	set := r.FormValue("mode") == "set"
	memo := r.FormValue("memo")
	all := r.FormValue("target") == "all"
	var target int64
	if !all {
		target, _ = strconv.ParseInt(r.FormValue("target"), 10, 64)
	}
	n, err := a.grant(target, all, amount, set, memo)
	if err != nil {
		redirectMsg(w, r, "/admin", "", trMsg(lang, err.Error()))
		return
	}
	redirectMsg(w, r, "/admin", fmt.Sprintf(tr(lang, "admin.updated"), n), "")
}

func (a *App) handleResult(w http.ResponseWriter, r *http.Request) {
	lang := langFromRequest(r)
	if !checkCSRF(r) {
		redirectMsg(w, r, "/admin", "", tr(lang, "err.session"))
		return
	}
	matchID, _ := strconv.ParseInt(r.FormValue("match"), 10, 64)
	ftH, _ := strconv.Atoi(r.FormValue("ftHome"))
	ftA, _ := strconv.Atoi(r.FormValue("ftAway"))
	if err := a.setResult(matchID, ftH, ftA); err != nil {
		redirectMsg(w, r, "/admin", "", trMsg(lang, err.Error()))
		return
	}
	redirectMsg(w, r, "/admin", tr(lang, "ok.saved"), "")
}
