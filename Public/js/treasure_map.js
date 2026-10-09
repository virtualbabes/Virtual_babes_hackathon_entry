// ============================================================================
// treasure_map.js — Treasure Hunt & Exploration Map
// ----------------------------------------------------------------------------
// Exposes treasure_cache.go: spawn, claim, map view, locations
// Unique visual style: pirate map / treasure hunt with X marks
// ============================================================================

var API_BASE = '/api';

// --- State ---
let treasures = [];
let claimedCount = 0;

// --- Init ---
export function initTreasureMap() {
    const el = document.getElementById('wd-treasure');
    if (!el) return;
    renderTreasureLoading(el);
    fetchTreasureData(el);
}

async function fetchTreasureData(el) {
    try {
        const resp = await fetch(`${API_BASE}/treasure/spawn`);
        if (!resp.ok) throw new Error('Failed to fetch treasure data');
        const data = await resp.json();
        treasures = data.treasures || [];
        claimedCount = data.claimed || 0;
        renderTreasureMap(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Treasure data unavailable</p>';
    }
}

function renderTreasureLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading treasure map…</div>';
}

function renderTreasureMap(el) {
    let html = `
        <div class="treasure-map-container">
            <!-- Map Header -->
            <div class="treasure-header">
                <div class="treasure-icon">🗺️</div>
                <div class="treasure-info">
                    <h3 style="color:#e5e7eb;margin:0;">Treasure Map</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Find and claim hidden treasures</p>
                </div>
                <div class="treasure-stats">
                    <div class="treasure-stat">
                        <span class="treasure-stat-num">${treasures.length}</span>
                        <span class="treasure-stat-label">Hidden</span>
                    </div>
                    <div class="treasure-stat">
                        <span class="treasure-stat-num">${claimedCount}</span>
                        <span class="treasure-stat-label">Claimed</span>
                    </div>
                </div>
            </div>

            <!-- Map Grid -->
            <div class="treasure-map-grid">
                ${treasures.length > 0 ? treasures.map(t => renderTreasureCard(t)).join('') : renderEmptyTreasures()}
            </div>

            <!-- Spawn Treasure -->
            <div class="treasure-spawn-section">
                <h4 style="color:#e5e7eb;">✨ Spawn Treasure</h4>
                <div class="treasure-spawn-form">
                    <div class="treasure-form-row">
                        <label>Region</label>
                        <input type="text" id="treasure-region" placeholder="region-1" />
                    </div>
                    <div class="treasure-form-row">
                        <label>Reward ($VBV)</label>
                        <input type="number" id="treasure-reward" min="1" value="100" />
                    </div>
                    <div class="treasure-form-row">
                        <label>Hidden?</label>
                        <select id="treasure-hidden">
                            <option value="false">No</option>
                            <option value="true">Yes</option>
                        </select>
                    </div>
                    <button class="vbt-btn vbt-btn-primary" id="treasure-spawn-btn">🗺️ Spawn</button>
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachTreasureListeners(el);
}

function renderTreasureCard(treasure) {
    const isClaimed = treasure.claimed || false;
    const isHidden = treasure.hidden || false;

    return `
        <div class="treasure-card ${isClaimed ? 'claimed' : ''} ${isHidden ? 'hidden' : ''}">
            <div class="treasure-card-icon">${isClaimed ? '✓' : isHidden ? '❓' : '💰'}</div>
            <div class="treasure-card-info">
                <span class="treasure-card-region">${escapeHtml(treasure.region || 'Unknown')}</span>
                <span class="treasure-card-reward">${formatVBV(treasure.reward_micro || 0)}</span>
            </div>
            ${!isClaimed ? `
                <button class="vbt-btn vbt-btn-primary treasure-claim-btn" data-id="${treasure.id}">
                    Claim
                </button>
            ` : '<span class="treasure-claimed-badge">CLAIMED</span>'}
        </div>
    `;
}

function renderEmptyTreasures() {
    return `
        <div class="treasure-empty">
            <span class="treasure-empty-icon">🗺️</span>
            <p>No treasures hidden. Spawn one to begin the hunt!</p>
        </div>
    `;
}

function attachTreasureListeners(el) {
    // Claim buttons
    el.querySelectorAll('.treasure-claim-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const id = btn.dataset.id;
            claimTreasure(id);
        });
    });

    // Spawn button
    const spawnBtn = el.querySelector('#treasure-spawn-btn');
    if (spawnBtn) {
        spawnBtn.addEventListener('click', () => {
            const region = el.querySelector('#treasure-region')?.value.trim();
            const reward = parseFloat(el.querySelector('#treasure-reward')?.value) * 1000000 || 0;
            const hidden = el.querySelector('#treasure-hidden')?.value === 'true';
            spawnTreasure(region, reward, hidden);
        });
    }
}

async function claimTreasure(id) {
    try {
        const resp = await fetch(`${API_BASE}/treasure/claim`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ cache_id: id })
        });
        if (!resp.ok) throw new Error('Failed to claim treasure');
        if (window.showToast) window.showToast('💰 Treasure claimed!', 'success');
    } catch (e) {
        if (window.showToast) window.showToast(`❌ ${e.message}`, 'error');
    }
}

async function spawnTreasure(region, reward, hidden) {
    try {
        const resp = await fetch(`${API_BASE}/treasure/spawn`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ region, reward_micro: reward, hidden })
        });
        if (!resp.ok) throw new Error('Failed to spawn treasure');
        if (window.showToast) window.showToast('🗺️ Treasure hidden!', 'success');
    } catch (e) {
        if (window.showToast) window.showToast(`❌ ${e.message}`, 'error');
    }
}

// --- Global handlers ---
window.initTreasureMap = initTreasureMap;

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
