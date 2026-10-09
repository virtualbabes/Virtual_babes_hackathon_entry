// ============================================================================
// settings_panel.js — Player settings: audio, graphics, account
// ============================================================================

(function () {
    'use strict';
    let overlayEl = null;

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('settings-overlay');
        if (!overlayEl) return;
        overlayEl.className = 'overlay settings-overlay';
        overlayEl.style.display = 'none';

        overlayEl.innerHTML = `
            <div class="settings-panel">
                <button class="settings-close" onclick="window.closeSettings()">✕</button>
                <h2>⚙️ Settings</h2>
                
                <div class="settings-section">
                    <h3>Audio</h3>
                    <div class="setting-row">
                        <label>Volume</label>
                        <input type="range" class="settings-slider" id="settings-volume" min="0" max="100" value="60" onchange="window.settingsSetVolume(this.value)" />
                        <span id="settings-volume-value">60%</span>
                    </div>
                    <div class="setting-row">
                        <label>Mute</label>
                        <input type="checkbox" id="settings-mute" onchange="window.settingsToggleMute(this.checked)" />
                    </div>
                </div>

                <div class="settings-section">
                    <h3>Graphics</h3>
                    <div class="setting-row">
                        <label>Quality</label>
                        <select id="settings-quality" onchange="window.settingsSetQuality(this.value)">
                            <option value="low">Low</option>
                            <option value="medium" selected>Medium</option>
                            <option value="high">High</option>
                        </select>
                    </div>
                </div>

                <div class="settings-section">
                    <h3>Account</h3>
                    <div class="setting-row">
                        <label>Wallet</label>
                        <span id="settings-wallet" class="settings-wallet-address">0x----...----</span>
                    </div>
                    <div class="setting-row">
                        <label>Joined</label>
                        <span id="settings-join-date">--</span>
                    </div>
                    <div class="setting-row">
                        <label>Total VBV Earned</label>
                        <span id="settings-total-vbv">0.00 VBV</span>
                    </div>
                </div>

                <div class="settings-section">
                    <h3>Tutorial</h3>
                    <button class="vbt-btn vbt-btn-secondary" onclick="window.settingsResetTutorial()">Reset Tutorial</button>
                </div>

                <div class="settings-section">
                    <h3>Wallet</h3>
                    <button class="vbt-btn vbt-btn-secondary" onclick="window.settingsDisconnect()">Disconnect Wallet</button>
                </div>
            </p>
        `;

        loadSettings();
    }

    function loadSettings() {
        // Load saved settings
        try {
            const saved = localStorage.getItem('settings');
            if (saved) {
                const settings = JSON.parse(saved);
                const volumeEl = document.getElementById('settings-volume');
                const muteEl = document.getElementById('settings-mute');
                const qualityEl = document.getElementById('settings-quality');
                if (volumeEl) volumeEl.value = settings.volume || 60;
                if (muteEl) muteEl.checked = settings.mute || false;
                if (qualityEl) qualityEl.value = settings.quality || 'medium';
            }
        } catch (e) {}
    }

    function saveSettings() {
        try {
            const settings = {
                volume: document.getElementById('settings-volume')?.value || 60,
                mute: document.getElementById('settings-mute')?.checked || false,
                quality: document.getElementById('settings-quality')?.value || 'medium',
            };
            localStorage.setItem('settings', JSON.stringify(settings));
        } catch (e) {}
    }

    function setVolume(value) {
        const el = document.getElementById('settings-volume-value');
        if (el) el.textContent = value + '%';
        if (typeof window.audioEngine !== 'undefined') {
            window.audioEngine.setVolume(value / 100);
        }
        saveSettings();
    }

    function toggleMute(checked) {
        if (typeof window.audioEngine !== 'undefined') {
            window.audioEngine.toggleMute();
        }
        saveSettings();
    }

    function setQuality(value) {
        // Adjust particle count, bloom, parallax based on quality
        const canvas = document.getElementById('constellation-particle-canvas');
        if (canvas) {
            if (value === 'low') canvas.style.display = 'none';
            else canvas.style.display = 'block';
        }
        saveSettings();
    }

    function resetTutorial() {
        try { localStorage.removeItem('constellation_tutorial_seen'); } catch (e) {}
        if (typeof window.showToast === 'function') {
            window.showToast('Tutorial reset!', 'success');
        }
    }

    function disconnect() {
        if (typeof window.walletDisconnect === 'function') {
            window.walletDisconnect();
        }
        if (typeof window.showToast === 'function') {
            showToast('Wallet disconnected', 'neutral');
        }
    }

    window.openSettings = function () {
        init();
        overlayEl.style.display = 'flex';
    };

    window.closeSettings = function () {
        if (overlayEl) overlayEl.style.display = 'none';
    };

    window.settingsSetVolume = setVolume;
    window.settingsToggleMute = toggleMute;
    window.settingsSetQuality = setQuality;
    window.settingsResetTutorial = resetTutorial;
    window.settingsDisconnect = disconnect;

    if (document.getElementById('settings-overlay')) init();
})();
