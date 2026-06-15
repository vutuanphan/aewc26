package app

// User is a player (or the admin) with a points wallet.
type User struct {
	ID       int64
	Username string
	Name     string
	Balance  int64
	IsAdmin  bool
}

// Team is a World Cup team.
type Team struct {
	Fifa          string
	Name          string
	ISO2          string
	Confederation string
}

// Match is a fixture.
type Match struct {
	ID          int64
	ExtID       string
	Stage       string
	GroupLetter string
	RoundLabel  string
	Num         int
	KickoffUnix int64
	HomeFifa    string
	AwayFifa    string
	HomeLabel   string
	AwayLabel   string
	Status      string // scheduled | finished | cancelled | postponed
	FtHome      int
	FtAway      int
	Settled     bool

	// joined for display
	HomeName string
	AwayName string
	HomeISO  string
	AwayISO  string
}

// HomeDisplay returns the home team's display name (or its placeholder label).
func (m Match) HomeDisplay() string {
	if m.HomeName != "" {
		return m.HomeName
	}
	if m.HomeLabel != "" {
		return m.HomeLabel
	}
	return "?"
}

// AwayDisplay returns the away team's display name (or its placeholder label).
func (m Match) AwayDisplay() string {
	if m.AwayName != "" {
		return m.AwayName
	}
	if m.AwayLabel != "" {
		return m.AwayLabel
	}
	return "?"
}

// Resolved reports whether both teams are known (so it can be bet on).
func (m Match) Resolved() bool {
	return m.HomeFifa != "" && m.AwayFifa != ""
}

// Odds holds bookmaker reference lines for a match.
type Odds struct {
	MatchID                       int64
	PHome, PDraw, PAway           float64
	HomeOdds, DrawOdds, AwayOdds  float64
	AhLine, AhHome, AhAway        float64
	OuLine, OuOver, OuUnder       float64
	Source                        string
}

// Bet is a head-to-head wager.
type Bet struct {
	ID            int64
	MatchID       int64
	BetType       string // wdl | ah | ou
	CreatorID     int64
	TakerID       int64 // 0 if open
	Stake         int64
	Pick          string
	Line          float64
	Status        string // open | matched | settled | cancelled | void
	Outcome       string
	CreatorReturn int64
	TakerReturn   int64
	Note          string
	CreatedUnix   int64

	// joined for display
	Match       Match
	CreatorName string
	TakerName   string
}

// Txn is a wallet ledger entry.
type Txn struct {
	ID           int64
	UserID       int64
	Amount       int64
	Kind         string // grant | stake_lock | refund | payout
	BetID        int64
	BalanceAfter int64
	Memo         string
	CreatedUnix  int64
}

// ChatMsg is one chat message.
type ChatMsg struct {
	ID          int64
	UserID      int64
	UserName    string
	Body        string
	CreatedUnix int64
}
