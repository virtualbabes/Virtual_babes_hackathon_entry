// ============================================================================
// theme_dashboard.js — Theme Engine Visualizer
// ----------------------------------------------------------------------------
// Wires orphaned backend routes:
//   - /api/theme/vector    → Tone, Element, Intensity, Entropy, MoodTag
//   - /api/market/weather  → Climate, DisasterTier per region
//   - /api/rivalry/world-dynamics → 10-signal world dynamics signature
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null, statusEl;
    let themeData = null, weatherData = null, dynamicsData = null;

    // Element definitions (matches backend Element enum)
    const ELEMENTS = [
        { id: 0, name: 'Fire', icon: '🔥', color: '#ff6b35' },
        { id: 1, name: 'Water', icon: '💧', color: '#00d4ff' },
        { id: 2, name: 'Earth', icon: '🌍', color: '#7cb342' },
        { id: 3, name: 'Air', icon: '💨', color: '#b0bec5' },
        { id: 4, name: 'Aether', icon: '✨', color: '#9c27b0' },
        { id: 5, name: 'Machine', icon: '⚙️', color: '#00bcd4' },
    ];

    // Tone definitions (matches backend Tone enum)
    const TONES = [
        { id: 0, name: 'Void', color: '#1a1a2e' },
        { id: 1, name: 'Malevolent', color: '#9c27b0' },
        { id: 2, name: 'Neutral', color: '#607d8b' },
        { id: 3, name: 'Benevolent', color: '#4caf50' },
        { id: 4, name: 'Order', color: '#2196f3' },
        { id: 5, name: 'Nature', color: '#8bc34a' },
        { id: 6, name: 'Machine', color: '#00bcd4' },
    ];

    function init() {
        if (overlayEl) return;
        const html = `
<div id="theme-dashboard-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel theme-dashboard-panel">
        <button id="btn-close-theme" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🌌 Theme Engine</h2>

        <div class="theme-tabs">
            <button class="theme-tab active" data-tab="vector" onclick="window.switchThemeTab('vector')">Theme Vector</button>
            <button class="theme-tab" data-tab="weather" onclick="window.switchThemeTab('weather')">Market Weather</button>
            <button class="theme-tab" data-tab="dynamics" onclick="window.switchThemeTab('dynamics')">World Dynamics</button>
        </div>

        <!-- THEME VECTOR TAB -->
        <div id="theme-panel-vector" class="theme-panel">
            <div class="theme-vector-display">
                <div class="theme-element-badge" id="theme-element">—</div>
                <div class="theme-tone-badge" id="theme-tone">—</div>
                <div class="theme-mood-tag" id="theme-mood">—</div>
            </div>
            <div class="theme-metrics-grid">
                <div class="theme-metric">
                    <span class="theme-metric-label">Intensity</span>
                    <div class="theme-metric-bar"><div class="theme-metric-fill" id="theme-intensity-bar" style="width:0%"></div></div>
                    <span class="theme-metric-value" id="theme-intensity">0</span>
                </div>
                <div class="theme-metric">
                    <span class="theme-metric-label">Entropy</span>
                    <div class="theme-metric-bar"><div class="theme-metric-fill entropy" id="theme-entropy-bar" style="width:0%"></div></div>
                    <span class="theme-metric-value" id="theme-entropy">0</span>
                </div>
            </div>
            <p class="theme-hint">Your ThemeVector shapes the world's mood. Tone drives particle behavior, Element drives color palette.</p>
        </div>

        <!-- MARKET WEATHER TAB -->
        <div id="theme-panel-weather" class="theme-panel hidden">
            <div class="weather-region-select">
                <label>Region:</label>
                <select id="weather-region-select" onchange="window.loadWeatherForRegion(this.value)">
                    <option value="">Select a region</option>
                </select>
            </div>
            <div class="weather-display" id="weather-display">
                <p style="color:#90a4ae;">Select a region to view weather.</p>
            </div>
        </div>

        <!-- WORLD DYNAMICS TAB -->
        <div id="theme-panel-dynamics" class="theme-panel hidden">
            <div class="dynamics-region-select">
                <label>Region:</label>
                <select id="dynamics-region-select" onchange="window.loadDynamicsForRegion(this.value)">
                    <option value="">Select a region</option>
                </select>
            </div>
            <div class="dynamics-display" id="dynamics-display">
                <p style="color:#90a4ae;">Select a region to view world dynamics.</p>
            </div>
        </div>

        <p id="theme-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('theme-dashboard-overlay');
        statusEl = document.getElementById('theme-status');
        document.getElementById('btn-close-theme').addEventListener('click', () => { overlayEl.style.display = 'none'; });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchThemeTab(tab) {
        document.querySelectorAll('.theme-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.theme-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('theme-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
    }

    // === THEME VECTOR ===
    async function loadThemeVector() {
        try {
            const res = await api('/api/theme/vector');
            if (!res.success) { setStatus(res.error || 'Failed to load theme vector'); return; }
            themeData = res;
            renderThemeVector(res);
        } catch (e) {
            setStatus('Theme vector unavailable: ' + e.message);
        }
    }

    function renderThemeVector(data) {
        const element = ELEMENTS[data.element] || ELEMENTS[0];
        const tone = TONES[data.tone] || TONES[0];

        const elEl = document.getElementById('theme-element');
        if (elEl) {
            elEl.innerHTML = `${element.icon} ${element.name}`;
            elEl.style.borderColor = element.color;
            elEl.style.boxShadow = `0 0 20px ${element.color}40`;
        }

        const toneEl = document.getElementById('theme-tone');
        if (toneEl) {
            toneEl.innerHTML = tone.name;
            toneEl.style.color = tone.color;
        }

        const moodEl = document.getElementById('theme-mood');
        if (moodEl) moodEl.innerHTML = data.mood_tag || '—';

        const intensityBar = document.getElementById('theme-intensity-bar');
        const intensityVal = document.getElementById('theme-intensity');
        if (intensityBar) intensityBar.style.width = (data.intensity || 0) + '%';
        if (intensityVal) intensityVal.textContent = data.intensity || 0;

        const entropyBar = document.getElementById('theme-entropy-bar');
        const entropyVal = document.getElementById('theme-entropy');
        if (entropyBar) entropyBar.style.width = (data.entropy || 0) + '%';
        if (entropyVal) entropyVal.textContent = data.entropy || 0;
    }

    // === MARKET WEATHER ===
    async function loadWeatherForRegion(region) {
        if (!region) return;
        try {
            const res = await api('/api/market/weather?region=' + encodeURIComponent(region));
            if (!res.success) { setStatus(res.error || 'Failed to load weather'); return; }
            weatherData = res;
            renderWeather(res);
        } catch (e) {
            setStatus('Weather unavailable: ' + e.message);
        }
    }

    function renderWeather(data) {
        const el = document.getElementById('weather-display');
        if (!el) return;
        const climateClass = (data.climate || 'UNKNOWN').toLowerCase();
        el.innerHTML = `
            <div class="weather-card ${climateClass}">
                <div class="weather-region">${esc(data.region)}</div>
                <div class="weather-climate">${esc(data.climate)}</div>
                <div class="weather-tier">Disaster Tier: ${data.disaster_tier || 0}</div>
            </div>`;
    }

    // === WORLD DYNAMICS ===
    async function loadDynamicsForRegion(region) {
        if (!region) return;
        try {
            const res = await api('/api/rivalry/world-dynamics?region=' + encodeURIComponent(region));
            if (!res.success) { setStatus(res.error || 'Failed to load dynamics'); return; }
            dynamicsData = res;
            renderDynamics(res);
        } catch (e) {
            setStatus('Dynamics unavailable: ' + e.message);
        }
    }

    function renderDynamics(data) {
        const el = document.getElementById('dynamics-display');
        if (!el) return;
        const signals = [
            { label: 'Market Vitality', value: data.market_vitality },
            { label: 'Citizen Gravity', value: data.citizen_gravity },
            { label: 'Theme Coherence', value: data.theme_coherence },
            { label: 'Event Dynamics', value: data.event_dynamics },
            { label: 'Profile Impact', value: data.profile_impact },
            { label: 'Faith Coherence', value: data.faith_coherence },
            { label: 'Domestic Coherence', value: data.domestic_coherence },
            { label: 'Rumor Coherence', value: data.rumor_coherence },
            { label: 'Economic Perk', value: data.economic_perk },
            { label: 'Entity Legitimacy', value: data.entity_legitimacy },
        ];
        el.innerHTML = `
            <div class="dynamics-header">
                <span class="dynamics-region">${esc(data.region)}</span>
                <span class="dynamics-score">Score: ${data.score || 0}</span>
            </div>
            <div class="dynamics-grid">
                ${signals.map(s => `
                    <div class="dynamics-signal">
                        <span class="signal-label">${s.label}</span>
                        <span class="signal-value">${s.value ?? 0}</span>
                    </div>
                `).join('')}
            </div>`;
    }

    // === REGION SELECT OPTIONS ===
    async function populateRegionSelects() {
        try {
            const res = await api('/api/regions');
            const regions = res.data || (Array.isArray(res) ? res : []);
            const weatherSelect = document.getElementById('weather-region-select');
            const dynamicsSelect = document.getElementById('dynamics-region-select');
            const opts = regions.map(r => `<option value="${esc(r.name || r.region_id)}">${esc(r.name || r.region_id)}</option>`).join('');
            if (weatherSelect) weatherSelect.innerHTML = '<option value="">Select a region</option>' + opts;
            if (dynamicsSelect) dynamicsSelect.innerHTML = '<option value="">Select a region</option>' + opts;
        } catch (e) {
            // regions optional
        }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.openThemeDashboard = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        if (window.PanelManager) {
            window.PanelManager.open(overlayEl);
        } else {
            overlayEl.style.display = 'flex';
        }
        loadThemeVector();
        populateRegionSelects();
    };

    window.switchThemeTab = switchThemeTab;
    window.loadWeatherForRegion = loadWeatherForRegion;
    window.loadDynamicsForRegion = loadDynamicsForRegion;

    if (document.getElementById('theme-dashboard-overlay')) init();
})();
