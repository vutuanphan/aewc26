package app

import "net/http"

// dict maps a key to {english, vietnamese}.
var dict = map[string][2]string{
	// nav
	"nav.bets":   {"Bets", "Kèo"},
	"nav.mine":   {"Mine", "Của tôi"},
	"nav.rank":   {"Ranking", "BXH"},
	"nav.chat":   {"Chat", "Chat"},
	"nav.wallet": {"Wallet", "Ví"},
	"nav.admin":  {"Admin", "Admin"},
	"pts":        {"pts", "điểm"},
	"logout":     {"Log out", "Đăng xuất"},

	// login
	"login.tagline": {"Friends' points betting arena", "Sân cá độ điểm của anh em"},
	"login.user":    {"Username", "Tên đăng nhập"},
	"login.pass":    {"Password", "Mật khẩu"},
	"login.btn":     {"Log in", "Đăng nhập"},
	"login.userph":  {"e.g. tam", "vd: tam"},

	// home / create form
	"home.newbet":   {"➕ New bet", "➕ Tạo kèo mới"},
	"form.match":    {"Match", "Trận đấu"},
	"form.choose":   {"— Pick a match —", "— Chọn trận —"},
	"form.type":     {"Bet type", "Loại kèo"},
	"type.wdl":      {"Win / Draw / Lose (1x2)", "Thắng / Hòa / Thua (1x2)"},
	"type.ah":       {"Asian handicap", "Chấp châu Á"},
	"type.ou":       {"Over / Under", "Tài / Xỉu"},
	"type.cs":       {"Correct score", "Tỷ số chính xác"},
	"form.predscore": {"Predicted score", "Tỷ số dự đoán"},
	"live":          {"LIVE", "LIVE"},
	"form.line":     {"Line (multiple of 0.25)", "Mức kèo (bội số 0.25)"},
	"form.stake":    {"Stake (points)", "Số điểm cược"},
	"form.note":     {"Note (optional)", "Ghi chú (tuỳ chọn)"},
	"form.noteph":   {"a little trash talk", "chém gió tí cho vui"},
	"form.makebet":  {"Place bet", "Ra kèo"},
	"home.openbets": {"Open bets", "Kèo đang mở"},
	"home.nobets":   {"No bets yet. Create the first one!", "Chưa có kèo nào. Tạo kèo đầu tiên đi!"},
	"bet.backs":     {"backs", "đặt"},
	"bet.youtake":   {"You take", "Bạn ăn cửa"},
	"bet.cancel":    {"Cancel", "Huỷ"},
	"bet.take":      {"Take other side", "Bắt cửa ngược"},

	// mybets
	"mine.active":  {"In progress", "Đang diễn ra"},
	"mine.noactive": {"No pending bets.", "Không có kèo đang chờ."},
	"mine.done":    {"Finished", "Đã xong"},
	"mine.nodone":  {"No finished bets yet.", "Chưa có kèo nào kết thúc."},
	"mine.myside":  {"My side:", "Cửa của tôi:"},
	"mine.result":  {"Result:", "Kết quả:"},

	// leaderboard
	"rank.title": {"Leaderboard", "Bảng xếp hạng"},
	"rank.sub":   {"Ranked by current wallet balance", "Xếp theo số điểm ví hiện tại"},
	"rank.you":   {"(you)", "(bạn)"},

	// wallet
	"wallet.balance": {"Current balance", "Số dư hiện tại"},
	"wallet.history": {"Transaction history", "Lịch sử giao dịch"},
	"wallet.none":    {"No transactions yet.", "Chưa có giao dịch."},
	"wallet.left":    {"left", "còn"},

	// chat
	"chat.title":  {"Friends chat", "Chat anh em"},
	"chat.ph":     {"Type a message…", "Nhập tin nhắn…"},
	"chat.send":   {"Send", "Gửi"},
	"chat.none":   {"No messages yet. Break the ice!", "Chưa có tin nhắn. Mở màn đi!"},

	// admin
	"admin.grant":   {"Grant / top-up points", "Phát / nạp điểm"},
	"admin.to":      {"Recipient", "Người nhận"},
	"admin.all":     {"All players", "Tất cả người chơi"},
	"admin.amount":  {"Points", "Số điểm"},
	"admin.mode":    {"Mode", "Kiểu"},
	"admin.add":     {"Add", "Cộng thêm"},
	"admin.set":     {"Set to", "Đặt thành"},
	"admin.memo":    {"Note", "Ghi chú"},
	"admin.update":  {"Update", "Cập nhật"},
	"admin.result":  {"Enter / edit result", "Nhập / sửa kết quả"},
	"admin.match":   {"Match", "Trận"},
	"admin.home":    {"Home goals", "Bàn chủ"},
	"admin.away":    {"Away goals", "Bàn khách"},
	"admin.save":    {"Save result & settle", "Lưu kết quả & chung chi"},
	"admin.settled": {"settled", "đã chốt"},
	"admin.hint":    {"Saving a result auto-settles every matched bet on it.", "Lưu kết quả sẽ tự chung chi mọi kèo đã khớp của trận này."},
	"admin.updated":  {"Updated %d accounts", "Đã cập nhật %d tài khoản"},
	"admin.users":    {"Players", "Người chơi"},
	"admin.newuser":  {"Add player", "Thêm người chơi"},
	"admin.username": {"Username", "Tên đăng nhập"},
	"admin.dname":    {"Display name", "Tên hiển thị"},
	"admin.pw":       {"Password", "Mật khẩu"},
	"admin.startbal": {"Starting balance", "Điểm khởi đầu"},
	"admin.create":   {"Create", "Tạo"},
	"admin.resetpw":  {"Reset", "Đặt lại"},
	"admin.newpw":    {"new password", "mật khẩu mới"},

	// bet phrases (built with team names)
	"w.win":   {"win", "thắng"},
	"w.draw":  {"Draw", "Hòa"},
	"w.side":  {"Side", "Cửa"},
	"w.over":     {"Over", "Tài"},
	"w.under":    {"Under", "Xỉu"},
	"w.or":       {"or", "hoặc"},
	"w.score":    {"Score", "Tỷ số"},
	"w.notscore": {"Not", "Khác"},

	// outcomes
	"o.creator":      {"Creator wins", "Người tạo thắng"},
	"o.taker":        {"Taker wins", "Người bắt thắng"},
	"o.creator_half": {"Creator wins half", "Người tạo thắng nửa"},
	"o.taker_half":   {"Taker wins half", "Người bắt thắng nửa"},
	"o.push":         {"Push — refunded", "Hòa kèo — hoàn điểm"},
	"o.void":         {"Match void — refunded", "Trận huỷ — hoàn điểm"},
	"o.no_taker":     {"No taker — refunded", "Không ai bắt — hoàn điểm"},
	"o.cancelled":    {"Cancelled", "Đã huỷ"},
	"o.live_voided":  {"Voided (goal) — refunded", "Có bàn thắng — hoàn điểm"},

	// bet statuses
	"s.waiting": {"Waiting for a taker", "Đang chờ người bắt"},
	"s.open":    {"Open", "Đang mở"},
	"s.matched": {"Matched — awaiting result", "Đã khớp — chờ kết quả"},

	// txn kinds
	"k.grant":      {"Points granted", "Cấp điểm"},
	"k.stake_lock": {"Bet placed/taken", "Đặt/bắt kèo"},
	"k.refund":     {"Refund", "Hoàn điểm"},
	"k.payout":     {"Payout", "Chung chi"},

	// errors / flashes
	"err.session":  {"Session expired, try again", "Phiên hết hạn, thử lại"},
	"err.login":    {"Wrong username or password", "Sai tên đăng nhập hoặc mật khẩu"},
	"ok.created":   {"Bet created!", "Đã tạo kèo!"},
	"ok.taken":     {"Bet taken!", "Đã bắt kèo!"},
	"ok.cancelled": {"Bet cancelled, points refunded.", "Đã huỷ kèo, hoàn điểm."},
	"ok.saved":     {"Result saved & settled", "Đã lưu kết quả & chung chi"},
	"ok.usercreated": {"Player created", "Đã tạo người chơi"},
	"ok.pwreset":     {"Password reset", "Đã đặt lại mật khẩu"},
}

