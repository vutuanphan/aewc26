package app

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"
)

//go:embed web/templates/*.html
var tmplFS embed.FS

//go:embed web/static/*
var staticFS embed.FS

type pageData struct {
	Title    string
	Brand    string
	User     *User
	Active   string
	CSRF     string
	Lang     string
	Flash    string
	FlashErr string
	Data     any
}

func (a *App) parseTemplates() error {
	funcs := template.FuncMap{
		"vnd":         vnd,
		"odd":         odd,
		"pct":         pct,
		"fmtLine":     fmtLineStr,
		"typeLabel":   typeLabel,
		"creatorSide": creatorSide,
		"takerSide":   takerSide,
		"outcome":     outcomeLabel,
		"txnKind":     txnKindLabel,
		"t":           tr,
		"neg":         func(f float64) float64 { return -f },
		"add1":        func(i int) int { return i + 1 },
		"ftime": func(u int64) string {
			if u == 0 {
				return ""
			}
			return time.Unix(u, 0).In(a.loc).Format("15:04 02/01")
		},
	}
	t, err := template.New("").Funcs(funcs).ParseFS(tmplFS, "web/templates/*.html")
	if err != nil {
		return err
	}
	a.tmpl = t
	return nil
}

func (a *App) render(w http.ResponseWriter, name string, pd pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, name, pd); err != nil {
		log.Printf("[render] %s: %v", name, err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// page builds pageData with the common fields filled from the request.
func (a *App) page(r *http.Request, title, active string) pageData {
	return pageData{
		Title:    title,
		Brand:    a.brand,
		User:     currentUser(r),
		Active:   active,
		CSRF:     csrfToken(r),
		Lang:     langFromRequest(r),
		Flash:    r.URL.Query().Get("ok"),
		FlashErr: r.URL.Query().Get("err"),
	}
}

func (a *App) router() http.Handler {
	mux := http.NewServeMux()

	staticSub, _ := fs.Sub(staticFS, "web/static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	mux.HandleFunc("GET /lang", a.handleLang)
	mux.HandleFunc("GET /login", a.handleLoginForm)
	mux.HandleFunc("POST /login", a.handleLogin)
	mux.HandleFunc("POST /logout", a.requireAuth(a.handleLogout))

	mux.HandleFunc("GET /{$}", a.requireAuth(a.handleHome))
	mux.HandleFunc("POST /bets/create", a.requireAuth(a.handleCreateBet))
	mux.HandleFunc("POST /bets/take", a.requireAuth(a.handleTakeBet))
	mux.HandleFunc("POST /bets/cancel", a.requireAuth(a.handleCancelBet))
	mux.HandleFunc("GET /mybets", a.requireAuth(a.handleMyBets))
	mux.HandleFunc("GET /leaderboard", a.requireAuth(a.handleLeaderboard))
	mux.HandleFunc("GET /wallet", a.requireAuth(a.handleWallet))
	mux.HandleFunc("GET /chat", a.requireAuth(a.handleChat))
	mux.HandleFunc("POST /chat", a.requireAuth(a.handleChatPost))
	mux.HandleFunc("GET /chat/feed", a.requireAuth(a.handleChatFeed))

	mux.HandleFunc("GET /admin", a.requireAdmin(a.handleAdmin))
	mux.HandleFunc("POST /admin/grant", a.requireAdmin(a.handleGrant))
	mux.HandleFunc("POST /admin/result", a.requireAdmin(a.handleResult))
	mux.HandleFunc("POST /admin/user/create", a.requireAdmin(a.handleCreateUser))
	mux.HandleFunc("POST /admin/user/resetpw", a.requireAdmin(a.handleResetPw))

	return mux
}
