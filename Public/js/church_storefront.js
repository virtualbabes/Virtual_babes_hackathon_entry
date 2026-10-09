// ============================================================================
// church_storefront.js — Faith Economy & Church Management
// ----------------------------------------------------------------------------
// Exposes faith_church.go: rituals, members, items, leaderboard
// Unique visual style: cathedral/gothic with stained glass effects
// ============================================================================

var API_BASE = '/api';

// --- Dogma Types ---
const DOGMA_TYPES = [
    { id: 'purist', name: 'Purist', icon: '⚔️', desc: 'Heresy-war rivalry bonus', color: '#f44336' },
    { id: 'syncretic', name: 'Syncretic', icon: '🤝', desc: 'Coalition synergy', color: '#4caf50' },
    { id: 'orthodox', name: 'Orthodox', icon: '📜', desc: 'Same-faith dominance', color: '#9c27b0' },
];

// --- State ---
// `faithCoherence` and the FAITH_ITEMS table were REMOVED: the first was a local counter the UI
// presented as an economic figure ("+10 Faith Coherence") that no engine ever saw, and the second
// was a hard-coded shop with invented prices and no route behind it. A leaf must not display a
// number or a price that the server has not reported.
let churchData = null;
let ritualHistory = [];

// --- Init ---
export function initChurchStorefront() {
    const el = document.getElementById('wd-church');
    if (!el) return;
    renderChurchLoading(el);
    fetchChurchData(el);
}

async function fetchChurchData(el) {
    try {
        const resp = await fetch(`${API_BASE}/church/owner`);
        if (!resp.ok) throw new Error('Failed to fetch church data');
        churchData = await resp.json();
        renderChurchStorefront(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Church data unavailable</p>';
    }
}

function renderChurchLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading cathedral…</div>';
}

function renderChurchStorefront(el) {
    const churches = churchData?.churches || [];
    const totalIncome = churchData?.total_income_micro || 0;

    let html = `
        <div class="church-storefront-container">
            <!-- Cathedral Header -->
            <div class="cathedral-header">
                <div class="cathedral-stained-glass">
                    <div class="stained-pane" style="background:linear-gradient(135deg, #f44336, #ff9800)"></div>
                    <div class="stained-pane" style="background:linear-gradient(135deg, #2196f3, #4caf50)"></div>
                    <div class="stained-pane" style="background:linear-gradient(135deg, #9c27b0, #e91e63)"></div>
                </div>
                <div class="cathedral-title">
                    <span class="cathedral-icon">⛪</span>
                    <div>
                        <h3 style="color:#e5e7eb;margin:0;">Cathedral Storefront</h3>
                        <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Manage your faith economy</p>
                    </div>
                </div>
                <div class="faith-coherence-display">
                    <span class="faith-coherence-label">Faith Coherence</span>
                    <!-- This used to print a LOCAL counter that defaulted to 0 and was incremented by
                         the client's own (fabricated) ritual, so it showed 10 after one click and
                         the engine never knew. It now prints what the engine reported, or says so. -->
                    <span class="faith-coherence-value" style="color:${faithCoherenceValue() === null ? '#90a4ae' : (faithCoherenceValue() > 50 ? '#4caf50' : faithCoherenceValue() > 20 ? '#ff9800' : '#f44336')}">${faithCoherenceValue() === null ? 'not reported' : faithCoherenceValue()}</span>
                </div>
            </div>

            <!-- Dogma Selection -->
            <div class="dogma-section">
                <h4 style="color:#e5e7eb;">🕊️ Dogma Path</h4>
                <div class="dogma-cards">
                    ${DOGMA_TYPES.map(d => `
                        <div class="dogma-card" data-dogma="${d.id}" style="--dogma-color:${d.color}">
                            <span class="dogma-icon">${d.icon}</span>
                            <span class="dogma-name">${d.name}</span>
                            <span class="dogma-desc">${d.desc}</span>
                        </div>
                    `).join('')}
                </div>
            </div>

            <!-- Church List -->
            ${churches.length > 0 ? `
                <div class="church-list-section">
                    <h4 style="color:#e5e7eb;">🏛️ Your Churches</h4>
                    <div class="church-grid">
                        ${churches.map(c => renderChurchCard(c)).join('')}
                    </div>
                </div>
            ` : renderNoChurches()}

            <!-- Faith Store. The four items below were HARD-CODED client data with invented prices
                 (500 / 1,000 / 2,500 / 5,000 $VBV) and no endpoint behind them, and "Buy" toasted
                 "Purchasing …" while sending NOTHING — a receipt for a request that was never made.
                 No route serves these SKUs or those prices, so the store is stated as unbacked
                 rather than shown as a shop the player cannot use. -->
            <div class="faith-store-section">
                <h4 style="color:#e5e7eb;">🛒 Faith Store</h4>
                <p class="church-note">Not served yet — no engine route sells faith items, so no price here is authoritative.</p>
            </div>

            <!-- Church lookup. POST/GET /api/church/get had NO owner anywhere in the client: it is
                 the only way to read ONE church by id (handleChurchGet → faithChurchEngine
                 .GetChurch). Placed here because reading a church IS this leaf's domain. -->
            <div class="church-lookup-section">
                <h4 style="color:#e5e7eb;">🔎 Church Lookup</h4>
                <div class="church-lookup-row">
                    <input id="church-lookup-id" type="text" placeholder="Church id" />
                    <button class="vbt-btn vbt-btn-primary" id="church-lookup-btn">Read</button>
                </div>
                <div id="church-lookup-out" class="church-lookup-out"></div>
            </div>

            <!-- Ritual History -->
            ${ritualHistory.length > 0 ? `
                <div class="ritual-history-section">
                    <h4 style="color:#e5e7eb;">📜 Recent Rituals</h4>
                    <div class="ritual-history-list">
                        ${ritualHistory.map(r => `
                            <div class="ritual-history-item">
                                <span class="ritual-icon">${r.icon}</span>
                                <span class="ritual-name">${r.name}</span>
                                <span class="ritual-effect">+${r.effect} coherence</span>
                            </div>
                        `).join('')}
                    </div>
                </div>
            ` : ''}
        </div>
    `;

    el.innerHTML = html;
    attachChurchListeners(el);
}

