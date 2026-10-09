// ============================================================================
// Quick batch: Remaining 15 tabs with unique mini-themes
// Each gets a distinct visual identity even if compact
// ============================================================================

// --- LAUNCHPAD ---
export function initLaunchpad() {
    const el = document.getElementById('wd-launches');
    if (!el) return;
    el.innerHTML = `
        <div class="launchpad-container">
            <div class="launchpad-header">
                <span class="launchpad-icon">🚀</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Launchpad</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Track and launch new projects</p>
                </div>
            </div>
            <div class="launchpad-grid">
                <div class="launch-card">
                    <span class="launch-icon">🌟</span>
                    <span class="launch-name">Season 2 Update</span>
                    <span class="launch-status status-active">Active</span>
                </div>
            </div>
        </div>
    `;
}

// --- ADS ---
export function initAds() {
    const el = document.getElementById('wd-ads');
    if (!el) return;
    el.innerHTML = `
        <div class="ads-container">
            <div class="ads-header">
                <span class="ads-icon">📢</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Ad Center</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Create and manage ad campaigns</p>
                </div>
            </div>
            <div class="ads-grid">
                <div class="ad-card">
                    <span class="ad-icon">📺</span>
                    <span class="ad-name">Arena Promo</span>
                    <span class="ad-status">Running</span>
                </div>
            </div>
        </div>
    `;
}

// --- VEHICLES ---
export function initVehicles() {
    const el = document.getElementById('wd-vehicles');
    if (!el) return;
    el.innerHTML = `
        <div class="vehicles-container">
            <div class="vehicles-header">
                <span class="vehicles-icon">🚗</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Vehicle Bay</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Deploy vehicles to regions</p>
                </div>
            </div>
            <div class="vehicles-grid">
                <div class="vehicle-card">
                    <span class="vehicle-icon">🏎️</span>
                    <span class="vehicle-name">Speedster</span>
                    <span class="vehicle-status">Idle</span>
                </div>
            </div>
        </div>
    `;
}

// --- STATS ---
export function initStatsOverlay() {
    const el = document.getElementById('wd-stats');
    if (!el) return;
    el.innerHTML = `
        <div class="stats-overlay-container">
            <div class="stats-header">
                <span class="stats-icon">📊</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Power Stats</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Your combat statistics</p>
                </div>
            </div>
            <div class="stats-grid">
                <div class="stat-card"><span class="stat-label">Power Level</span><span class="stat-val">0</span></div>
                <div class="stat-card"><span class="stat-label">Effective</span><span class="stat-val">0</span></div>
                <div class="stat-card"><span class="stat-label">Stat Sum</span><span class="stat-val">0</span></div>
            </div>
        </div>
    `;
}

// --- WORLD CONTENT ---
export function initWorldContent() {
    const el = document.getElementById('wd-worldcontent');
    if (!el) return;
    el.innerHTML = `
        <div class="worldcontent-container">
            <div class="worldcontent-header">
                <span class="worldcontent-icon">🌍</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">World Content</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Deploy entities to regions</p>
                </div>
            </div>
            <div class="worldcontent-grid">
                <div class="content-card">
                    <span class="content-icon">🏰</span>
                    <span class="content-name">Castle</span>
                    <span class="content-status">Active</span>
                </div>
            </div>
        </div>
    `;
}

// --- GAMING OS ---
export function initGamingOS() {
    const el = document.getElementById('wd-gamingos');
    if (!el) return;
    el.innerHTML = `
        <div class="gamingos-container">
            <div class="gamingos-header">
                <span class="gamingos-icon">🖥️</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Gaming OS</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">System modules and leases</p>
                </div>
            </div>
            <div class="gamingos-grid">
                <div class="module-card">
                    <span class="module-icon">⚙️</span>
                    <span class="module-name">Economy</span>
                    <span class="module-status">Active</span>
                </div>
            </div>
        </div>
    `;
}

