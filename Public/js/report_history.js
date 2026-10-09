// ============================================================================
// report_history.js — Player Reporting & Moderation History
// ----------------------------------------------------------------------------
// Exposes report_player.go: file report, status tracking, admin response
// Unique visual style: file cabinet / case management theme
// ============================================================================

var API_BASE = '/api';

// --- Report Status ---
const REPORT_STATUS = {
    pending: { color: '#ff9800', label: 'Pending', icon: '⏳' },
    investigating: { color: '#2196f3', label: 'Investigating', icon: '🔍' },
    resolved: { color: '#4caf50', label: 'Resolved', icon: '✅' },
    dismissed: { color: '#9e9e9e', label: 'Dismissed', icon: '❌' },
};

// --- State ---
let reports = [];
let reportHistory = [];

// --- Init ---
export function initReportHistory() {
    const el = document.getElementById('wd-report');
    if (!el) return;
    renderReportLoading(el);
    fetchReportHistory(el);
}

async function fetchReportHistory(el) {
    try {
        const resp = await fetch(`${API_BASE}/compliance/records`);
        if (!resp.ok) throw new Error('Failed to fetch report history');
        const data = await resp.json();
        reports = data.records || [];
        renderReportHistory(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Report history unavailable</p>';
    }
}

function renderReportLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading case files…</div>';
}

function renderReportHistory(el) {
    let html = `
        <div class="report-history-container">
            <!-- File Cabinet Header -->
            <div class="report-header">
                <div class="report-header-icon">📋</div>
                <div class="report-header-info">
                    <h3 style="color:#e5e7eb;margin:0;">Case Files</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Track your reports and moderation history</p>
                </div>
                <div class="report-stats">
                    <div class="report-stat">
                        <span class="report-stat-num">${reports.length}</span>
                        <span class="report-stat-label">Total</span>
                    </div>
                    <div class="report-stat">
                        <span class="report-stat-num">${reports.filter(r => r.status === 'pending').length}</span>
                        <span class="report-stat-label">Pending</span>
                    </div>
                </div>
            </div>

            <!-- New Report Form -->
            <div class="report-new-section">
                <h4 style="color:#e5e7eb;">📝 File New Report</h4>
                <div class="report-form">
                    <div class="report-form-row">
                        <label>Player Wallet</label>
                        <input type="text" id="report-wallet-input" placeholder="0x..." />
                    </div>
                    <div class="report-form-row">
                        <label>Reason</label>
                        <select id="report-reason-select">
                            <option value="cheating">Cheating</option>
                            <option value="harassment">Harassment</option>
                            <option value="scam">Scam</option>
                            <option value="exploit">Exploit</option>
                            <option value="other">Other</option>
                        </select>
                    </div>
                    <div class="report-form-row">
                        <label>Details</label>
                        <textarea id="report-details-input" placeholder="Describe the incident..." rows="3"></textarea>
                    </div>
                    <button class="vbt-btn vbt-btn-primary" id="report-submit-btn">📤 Submit Report</button>
                </div>
            </div>

            <!-- Report History -->
            <div class="report-list-section">
                <h4 style="color:#e5e7eb;">📜 Report History</h4>
                <div class="report-list">
                    ${reports.length > 0 ? reports.map(r => renderReportCard(r)).join('') : renderEmptyReports()}
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachReportListeners(el);
}

function renderReportCard(report) {
    const status = REPORT_STATUS[report.status] || REPORT_STATUS.pending;
    return `
        <div class="report-card status-${report.status}" style="--status-color:${status.color}">
            <div class="report-card-header">
                <span class="report-card-id">#${report.id || report.case_id || '0000'}</span>
                <span class="report-card-status" style="background:${status.color}22;color:${status.color}">
                    ${status.icon} ${status.label}
                </span>
            </div>
            <div class="report-card-body">
                <span class="report-card-target">🎯 ${escapeHtml(report.target_wallet || report.wallet || 'Unknown')}</span>
                <span class="report-card-reason">${escapeHtml(report.reason || report.type || 'Unknown')}</span>
            </div>
            ${report.admin_response ? `
                <div class="report-card-response">
                    <span class="response-label">Admin:</span>
                    <span class="response-text">${escapeHtml(report.admin_response)}</span>
                </div>
            ` : ''}
            <div class="report-card-footer">
                <span class="report-card-date">${report.created_at || report.timestamp || 'Unknown'}</span>
            </div>
        </div>
    `;
}

function renderEmptyReports() {
    return `
        <div class="report-empty">
            <span class="report-empty-icon">📂</span>
            <p>No reports filed. Your case folder is empty.</p>
        </div>
    `;
}

function attachReportListeners(el) {
    // Submit report
    const submitBtn = el.querySelector('#report-submit-btn');
    if (submitBtn) {
        submitBtn.addEventListener('click', () => {
            const wallet = el.querySelector('#report-wallet-input')?.value.trim();
            const reason = el.querySelector('#report-reason-select')?.value;
            const details = el.querySelector('#report-details-input')?.value.trim();
            if (!wallet) {
                alert('Please enter a wallet address');
                return;
            }
            submitReport(wallet, reason, details);
        });
    }
}

async function submitReport(wallet, reason, details) {
    try {
        const resp = await fetch(`${API_BASE}/report-player`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ wallet, reason, details })
        });
        if (!resp.ok) throw new Error('Failed to submit report');
        if (window.showToast) window.showToast('📤 Report submitted!', 'success');
    } catch (e) {
        if (window.showToast) window.showToast(`❌ ${e.message}`, 'error');
    }
}

// --- Global handlers ---
window.initReportHistory = initReportHistory;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}
