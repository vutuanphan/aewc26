(function () {
  var VI = window.LANG === 'vi';
  var L = VI ? {
    refHead: 'Kèo nhà cái (tham khảo)', srcReal: 'đồng thuận nhà cái', srcEst: 'ước lượng BXH FIFA',
    none: 'Chưa có kèo cho trận này.', useAh: 'Dùng mức chấp ', useOu: 'Dùng mức ',
    win: 'thắng', draw: 'Hòa', side: 'Cửa ', over: 'Tài (trên)', under: 'Xỉu (dưới)',
    home: 'Đội nhà', away: 'Đội khách'
  } : {
    refHead: 'Bookmaker odds (reference)', srcReal: 'bookmaker consensus', srcEst: 'FIFA-ranking estimate',
    none: 'No odds for this match yet.', useAh: 'Use handicap ', useOu: 'Use line ',
    win: 'win', draw: 'Draw', side: 'Side ', over: 'Over (higher)', under: 'Under (lower)',
    home: 'Home', away: 'Away'
  };

  // ---- create-bet form ----
  var toggle = document.getElementById('toggleForm');
  var form = document.getElementById('betform');
  if (toggle && form) {
    toggle.addEventListener('click', function () { form.classList.toggle('hidden'); });

    var matchSel = document.getElementById('match');
    var typeSel = document.getElementById('betType');
    var pickWrap = document.getElementById('pickWrap');
    var oddsRef = document.getElementById('oddsRef');
    var lineWrap = document.getElementById('lineWrap');
    var lineInp = document.getElementById('line');
    var FEED = window.FEED || {};

    function radios(opts) {
      pickWrap.innerHTML = opts.map(function (o, i) {
        return '<label><input type="radio" name="pick" value="' + o.v + '"' + (i === 0 ? ' checked' : '') + '>' + o.l + '</label>';
      }).join('');
    }
    function fmt(n) { return (n > 0 ? '+' : '') + n; }
    function od(n) { return n ? n.toFixed(2) : '—'; }
    function pc(n) { return n ? Math.round(n * 100) + '%' : ''; }

    function rebuild() {
      var m = FEED[matchSel.value];
      var home = m ? m.Home : L.home, away = m ? m.Away : L.away;
      var t = typeSel.value;
      if (t === 'wdl') {
        radios([{ v: 'home', l: home + ' ' + L.win }, { v: 'draw', l: L.draw }, { v: 'away', l: away + ' ' + L.win }]);
        lineWrap.classList.add('hidden');
      } else if (t === 'ah') {
        radios([{ v: 'home', l: L.side + home }, { v: 'away', l: L.side + away }]);
        lineWrap.classList.remove('hidden');
        lineInp.value = (m && m.Has && m.AhLine) ? m.AhLine : -0.5;
      } else {
        radios([{ v: 'over', l: L.over }, { v: 'under', l: L.under }]);
        lineWrap.classList.remove('hidden');
        lineInp.value = (m && m.Has && m.OuLine) ? m.OuLine : 2.5;
      }
      renderOdds(m, t, home, away);
    }

    function renderOdds(m, t, home, away) {
      if (!m) { oddsRef.innerHTML = ''; return; }
      var head = '<div class="oh">' + L.refHead + ' · ' + (m.Has ? L.srcReal : L.srcEst) + '</div>';
      var body = '<div class="muted small">' + L.none + '</div>';
      if (t === 'wdl' && m.HomeOdds) {
        body = '<div class="og"><div><span>' + home + '</span><b>' + od(m.HomeOdds) + '</b><i>' + pc(m.PHome) + '</i></div>'
          + '<div><span>' + L.draw + '</span><b>' + od(m.DrawOdds) + '</b><i>' + pc(m.PDraw) + '</i></div>'
          + '<div><span>' + away + '</span><b>' + od(m.AwayOdds) + '</b><i>' + pc(m.PAway) + '</i></div></div>';
      } else if (t === 'ah' && m.AhHome) {
        body = '<div class="og"><div><span>' + home + ' ' + fmt(m.AhLine) + '</span><b>' + od(m.AhHome) + '</b></div>'
          + '<div><span>' + away + ' ' + fmt(-m.AhLine) + '</span><b>' + od(m.AhAway) + '</b></div></div>'
          + '<button type="button" class="useline" data-line="' + m.AhLine + '">' + L.useAh + fmt(m.AhLine) + '</button>';
      } else if (t === 'ou' && m.OuOver) {
        body = '<div class="og"><div><span>' + L.over + ' ' + m.OuLine + '</span><b>' + od(m.OuOver) + '</b></div>'
          + '<div><span>' + L.under + ' ' + m.OuLine + '</span><b>' + od(m.OuUnder) + '</b></div></div>'
          + '<button type="button" class="useline" data-line="' + m.OuLine + '">' + L.useOu + m.OuLine + '</button>';
      }
      oddsRef.innerHTML = head + body;
      var btn = oddsRef.querySelector('.useline');
      if (btn) btn.addEventListener('click', function () { lineInp.value = btn.getAttribute('data-line'); });
    }

    matchSel.addEventListener('change', rebuild);
    typeSel.addEventListener('change', rebuild);
    rebuild();

    var quick = document.getElementById('quick');
    var stake = document.getElementById('stake');
    if (quick && stake) {
      [50, 100, 500, 1000, 5000].forEach(function (q) {
        var b = document.createElement('button');
        b.type = 'button'; b.textContent = q.toLocaleString('vi-VN');
        b.addEventListener('click', function () { stake.value = q; });
        quick.appendChild(b);
      });
    }
  }

  // ---- chat ----
  var chatbox = document.getElementById('chatbox');
  if (chatbox) {
    var stick = function () { chatbox.scrollTop = chatbox.scrollHeight; };
    stick();
    setInterval(function () {
      fetch('/chat/feed?_=' + Date.now(), { credentials: 'same-origin', cache: 'no-store' }).then(function (r) { return r.text(); }).then(function (html) {
        var near = chatbox.scrollHeight - chatbox.scrollTop - chatbox.clientHeight < 80;
        chatbox.innerHTML = html;
        if (near) stick();
      }).catch(function () {});
    }, 4000);
  }
})();
