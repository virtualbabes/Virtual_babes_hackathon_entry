// Match Arena — spectator match list + wagering (rebuilt 2026-09-09).
// GET  /api/match/active -> { count, matches:[{id,p1_id,p2_id,rating,territory,spectator_count,start_time}] }
// POST /api/match/wager  -> { spectator_wallet, match_id, bet_on_wallet, wager_micro }
(function () {
  let arenaEl = null;

  function esc(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  }

  function build() {
    if (arenaEl) return;
    arenaEl = document.createElement('div');
    arenaEl.id = 'match-arena-overlay';
    arenaEl.className = 'overlay vbt-overlay';
    arenaEl.style.display = 'none';
    arenaEl.innerHTML = `
      <div class="neon-glass-panel match-arena-panel">
        <div class="match-arena-header">
          <h2>🥊 Match Arena</h2>
          <button class="vbt-btn vbt-btn-secondary" onclick="window.closeMatchArena()">✕</button>
        </div>
        <div id="match-arena-list" class="match-arena-list scroll-y"></div>
        <p class="pp-hint">Spectate live matches and wager on the outcome. Winnings settle to your $VBV balance.</p>
      </div>`;
    document.body.appendChild(arenaEl);
  }

  async function load() {
    const list = document.getElementById('match-arena-list');
    if (!list) return;
    list.innerHTML = '<div class="pp-loading">Loading matches…</div>';
    try {
      const resp = await fetch('/api/match/active');
      if (!resp.ok) throw new Error('HTTP ' + resp.status);
      const data = await resp.json();
      const matches = Array.isArray(data.matches) ? data.matches : [];
      if (matches.length === 0) { list.innerHTML = '<div class="pp-empty">No active matches right now.</div>'; return; }
      list.innerHTML = matches.map(renderCard).join('');
    } catch (e) {
      list.innerHTML = '<div class="pp-empty">Failed to load matches: ' + esc(e.message) + '</div>';
    }
  }

  function renderCard(m) {
    const started = m.start_time ? new Date(m.start_time).toLocaleTimeString() : '—';
    const p1 = m.p1_id ? m.p1_id.slice(0, 8) : '?';
    const p2 = m.p2_id ? m.p2_id.slice(0, 8) : '?';
    return `<div class="match-card" data-match-id="${esc(m.id)}">
      <div class="match-card-head">${esc(p1)} <span class="rv-vs">vs</span> ${esc(p2)}</div>
      <div class="match-card-meta">Rating ${esc(m.rating || '—')} · Territory ${esc(m.territory || '—')} · 👁 ${m.spectator_count || 0} · ⏱ ${esc(started)}</div>
      <div class="match-card-actions">
        <select class="glass-input match-bet-on" data-match-id="${esc(m.id)}">
          <option value="${esc(m.p1_id)}">Bet P1 (${esc(p1)})</option>
          <option value="${esc(m.p2_id)}">Bet P2 (${esc(p2)})</option>
        </select>
        <input type="number" min="1" class="glass-input match-wager" data-match-id="${esc(m.id)}" placeholder="Wager ($VBV micro)" />
        <button class="vbt-btn vbt-btn-primary" onclick="window.placeMatchWager('${esc(m.id)}')">WAGER</button>
      </div>
    </div>`;
  }

  window.placeMatchWager = async function (matchId) {
    const sel = document.querySelector('.match-bet-on[data-match-id="' + matchId + '"]');
    const inp = document.querySelector('.match-wager[data-match-id="' + matchId + '"]');
    if (!sel || !inp) return;
    const betOn = sel.value;
    const wagerMicro = parseInt(inp.value, 10);
    if (!betOn || !wagerMicro || wagerMicro <= 0) { if (window.showToast) window.showToast('Enter a valid wager amount', 'error'); return; }
    const state = (typeof window.GetGameState === 'function') ? window.GetGameState() : {};
    const spectator = state.wallet || state.VAULT_ADDRESS || (window.CONFIG && window.CONFIG.VAULT_ADDRESS) || '';
    if (!spectator) { if (window.showToast) window.showToast('Connect a wallet first', 'error'); return; }
    try {
      const resp = await fetch('/api/match/wager', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ spectator_wallet: spectator, match_id: matchId, bet_on_wallet: betOn, wager_micro: wagerMicro })
      });
      if (!resp.ok) { const d = await resp.json().catch(() => ({})); throw new Error(d.error || ('HTTP ' + resp.status)); }
      if (window.showToast) window.showToast('✅ Wager placed on ' + matchId.slice(0, 8), 'success');
      load();
    } catch (e) {
      if (window.showToast) window.showToast('❌ Wager failed: ' + e.message, 'error');
    }
  };

  window.openMatchArena = function () {
    if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
    build();
    arenaEl.style.display = 'flex';
    load();
  };

  window.closeMatchArena = function () {
    if (arenaEl) arenaEl.style.display = 'none';
  };
})();
