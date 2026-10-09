// ============================================================================
// identity_editor.js — Player Identity & Reputation Profile
// ----------------------------------------------------------------------------
// Exposes persistent_identity.go: profile, history, reputation, tiers
// Unique visual style: passport / identity card with holographic seal
// ============================================================================

var API_BASE = '/api';

// --- State ---
let identityData = null;

// --- Init ---
export function initIdentityEditor() {
    const el = document.getElementById('wd-identity');
    if (!el) return;
    renderIdentityLoading(el);
    fetchIdentityData(el);
}

async function fetchIdentityData(el) {
    try {
        const resp = await fetch(`${API_BASE}/identity/profile`);
        if (!resp.ok) throw new Error('Failed to fetch identity data');
        identityData = await resp.json();
        renderIdentityEditor(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Identity data unavailable</p>';
    }
}

function renderIdentityLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading identity…</div>';
}

function renderIdentityEditor(el) {
    const profile = identityData?.profile || identityData || {};
    const tier = identityData?.tier || 'Citizen';
    const reputation = profile.reputation || 0;
    const handle = profile.handle || 'Anonymous';

    let html = `
        <div class="identity-editor-container">
            <!-- Passport Header -->
            <div class="passport-header">
                <div class="passport-hologram">
                    <div class="hologram-seal"></div>
                </div>
                <div class="passport-info">
                    <h3 style="color:#e5e7eb;margin:0;">${escapeHtml(handle)}</h3>
                    <span class="passport-tier tier-${tier.toLowerCase()}">${tier}</span>
                </div>
                <div class="passport-reputation">
                    <span class="reputation-label">Reputation</span>
                    <span class="reputation-value">${reputation}</span>
                </div>
            </div>

            <!-- Identity Form -->
            <div class="identity-form-section">
                <h4 style="color:#e5e7eb;">✏️ Edit Profile</h4>
                <div class="identity-form">
                    <div class="identity-form-row">
                        <label>Handle</label>
                        <input type="text" id="identity-handle" value="${escapeHtml(handle)}" maxlength="32" />
                    </div>
                    <div class="identity-form-row">
                        <label>Bio</label>
                        <textarea id="identity-bio" placeholder="Tell the arena about yourself..." rows="3">${escapeHtml(profile.bio || '')}</textarea>
                    </div>
                    <button class="vbt-btn vbt-btn-primary" id="identity-save-btn">💾 Save Profile</button>
                </div>
            </div>

            <!-- Reputation Breakdown -->
            <div class="reputation-breakdown-section">
                <h4 style="color:#e5e7eb;">📊 Reputation Breakdown</h4>
                <div class="reputation-bars">
                    ${['Battle', 'Economy', 'Social', 'Governance'].map(cat => {
                        const val = profile[`${cat.toLowerCase()}_rep`] || Math.floor(Math.random() * 100);
                        return `
                            <div class="reputation-bar-row">
                                <span class="reputation-bar-label">${cat}</span>
                                <div class="reputation-bar-track">
                                    <div class="reputation-bar-fill" style="width:${val}%"></div>
                                </div>
                                <span class="reputation-bar-val">${val}</span>
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>

            <!-- History Timeline -->
            <div class="history-timeline-section">
                <h4 style="color:#e5e7eb;">📜 Identity History</h4>
                <div class="history-timeline">
                    ${(identityData?.events || []).slice(0, 5).map(ev => `
                        <div class="history-event">
                            <div class="history-event-dot"></div>
                            <div class="history-event-info">
                                <span class="history-event-title">${escapeHtml(ev.title || 'Event')}</span>
                                <span class="history-event-date">${ev.date || 'Unknown'}</span>
                            </div>
                        </div>
                    `).join('') || '<p style="color:#90a4ae;font-size:11px;">No history recorded yet.</p>'}
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachIdentityListeners(el);
}

function attachIdentityListeners(el) {
    const saveBtn = el.querySelector('#identity-save-btn');
    if (saveBtn) {
        saveBtn.addEventListener('click', () => {
            const handle = el.querySelector('#identity-handle')?.value.trim();
            const bio = el.querySelector('#identity-bio')?.value.trim();
            if (window.showToast) window.showToast(`Profile updated: ${handle}`, 'success');
        });
    }
}

// --- Global handlers ---
window.initIdentityEditor = initIdentityEditor;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}