func langIdx(lang string) int {
	if lang == "vi" {
		return 1
	}
	return 0
}

func tr(lang, key string) string {
	if v, ok := dict[key]; ok {
		return v[langIdx(lang)]
	}
	return key
}

// langFromRequest reads the language cookie; default English.
func langFromRequest(r *http.Request) string {
	if c, err := r.Cookie("lang"); err == nil && c.Value == "vi" {
		return "vi"
	}
	return "en"
}

// errEN maps backend (Vietnamese) error strings to English.
var errEN = map[string]string{
	"không đủ điểm":                     "Not enough points",
	"không tìm thấy trận":               "Match not found",
	"trận đã bắt đầu hoặc kết thúc":      "Match already started or finished",
	"đã qua giờ bóng lăn":               "Kickoff time has passed",
	"loại kèo / cửa không hợp lệ":        "Invalid bet type / pick",
	"số điểm cược phải ≥ 1":             "Stake must be ≥ 1",
	"mức kèo phải là bội số 0.25":        "Line must be a multiple of 0.25",
	"mức tài/xỉu phải > 0":              "Over/Under line must be > 0",
	"không tìm thấy kèo":                "Bet not found",
	"kèo đã được bắt hoặc đã đóng":       "Bet already taken or closed",
	"không thể tự bắt kèo của mình":      "You can't take your own bet",
	"chỉ người tạo mới huỷ được":         "Only the creator can cancel",
	"kèo đã được bắt, không huỷ được":    "Bet already taken, can't cancel",
	"tên đăng nhập đã tồn tại":           "Username already exists",
	"thông tin không hợp lệ":            "Invalid input",
	"mật khẩu tối thiểu 4 ký tự":         "Password must be at least 4 characters",
}

// trMsg localizes a backend message for the request language.
func trMsg(lang, msg string) string {
	if lang == "en" {
		if en, ok := errEN[msg]; ok {
			return en
		}
	}
	return msg
}