function renderChurchCard(church) {
    const members = church.members || church.Members || {};
    const items = church.items || church.Items || {};
    const rituals = church.rituals_done || church.RitualsDone || 0;

    return `
        <div class="church-card">
            <div class="church-card-header">
                <span class="church-card-name">${escapeHtml(church.name || church.Name)}</span>
                <span class="church-card-dogma ${church.dogma || church.DogmaTag || 'orthodox'}">${(church.dogma || church.DogmaTag || 'ORTHO').toUpperCase()}</span>
            </div>
            <div class="church-card-stats">
                <div class="church-stat">
                    <span class="church-stat-val">${Object.keys(members).length}</span>
                    <span class="church-stat-label">Members</span>
                </div>
                <div class="church-stat">
                    <span class="church-stat-val">${Object.keys(items).length}</span>
                    <span class="church-stat-label">Items</span>
                </div>
                <div class="church-stat">
                    <span class="church-stat-val">${rituals}</span>
                    <span class="church-stat-label">Rituals</span>
                </div>
            </div>
            <div class="church-card-actions">
                <button class="vbt-btn vbt-btn-primary church-perform-ritual">🕯️ Ritual</button>
                <button class="vbt-btn vbt-btn-secondary church-add-member">👥 Invite</button>
            </div>
        </div>
    `;
}

// The engine's own coherence figure for the church payload, or null when it reports none. The
// panel must never substitute a client-side number for an economic one.
function faithCoherenceValue() {
    const raw = churchData && (churchData.faith_coherence ?? churchData.coherence);
    const n = Number(raw);
    return Number.isFinite(n) ? n : null;
}

function renderNoChurches() {
    return `
        <div class="church-empty">
            <span class="church-empty-icon">⛪</span>
            <p>No churches yet. Found a church to begin your faith journey!</p>
            <button class="vbt-btn vbt-btn-primary" onclick="if(window.openFaithChurch) window.openFaithChurch()">🏛️ Faith &amp; Church</button>
        </div>
    `;
}

function attachChurchListeners(el) {
    // Dogma selection
    el.querySelectorAll('.dogma-card').forEach(card => {
        card.addEventListener('click', () => {
            el.querySelectorAll('.dogma-card').forEach(c => c.classList.remove('selected'));
            card.classList.add('selected');
        });
    });

    // Perform ritual — the REAL route, and the ENGINE's number is what gets shown.
    el.querySelectorAll('.church-perform-ritual').forEach(btn => {
        btn.addEventListener('click', () => {
            performRitual(el);
        });
    });

    // Church lookup — /api/church/get, the capability that had no owner.
    const lookupBtn = document.getElementById('church-lookup-btn');
    if (lookupBtn) lookupBtn.addEventListener('click', async () => {
        const id = document.getElementById('church-lookup-id')?.value.trim();
        const out = document.getElementById('church-lookup-out');
        if (!out) return;
        if (!id) { out.innerHTML = '<p class="church-note">A church id is required.</p>'; return; }
        try {
            const res = await api('/church/get?church_id=' + encodeURIComponent(id));
            out.innerHTML = `<pre class="church-lookup-json">${escapeHtml(JSON.stringify(res.church, null, 2))}</pre>`;
        } catch (e) {
            out.innerHTML = `<p class="church-note">Could not read: ${escapeHtml(e.message)}</p>`;
        }
    });
}

// fetch uses API_BASE = '/api'; `api()` prefixes only when the path is not already absolute.
async function api(path, opts) {
    const url = path.startsWith('/api/') ? path : API_BASE + path;
    const resp = await fetch(url, opts);
    if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch (e) {} throw new Error(d || ('HTTP ' + resp.status)); }
    return resp.json();
}

async function performRitual(el) {
    // This used to be fabricated state: `faithCoherence += 10` and a toast promising
    // "+10 Faith Coherence" — a local counter that no engine ever saw, presented as an economic
    // outcome. It now calls the real route and RE-READS, so what is shown is the engine's.
    const id = prompt('Church id to perform the ritual for:');
    if (!id) return;
    try {
        await api('/church/ritual', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ church_id: id }),
        });
        if (window.showToast) window.showToast('🕯️ Ritual recorded.', 'success');
        await fetchChurchData(el);
    } catch (e) {
        if (window.showToast) window.showToast('Ritual refused: ' + e.message, 'error');
    }
}

// --- Global handlers ---
// `window.openChurchFoundry` was REMOVED: it was `alert('Church foundry opening...')` — a stub that
// announced a surface it never opened, so the "Found Church" control was a lie. The leaf now points
// at the real Faith & Church surface, and church CREATION belongs to the Community Console, which
// owns /api/church/* (add-item, add-member, remove-member, ritual).

window.initChurchStorefront = initChurchStorefront;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}

function formatVBV(micro) {
    const vbv = Number(micro) / 1_000_000;
    if (vbv >= 1_000_000) return `${(vbv / 1_000_000).toFixed(1)}M`;
    if (vbv >= 1_000) return `${(vbv / 1_000).toFixed(1)}K`;
    return vbv.toFixed(2) + ' $VBV';
}
