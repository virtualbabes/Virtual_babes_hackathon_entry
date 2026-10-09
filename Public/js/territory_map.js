// ============================================================================
// territory_map.js — Territory Control & District Contest
// ----------------------------------------------------------------------------
// Exposes club_service.go: territories, districts, control, contest
// Unique visual style: fantasy map with territory borders and flags
// ============================================================================

var API_BASE = '/api';

// --- State ---
let territories = [];
let controlledTerritories = 0;

// --- Init ---
export function initTerritoryMap() {
    const el = document.getElementById('wd-territory');
    if (!el) return;
    renderTerritoryLoading(el);
    fetchTerritoryData(el);
}

async function fetchTerritoryData(el) {
    try {
        const resp = await fetch(`${API_BASE}/regions`);
        if (!resp.ok) throw new Error('Failed to fetch territory data');
        const data = await resp.json();
        territories = data.regions || [];
        controlledTerritories = territories.filter(r => r.controlled_by).length;
        renderTerritoryMap(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Territory data unavailable</p>';
    }
}

function renderTerritoryLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading map…</div>';
}

function renderTerritoryMap(el) {
    let html = `
        <div class="territory-map-container">
            <!-- Map Header -->
            <div class="territory-header">
                <div class="territory-icon">🗺️</div>
                <div class="territory-info">
                    <h3 style="color:#e5e7eb;margin:0;">Territory Map</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Control districts, expand your empire</p>
                </div>
                <div class="territory-control">
                    <span class="territory-control-num">${controlledTerritories}/${territories.length}</span>
                    <span class="territory-control-label">Controlled</span>
                </div>
            </div>

            <!-- Map Grid -->
            <div class="territory-grid">
                ${territories.length > 0 ? territories.map(t => renderTerritoryCard(t)).join('') : renderSampleTerritories()}
            </div>

            <!-- Legend -->
            <div class="territory-legend">
                <span class="legend-title">Control:</span>
                <span class="legend-item"><span class="legend-dot" style="background:#4caf50"></span>Yours</span>
                <span class="legend-item"><span class="legend-dot" style="background:#f44336"></span>Enemy</span>
                <span class="legend-item"><span class="legend-dot" style="background:#9e9e9e"></span>Neutral</span>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachTerritoryListeners(el);
}

function renderTerritoryCard(territory) {
    const isControlled = territory.controlled_by;
    const controlColor = !isControlled ? '#9e9e9e' : isControlled.toLowerCase().includes('you') ? '#4caf50' : '#f44336';

    return `
        <div class="territory-card" data-region="${territory.name}" style="--control-color:${controlColor}">
            <div class="territory-card-header">
                <span class="territory-name">${escapeHtml(territory.name)}</span>
                <span class="territory-districts">${(territory.districts || 0)} districts</span>
            </div>
            <div class="territory-card-body">
                <span class="territory-controller">Controller: ${escapeHtml(isControlled || 'Neutral')}</span>
            </div>
            <button class="vbt-btn vbt-btn-primary territory-contest-btn" ${isControlled ? '' : ''}>
                ${isControlled ? 'Manage' : 'Contest'}
            </button>
        </div>
    `;
}

function renderSampleTerritories() {
    const samples = [
        { name: 'Governor', districts: 5, controlled_by: 'You' },
        { name: 'North District', districts: 3, controlled_by: 'Club A' },
        { name: 'South District', districts: 4, controlled_by: null },
        { name: 'East District', districts: 2, controlled_by: 'Club B' },
    ];
    return samples.map(t => renderTerritoryCard(t)).join('');
}

function attachTerritoryListeners(el) {
    el.querySelectorAll('.territory-contest-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const card = btn.closest('.territory-card');
            const region = card?.dataset.region;
            if (window.showToast) window.showToast(`Contesting ${region}...`, 'info');
        });
    });
}

// --- Global handlers ---
window.initTerritoryMap = initTerritoryMap;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}
