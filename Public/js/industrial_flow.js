// ============================================================================
// industrial_flow.js — Industrial Loop Flow Visualization
// ----------------------------------------------------------------------------
// Exposes industrial_loop.go: metrics, health, flow phases
// Unique visual style: industrial machinery / pipeline theme with animated flow
// ============================================================================

var API_BASE = '/api';

// --- Loop Phases ---
const LOOP_PHASES = [
    { id: 'activity', name: 'Activity', icon: '⚡', color: '#f44336', desc: 'Battle rewards, tournaments' },
    { id: 'business', name: 'Business', icon: '🏢', color: '#ff9800', desc: 'Shops, creator store' },
    { id: 'employment', name: 'Employment', icon: '👥', color: '#ffc107', desc: 'Hire & salary' },
    { id: 'purchasing', name: 'Purchasing', icon: '🛒', color: '#4caf50', desc: 'Buy items & services' },
    { id: 'taxation', name: 'Taxation', icon: '💰', color: '#2196f3', desc: 'Auto-tax routing' },
    { id: 'treasury', name: 'Treasury', icon: '🏛️', color: '#9c27b0', desc: 'Club treasuries' },
    { id: 'development', name: 'Development', icon: '🏗️', color: '#00bcd4', desc: 'Regional projects' },
    { id: 'events', name: 'Events', icon: '🎉', color: '#e91e63', desc: 'Seasonal events' },
];

// --- State ---
let industrialMetrics = null;
let loopHealth = 0;

// --- Init ---
export function initIndustrialFlow() {
    const el = document.getElementById('wd-industrial');
    if (!el) return;
    renderIndustrialLoading(el);
    fetchIndustrialData(el);
}

async function fetchIndustrialData(el) {
    try {
        const resp = await fetch(`${API_BASE}/industrial-loop/metrics`);
        if (!resp.ok) throw new Error('Failed to fetch industrial data');
        industrialMetrics = await resp.json();
        loopHealth = calculateLoopHealth(industrialMetrics);
        renderIndustrialFlow(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Industrial data unavailable</p>';
    }
}

function renderIndustrialLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading industrial flow…</div>';
}

function renderIndustrialFlow(el) {
    const metrics = industrialMetrics || {};

    let html = `
        <div class="industrial-flow-container">
            <!-- Industrial Header -->
            <div class="industrial-header">
                <div class="industrial-gear-icon">⚙️</div>
                <div class="industrial-header-info">
                    <h3 style="color:#e5e7eb;margin:0;">Industrial Loop</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Perpetual circulation engine</p>
                </div>
                <div class="industrial-health">
                    <span class="industrial-health-label">Health</span>
                    <span class="industrial-health-value" style="color:${loopHealth > 70 ? '#4caf50' : loopHealth > 40 ? '#ff9800' : '#f44336'}">${loopHealth}%</span>
                </div>
            </div>

            <!-- Flow Pipeline -->
            <div class="industrial-pipeline">
                <h4 style="color:#e5e7eb;">🔄 Flow Pipeline</h4>
                <div class="pipeline-scroll">
                    <div class="pipeline-flow">
                        ${LOOP_PHASES.map((phase, i) => renderPipelinePhase(phase, i)).join('')}
                    </div>
                </div>
            </div>

            <!-- Metrics Grid -->
            <div class="industrial-metrics-section">
                <h4 style="color:#e5e7eb;">📊 Metrics</h4>
                <div class="industrial-metrics-grid">
                    <div class="industrial-metric-card">
                        <span class="metric-value">${metrics.business_count || 0}</span>
                        <span class="metric-label">Businesses</span>
                    </div>
                    <div class="industrial-metric-card">
                        <span class="metric-value">${metrics.employee_count || 0}</span>
                        <span class="metric-label">Employees</span>
                    </div>
                    <div class="industrial-metric-card">
                        <span class="metric-value">${formatVBV(metrics.treasury_micro || 0)}</span>
                        <span class="metric-label">Treasury</span>
                    </div>
                    <div class="industrial-metric-card">
                        <span class="metric-value">${formatVBV(metrics.circulation_micro || 0)}</span>
                        <span class="metric-label">Circulated</span>
                    </div>
                </div>
            </div>

            <!-- Phase Details -->
            <div class="industrial-phases-section">
                <h4 style="color:#e5e7eb;">📋 Phase Breakdown</h4>
                <div class="industrial-phases-list">
                    ${LOOP_PHASES.map(phase => renderPhaseDetail(phase, metrics)).join('')}
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
}

function renderPipelinePhase(phase, index) {
    const isActive = index <= Math.floor(loopHealth / 12.5);
    return `
        <div class="pipeline-phase ${isActive ? 'phase-active' : 'phase-inactive'}" style="--phase-color:${phase.color}">
            <div class="pipeline-phase-icon">${phase.icon}</div>
            <div class="pipeline-phase-name">${phase.name}</div>
            ${index < LOOP_PHASES.length - 1 ? '<div class="pipeline-arrow">→</div>' : ''}
        </div>
    `;
}

function renderPhaseDetail(phase, metrics) {
    const volume = metrics[`${phase.id}_micro`] || 0;
    return `
        <div class="phase-detail-card" style="--phase-color:${phase.color}">
            <div class="phase-detail-icon">${phase.icon}</div>
            <div class="phase-detail-info">
                <span class="phase-detail-name">${phase.name}</span>
                <span class="phase-detail-desc">${phase.desc}</span>
            </div>
            <span class="phase-detail-volume">${formatVBV(volume)}</span>
        </div>
    `;
}

function calculateLoopHealth(metrics) {
    if (!metrics) return 0;
    const phases = ['activity', 'business', 'employment', 'purchasing', 'taxation', 'treasury', 'development', 'events'];
    const activePhases = phases.filter(p => (metrics[`${p}_micro`] || 0) > 0).length;
    return Math.round(activePhases / phases.length * 100);
}

// --- Global handlers ---
window.initIndustrialFlow = initIndustrialFlow;

// --- Utility ---
function formatVBV(micro) {
    const vbv = Number(micro) / 1_000_000;
    if (vbv >= 1_000_000) return `${(vbv / 1_000_000).toFixed(1)}M`;
    if (vbv >= 1_000) return `${(vbv / 1_000).toFixed(1)}K`;
    return vbv.toFixed(0);
}