// --- LOCAL LLM ---
export function initLocalModel() {
    const el = document.getElementById('wd-localmodel');
    if (!el) return;
    el.innerHTML = `
        <div class="localmodel-container">
            <div class="localmodel-header">
                <span class="localmodel-icon">🤖</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Local LLM</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Bot-child inference</p>
                </div>
            </div>
            <div class="localmodel-grid">
                <div class="model-card">
                    <span class="model-tier">Base</span>
                    <span class="model-instances">0 instances</span>
                    <span class="model-backend">GPU</span>
                </div>
            </div>
        </div>
    `;
}

// --- GOVERNOR ---
export function initGovernor() {
    const el = document.getElementById('wd-governor');
    if (!el) return;
    el.innerHTML = `
        <div class="governor-container">
            <div class="governor-header">
                <span class="governor-icon">👑</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Governor</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Regional control</p>
                </div>
            </div>
            <div class="governor-stats">
                <div class="gov-stat"><span class="gov-stat-label">Voting Power</span><span class="gov-stat-val">0</span></div>
                <div class="gov-stat"><span class="gov-stat-label">Delegated</span><span class="gov-stat-val">None</span></div>
            </div>
        </div>
    `;
}

// --- LEADERBOARD ---
export function initLeaderboard() {
    const el = document.getElementById('wd-leaderboard');
    if (!el) return;
    el.innerHTML = `
        <div class="leaderboard-container">
            <div class="leaderboard-header">
                <span class="leaderboard-icon">🏆</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Leaderboard</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Top players</p>
                </div>
            </div>
            <div class="leaderboard-list">
                <div class="leaderboard-entry"><span class="lb-rank">#1</span><span class="lb-name">Player</span><span class="lb-score">0</span></div>
            </div>
        </div>
    `;
}

// --- COMPLIANCE ---
// This leaf used to be FABRICATED: it rendered a hard-coded "KYC / Clear" row and made no request
// at all, so a player saw a compliance record the engine had never produced. It now reads the
// engine's REAL records and exposes the two lifecycle controls that had no owner anywhere in the
// client — POST /api/compliance/escalate and POST /api/compliance/resolve (ComplianceEngine
// .EscalateRecord / .ResolveRecord). The summary above them is the server's own counts.
var COMPLIANCE_API_BASE = '/api';

async function complianceApi(path, opts) {
    const url = path.startsWith('/api/') ? path : COMPLIANCE_API_BASE + path;
    const resp = await fetch(url, opts);
    if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch (e) {} throw new Error(d || ('HTTP ' + resp.status)); }
    return resp.json();
}

function complianceEsc(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

export function initCompliance() {
    const el = document.getElementById('wd-compliance');
    if (!el) return;
    el.innerHTML = '<div class="wd-loading">Loading compliance records…</div>';
    complianceLoad(el);
}

async function complianceLoad(el) {
    let summary = null, records = [];
    const problems = [];
    try {
        const res = await complianceApi('/compliance/summary');
        summary = res.summary || null;
    } catch (e) { problems.push('summary: ' + e.message); }
    try {
        const res = await complianceApi('/compliance/records');
        records = Array.isArray(res.records) ? res.records : [];
    } catch (e) { problems.push('records: ' + e.message); }

    const summaryHtml = summary
        ? Object.entries(summary).map(([k, v]) => `
            <div class="compliance-stat"><span class="compliance-stat-label">${complianceEsc(k)}</span><span class="compliance-stat-val">${complianceEsc(typeof v === 'object' ? JSON.stringify(v) : v)}</span></div>
        `).join('')
        : '<p class="compliance-note">The engine did not report a compliance summary.</p>';

    const rows = records.length
        ? records.map(r => `
            <div class="compliance-record" data-record="${complianceEsc(r.id)}">
                <span class="record-type">${complianceEsc(r.record_type || '—')}</span>
                <span class="record-desc">${complianceEsc(r.description || '')}</span>
                <span class="record-sev sev-${complianceEsc(String(r.severity || 'low').toLowerCase())}">${complianceEsc(r.severity || '—')}</span>
                <span class="record-status">${complianceEsc(r.status || '—')}</span>
                <button class="vbt-btn vbt-btn-secondary compliance-escalate" data-record="${complianceEsc(r.id)}">Escalate</button>
                <button class="vbt-btn vbt-btn-primary compliance-resolve" data-record="${complianceEsc(r.id)}">Resolve</button>
            </div>`).join('')
        : '<p class="compliance-note">The engine holds no compliance records.</p>';

    el.innerHTML = `
        <div class="compliance-container">
            <div class="compliance-header">
                <span class="compliance-icon">📋</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Compliance</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Engine records · escalate · resolve</p>
                </div>
            </div>
            <div class="compliance-summary-grid">${summaryHtml}</div>
            <div class="compliance-list">${rows}</div>
            <p class="compliance-note" id="compliance-status">${problems.length ? 'Could not read — ' + complianceEsc(problems.join('; ')) : ''}</p>
        </div>
    `;

    const act = async (recordId, verb, path) => {
        const status = document.getElementById('compliance-status');
        try {
            // The body carries ONLY record_id, which is what the handler decodes.
            await complianceApi(path, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ record_id: recordId }),
            });
            if (status) status.textContent = 'Record ' + recordId + ' ' + verb + '.';
            complianceLoad(el);   // re-read, so the shown status is the engine's, not ours
        } catch (e) {
            if (status) status.textContent = verb.charAt(0).toUpperCase() + verb.slice(1) + ' refused: ' + e.message;
        }
    };
    el.querySelectorAll('.compliance-escalate').forEach(b => b.addEventListener('click', () => act(b.dataset.record, 'escalated', '/compliance/escalate')));
    el.querySelectorAll('.compliance-resolve').forEach(b => b.addEventListener('click', () => act(b.dataset.record, 'resolved', '/compliance/resolve')));
}

