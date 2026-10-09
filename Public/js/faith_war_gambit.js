// ============================================================================
// faith_war_gambit.js — Faith Coherence & War Gambit
// ----------------------------------------------------------------------------
// Exposes faith_church.go: coherence, rituals, war gambit, religious leader cards
// Unique visual style: cathedral interior with candlelight and stained glass
// ============================================================================

var API_BASE = '/api';

// --- State ---
let faithData = null;

// --- Init ---
export function initFaithWarGambit() {
    const el = document.getElementById('wd-faith');
    if (!el) return;
    renderFaithLoading(el);
    fetchFaithData(el);
}

async function fetchFaithData(el) {
    try {
        const resp = await fetch(`${API_BASE}/faith/coherence`);
        if (!resp.ok) throw new Error('Failed to fetch faith data');
        faithData = await resp.json();
        renderFaithWarGambit(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Faith data unavailable</p>';
    }
}

function renderFaithLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading cathedral…</div>';
}

function renderFaithWarGambit(el) {
    const coherence = faithData?.faith_coherence || 0;
    const score = faithData?.score || 0;

    let html = `
        <div class="faith-container">
            <!-- Cathedral Header -->
            <div class="faith-header">
                <div class="faith-candles">🕯️</div>
                <div class="faith-info">
                    <h3 style="color:#e5e7eb;margin:0;">Faith & Devotion</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Your spiritual influence</p>
                </div>
                <div class="faith-coherence">
                    <span class="faith-coherence-label">Coherence</span>
                    <span class="faith-coherence-value" style="color:${coherence > 50 ? '#4caf50' : coherence > 20 ? '#ff9800' : '#f44336'}">${coherence}</span>
                </div>
            </div>

            <!-- Coherence Bar -->
            <div class="faith-coherence-section">
                <h4 style="color:#e5e7eb;">⛪ Faith Coherence</h4>
                <div class="faith-coherence-bar">
                    <div class="faith-coherence-fill" style="width:${Math.min(100, coherence / 10)}%"></div>
                </div>
                <span class="faith-coherence-text">Score: ${score}</span>
            </div>

            <!-- Rituals -->
            <div class="faith-rituals-section">
                <h4 style="color:#e5e7eb;">🕯️ Rituals</h4>
                <div class="faith-rituals-grid">
                    <div class="ritual-card">
                        <span class="ritual-icon">🙏</span>
                        <span class="ritual-name">Prayer</span>
                        <span class="ritual-effect">+5 Coherence</span>
                        <button class="vbt-btn vbt-btn-primary ritual-perform-btn">Perform</button>
                    </div>
                    <div class="ritual-card">
                        <span class="ritual-icon">📿</span>
                        <span class="ritual-name">Meditation</span>
                        <span class="ritual-effect">+10 Coherence</span>
                        <button class="vbt-btn vbt-btn-primary ritual-perform-btn">Perform</button>
                    </div>
                </div>
            </div>

            <!-- War Gambit -->
            <div class="faith-gambit-section">
                <h4 style="color:#e5e7eb;">⚔️ Faith War Gambit</h4>
                <div class="gambit-card">
                    <p style="color:#90a4ae;font-size:11px;">Stake your favorite card in a faith battle</p>
                    <button class="vbt-btn vbt-btn-primary gambit-btn">⚔️ Enter Gambit</button>
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachFaithListeners(el);
}

function attachFaithListeners(el) {
    el.querySelectorAll('.ritual-perform-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            if (window.showToast) window.showToast('🕯️ Ritual performed!', 'success');
        });
    });

    const gambitBtn = el.querySelector('.gambit-btn');
    if (gambitBtn) {
        gambitBtn.addEventListener('click', () => {
            if (window.showToast) window.showToast('⚔️ Entering Faith War Gambit...', 'info');
        });
    }
}

// --- Global handlers ---
window.initFaithWarGambit = initFaithWarGambit;
