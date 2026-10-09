// ============================================================================
// counterfeit_scanner.js — Counterfeit Detection & Forensic Lab
// ----------------------------------------------------------------------------
// Exposes counterfeit_service.go: detect, generate, scan history
// Unique visual style: forensic lab with scan lines and UV light effects
// ============================================================================

var API_BASE = '/api';

// --- State ---
let scanHistory = [];
let detectedCount = 0;

// --- Init ---
export function initCounterfeitScanner() {
    const el = document.getElementById('wd-counterfeit');
    if (!el) return;
    renderCounterfeitLoading(el);
    fetchCounterfeitData(el);
}

async function fetchCounterfeitData(el) {
    try {
        const resp = await fetch(`${API_BASE}/counterfeit/detect`);
        if (!resp.ok) throw new Error('Failed to fetch counterfeit data');
        const data = await resp.json();
        detectedCount = data.detected || 0;
        scanHistory = data.history || [];
        renderCounterfeitScanner(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Scanner data unavailable</p>';
    }
}

function renderCounterfeitLoading(el) {
    el.innerHTML = '<div class="wd-loading">Initializing scanner…</div>';
}

function renderCounterfeitScanner(el) {
    let html = `
        <div class="counterfeit-scanner-container">
            <!-- Lab Header -->
            <div class="scanner-header">
                <div class="scanner-icon">🔬</div>
                <div class="scanner-info">
                    <h3 style="color:#e5e7eb;margin:0;">Forensic Lab</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Detect counterfeit cards and items</p>
                </div>
                <div class="scanner-detected">
                    <span class="scanner-detected-num">${detectedCount}</span>
                    <span class="scanner-detected-label">Detected</span>
                </div>
            </div>

            <!-- Scan Line Effect -->
            <div class="scanner-display">
                <div class="scanner-scanline"></div>
                <div class="scanner-screen">
                    <span class="scanner-screen-text">PLACE CARD IN SCANNER</span>
                </div>
            </div>

            <!-- Scan Button -->
            <div class="scanner-actions">
                <button class="vbt-btn vbt-btn-primary" id="scanner-scan-btn">
                    🔍 Scan Card
                </button>
                <button class="vbt-btn vbt-btn-secondary" id="scanner-generate-btn">
                    ⚠ Generate Counterfeit
                </button>
            </div>

            <!-- Scan History -->
            <div class="scanner-history-section">
                <h4 style="color:#e5e7eb;">📜 Scan History</h4>
                <div class="scanner-history-list">
                    ${scanHistory.length > 0 ? scanHistory.map(s => `
                        <div class="scanner-history-item ${s.authentic ? 'authentic' : 'counterfeit'}">
                            <span class="scanner-history-icon">${s.authentic ? '✓' : '✗'}</span>
                            <span class="scanner-history-name">${escapeHtml(s.card_id || 'Unknown')}</span>
                            <span class="scanner-history-result">${s.authentic ? 'Authentic' : 'COUNTERFEIT'}</span>
                        </div>
                    `).join('') : '<p style="color:#90a4ae;font-size:11px;">No scans recorded</p>'}
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachCounterfeitListeners(el);
}

function attachCounterfeitListeners(el) {
    const scanBtn = el.querySelector('#scanner-scan-btn');
    if (scanBtn) {
        scanBtn.addEventListener('click', () => {
            if (window.showToast) window.showToast('🔍 Scanning...', 'info');
        });
    }

    const genBtn = el.querySelector('#scanner-generate-btn');
    if (genBtn) {
        genBtn.addEventListener('click', () => {
            if (window.showToast) window.showToast('⚠ Counterfeit generation requires on-chain transaction', 'error');
        });
    }
}

// --- Global handlers ---
window.initCounterfeitScanner = initCounterfeitScanner;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}