// --- MAINTENANCE ---
export function initMaintenance() {
    const el = document.getElementById('wd-maintenance');
    if (!el) return;
    el.innerHTML = `
        <div class="maintenance-container">
            <div class="maintenance-header">
                <span class="maintenance-icon">🔧</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Maintenance</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">System status</p>
                </div>
            </div>
            <div class="maintenance-status">
                <span class="status-indicator active"></span>
                <span class="status-text">System Active</span>
            </div>
        </div>
    `;
}

// --- SYSTEM MSG ---
export function initSystemMsg() {
    const el = document.getElementById('wd-systemmsg');
    if (!el) return;
    el.innerHTML = `
        <div class="systemmsg-container">
            <div class="systemmsg-header">
                <span class="systemmsg-icon">📢</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">System Messages</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Announcements</p>
                </div>
            </div>
            <div class="systemmsg-list">
                <div class="msg-entry"><span class="msg-text">Welcome to the Arena</span><span class="msg-time">Now</span></div>
            </div>
        </div>
    `;
}

// --- ORPHANS ---
export function initOrphans() {
    const el = document.getElementById('wd-orphan');
    if (!el) return;
    el.innerHTML = `
        <div class="orphans-container">
            <div class="orphans-header">
                <span class="orphans-icon">🐾</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Orphan Welfare</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Adopt or reclaim orphaned entities</p>
                </div>
            </div>
            <div class="orphans-actions">
                <button class="vbt-btn vbt-btn-primary">Adopt</button>
                <button class="vbt-btn vbt-btn-secondary">Reclaim</button>
            </div>
        </div>
    `;
}

// --- REGIONS ---
export function initRegions() {
    const el = document.getElementById('wd-regions');
    if (!el) return;
    el.innerHTML = `
        <div class="regions-container">
            <div class="regions-header">
                <span class="regions-icon">🗺️</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Regions</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Region status and migration</p>
                </div>
            </div>
            <div class="regions-list">
                <div class="region-entry"><span class="region-name">Governor</span><span class="region-status">Neutral</span></div>
            </div>
        </div>
    `;
}

// Expose all
window.initLaunchpad = initLaunchpad;
window.initAds = initAds;
window.initVehicles = initVehicles;
window.initStatsOverlay = initStatsOverlay;
window.initWorldContent = initWorldContent;
window.initGamingOS = initGamingOS;
window.initLocalModel = initLocalModel;
window.initGovernor = initGovernor;
window.initLeaderboard = initLeaderboard;
window.initCompliance = initCompliance;
window.initMaintenance = initMaintenance;
window.initSystemMsg = initSystemMsg;
window.initOrphans = initOrphans;
window.initRegions = initRegions;
