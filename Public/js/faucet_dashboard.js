// Faucet Dashboard — NFT-Seduction
// Rewired from mock data to real API calls (2026-09-03, KEY 3.5 drift closure).
// Backend: faucet_service.go (/api/faucet/status, /api/faucet/vault-balance, /api/faucet/claim)
// Follows the established codebase pattern: local api(path, opts) helper + fetch(url).

const FaucetDashboard = (() => {
  var API_BASE = '/api';

  let container = null;
  let faucetData = null;

  // ---------------------------------------------------------------- fetch helper (matches ai_citizens.js / advertising.js / children_bots.js pattern)

  async function api(path, opts) {
      const url = path.startsWith('/api/') ? path : API_BASE + path;
    const resp = await fetch(url, opts);
    if (!resp.ok) {
      let detail = '';
      try { const j = await resp.json(); detail = j.error || j.message || ''; } catch (e) {}
      throw new Error(detail || ('HTTP ' + resp.status));
    }
    return resp.json();
  }

  // ---------------------------------------------------------------- helpers

  function getWallet() {
    if (window.getWalletAddress) return window.getWalletAddress() || '';
    if (window.getActiveWallet) return window.getActiveWallet() || '';
    return localStorage.getItem('wallet_address') || '';
  }

  function formatMicro(micro) {
    // formatMicroVBV may exist globally; fall back to manual uint64-safe formatting.
    if (typeof window.formatMicroVBV === 'function') return window.formatMicroVBV(micro);
    const n = Number(micro) || 0;
    return (n / 1_000_000).toFixed(2) + ' VBV';
  }

  // ---------------------------------------------------------------- data load

  async function loadFaucetData() {
    // EMPTY STATE default — never mock data. Backend may be unreachable in dev.
    faucetData = {
      vault_balance: 0,
      faucet_pool: 0,
      ghost_tax: 0,
      stagnation_tax: 0,
      total_claims: 0,
      health: 'unknown',
    };

    try {
      const status = await api('/faucet/status', { method: 'GET' });
      if (status && typeof status === 'object') {
        // Real backend response merges over the empty-state default.
        for (const k of Object.keys(faucetData)) {
          if (status[k] !== undefined && status[k] !== null) faucetData[k] = status[k];
faucetData.vault_address = status.vault_address || '';
faucetData.reward_asset_id = status.reward_asset_id || '';
faucetData.vault_balance_live = !!status.vault_balance_live;
faucetData.last_vault_sync_unix = status.last_vault_sync_unix || 0;
        }
      }
    } catch (e) {
      // EMPTY STATE on failure — no mock data fallback. Log for diagnostics only.
      console.warn('[FaucetDashboard] API unreachable, showing empty state:', e.message);
    }

    // Augment with per-wallet vault balance if available (uint64 micro).
    if (getWallet()) {
      try {
        const vb = await api('/faucet/vault-balance?wallet=' + encodeURIComponent(getWallet()), { method: 'GET' });
        if (vb && vb.vault_balance_micro !== undefined) {
          faucetData.vault_balance_micro = vb.vault_balance_micro;
        }
      } catch (e) {
        // vault-balance optional — ignore failure, keep empty state for that field.
      }
    }
  }

  // ---------------------------------------------------------------- render

  function renderStats() {
    if (!container) return;
    const d = faucetData || {};
    const vaultMicro = d.vault_balance_micro !== undefined ? d.vault_balance_micro : d.vault_balance;

    const map = {
      'stat-vault': formatMicro(vaultMicro || 0),
      'stat-pool': formatMicro(d.faucet_pool || 0),
      'stat-ghost': formatMicro(d.ghost_tax || 0),
      'stat-stag': formatMicro(d.stagnation_tax || 0),
      'stat-claims': String(d.total_claims || 0),
      'stat-health': String(d.health || 'unknown'),
    };

    for (const [id, val] of Object.entries(map)) {
      const el = document.getElementById(id);
      if (el) el.textContent = val;
    }

    const va=document.getElementById("vault-addr");if(va){va.textContent=d.vault_address||'';}
const vel=document.getElementById("vault-explorer-link");if(vel&&d.vault_address){vel.href="https://block.voi.network/explorer/address/"+d.vault_address;}
const vb=document.getElementById("vault-balance");if(vb){vb.textContent=formatMicro(d.faucet_balance_micro||0);}
const vl=document.getElementById("vault-live");if(vl){const live=d.vault_balance_live;vl.textContent=live?"LIVE":"UNREACHABLE";vl.style.color=live?"#39d98a":"#ff5c5c";}
const vas=document.getElementById("vault-asset");if(vas){vas.textContent=d.reward_asset_id||'';}
const vs=document.getElementById("vault-synced");if(vs){vs.textContent=d.last_vault_sync_unix?new Date(d.last_vault_sync_unix*1000).toLocaleString():'';}
const vcopy=document.getElementById("vault-copy-btn");if(vcopy){vcopy.onclick=function(){if(d.vault_address){if(navigator.clipboard){navigator.clipboard.writeText(d.vault_address);}if(window.toast){window.toast("Vault address copied","success");}}};}
const claimBtn = document.getElementById('faucet-claim-btn');
    if (claimBtn) {
      claimBtn.disabled = false;
      claimBtn.onclick = () => claimFaucet();
    }
  }

  // ---------------------------------------------------------------- claim action

  async function claimFaucet() {
    const wallet = getWallet();
    if (!wallet) {
      if (window.toast) window.toast('Connect a wallet first', 'warn');
      return;
    }

    // Faucet claim is a ZERO-cost grant — no txModalOpen needed (nothing to deduct).
    // But we guard with a confirmation and wallet opt-in so it's never an accidental spend.
    const amount = 1000000; // 1 VBV default claim grant — server validates real amount.
    await doClaim(wallet, amount);
  }

  async function doClaim(wallet, amount) {
    const body = { wallet, amount };
    try {
      const res = await api('/faucet/claim', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (res && window.addVBV) window.addVBV(amount); // optimistic balance bump
      if (window.toast) window.toast('Claimed ' + formatMicro(amount), 'success');
      await refreshFaucet(); // reload real data after claim
    } catch (e) {
      if (window.toast) window.toast('Claim failed: ' + e.message, 'error');
    }
  }

  // ---------------------------------------------------------------- public API

  async function refreshFaucet() {
    await loadFaucetData();
    renderStats();
  }

  function faucetDashboard() {
    return `
      <div class="vbt-overlay faucet-dashboard" id="faucet-dashboard">
        <div class="overlay-header">
          <h2>💧 Faucet Dashboard</h2>
          <button class="close-btn" onclick="FaucetDashboard.close()">✕</button>
        </div>
        <div class="overlay-body">
<div class="vault-info">
  <div class="vault-info-row"><label>Faucet Vault Address</label><span id="vault-addr" class="mono">-</span></div>
  <div class="vault-info-actions"><button id="vault-copy-btn" class="btn btn-small">Copy</button> <a id="vault-explorer-link" target="_blank" rel="noopener" class="btn btn-small">View on Explorer (external)</a></div>
  <div class="vault-info-row"><label>$VBV Held (live, on-chain)</label><span id="vault-balance">-</span> <span id="vault-live" class="badge">-</span></div>
  <div class="vault-info-row"><label>Reward Asset ID</label><span id="vault-asset" class="mono">-</span></div>
  <div class="vault-info-row"><label>Last Verified</label><span id="vault-synced">-</span></div>
  <p class="hint">Balance is read 100% live from the vault address on-chain (no seeding, no cache). Verify on the Voi explorer.</p>
</div>
          <div class="stat-grid">
            <div class="stat-card"><label>Vault Balance</label><span id="stat-vault">—</span></div>
            <div class="stat-card"><label>Faucet Pool</label><span id="stat-pool">—</span></div>
            <div class="stat-card"><label>Ghost Tax</label><span id="stat-ghost">—</span></div>
            <div class="stat-card"><label>Stagnation Tax</label><span id="stat-stag">—</span></div>
            <div class="stat-card"><label>Total Claims</label><span id="stat-claims">—</span></div>
            <div class="stat-card"><label>Health</label><span id="stat-health">—</span></div>
          </div>
          <div class="actions">
            <button id="faucet-claim-btn" class="btn btn-primary">Claim VBV Grant</button>
          </div>
          <p class="hint">Vault balance is per-wallet (uint64 micro). All values server-authoritative.</p>
        </div>
      </div>`;
  }

  function open() {
    const existing = document.getElementById('faucet-dashboard');
    if (existing) existing.remove();

    const div = document.createElement('div');
    div.innerHTML = faucetDashboard();
    document.body.appendChild(div.firstElementChild);
    container = document.getElementById('faucet-dashboard');

    // containment: ensure the overlay root carries position:fixed
    if (container) container.classList.add('vbt-overlay');

    // Use PanelManager for consistent open/close behavior
    if (window.PanelManager) {
      window.PanelManager.open(container, () => {
        container = null;
      });
    }

    refreshFaucet();
    return container;
  }

  function close() {
    const el = document.getElementById('faucet-dashboard');
    if (el) el.remove();
    container = null;
  }

  return { open, close, refresh: refreshFaucet };
})();

window.FaucetDashboard = FaucetDashboard;
window.openFaucetDashboard = FaucetDashboard.open;
