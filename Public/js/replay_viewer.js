// ============================================================================
// replay_viewer.js — Battle Replay & Spectate Viewer
// ----------------------------------------------------------------------------
// Exposes replay_engine.go: frame-by-frame replay, playback controls
// Unique visual style: film player / theater screen with frame scrubber
// ============================================================================

var API_BASE = '/api';

// --- State ---
let replayData = null;
let currentFrame = 0;
let isPlaying = false;

// --- Init ---
export function initReplayViewer() {
    const el = document.getElementById('wd-replay');
    if (!el) return;
    renderReplayLoading(el);
    fetchReplayData(el);
}

async function fetchReplayData(el) {
    try {
        const resp = await fetch(`${API_BASE}/replay/latest`);
        if (!resp.ok) throw new Error('Failed to fetch replay data');
        replayData = await resp.json();
        renderReplayViewer(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">No replays available</p>';
    }
}

function renderReplayLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading replay…</div>';
}

function renderReplayViewer(el) {
    const frames = replayData?.frames || [];
    const matchId = replayData?.match_id || 'Unknown';
    const totalFrames = frames.length;

    let html = `
        <div class="replay-viewer-container">
            <!-- Theater Header -->
            <div class="replay-header">
                <div class="replay-icon">🎬</div>
                <div class="replay-info">
                    <h3 style="color:#e5e7eb;margin:0;">Replay Theater</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Match: ${escapeHtml(matchId.slice(0, 16))}</p>
                </div>
                <div class="replay-controls-top">
                    <button class="replay-btn" id="replay-prev">⏮</button>
                    <button class="replay-btn" id="replay-play">${isPlaying ? '⏸' : '▶'}</button>
                    <button class="replay-btn" id="replay-next">⏭</button>
                </div>
            </div>

            <!-- Theater Screen -->
            <div class="replay-screen">
                <div class="replay-frame" id="replay-frame">
                    ${totalFrames > 0 ? `
                        <div class="replay-frame-content">
                            <span class="replay-frame-number">Frame ${currentFrame + 1} / ${totalFrames}</span>
                            <div class="replay-board">
                                ${frames[currentFrame]?.board || 'No board data'}
                            </div>
                        </div>
                    ` : `
                        <div class="replay-empty-screen">
                            <span class="replay-empty-icon">📽</span>
                            <p>No replay frames recorded</p>
                        </div>
                    `}
                </div>
            </div>

            <!-- Frame Scrubber -->
            <div class="replay-scrubber-section">
                <h4 style="color:#e5e7eb;">🎞️ Frame Scrubber</h4>
                <div class="replay-scrubber">
                    <input type="range" id="replay-slider" min="0" max="${Math.max(0, totalFrames - 1)}" value="${currentFrame}" />
                    <div class="replay-scrubber-labels">
                        <span>Frame 1</span>
                        <span id="replay-current-label">Frame ${currentFrame + 1}</span>
                        <span>Frame ${totalFrames}</span>
                    </div>
                </div>
            </div>

            <!-- Replay Info -->
            <div class="replay-info-section">
                <h4 style="color:#e5e7eb;">📋 Match Info</h4>
                <div class="replay-info-grid">
                    <div class="replay-info-card">
                        <span class="replay-info-label">Total Frames</span>
                        <span class="replay-info-value">${totalFrames}</span>
                    </div>
                    <div class="replay-info-card">
                        <span class="replay-info-label">Match ID</span>
                        <span class="replay-info-value">${escapeHtml(matchId.slice(0, 8))}</span>
                    </div>
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachReplayListeners(el);
}

function attachReplayListeners(el) {
    // Play button
    const playBtn = el.querySelector('#replay-play');
    if (playBtn) {
        playBtn.addEventListener('click', () => {
            isPlaying = !isPlaying;
            playBtn.textContent = isPlaying ? '⏸' : '▶';
        });
    }

    // Prev button
    const prevBtn = el.querySelector('#replay-prev');
    if (prevBtn) {
        prevBtn.addEventListener('click', () => {
            currentFrame = Math.max(0, currentFrame - 1);
            updateReplayFrame(el);
        });
    }

    // Next button
    const nextBtn = el.querySelector('#replay-next');
    if (nextBtn) {
        nextBtn.addEventListener('click', () => {
            const frames = replayData?.frames || [];
            currentFrame = Math.min(frames.length - 1, currentFrame + 1);
            updateReplayFrame(el);
        });
    }

    // Slider
    const slider = el.querySelector('#replay-slider');
    if (slider) {
        slider.addEventListener('input', () => {
            currentFrame = parseInt(slider.value);
            updateReplayFrame(el);
        });
    }
}

function updateReplayFrame(el) {
    const frames = replayData?.frames || [];
    const frameContent = el.querySelector('.replay-frame-number');
    const currentLabel = el.querySelector('#replay-current-label');
    const slider = el.querySelector('#replay-slider');

    if (frameContent) frameContent.textContent = `Frame ${currentFrame + 1} / ${frames.length}`;
    if (currentLabel) currentLabel.textContent = `Frame ${currentFrame + 1}`;
    if (slider) slider.value = currentFrame;
}

// --- Global handlers ---
window.initReplayViewer = initReplayViewer;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}
