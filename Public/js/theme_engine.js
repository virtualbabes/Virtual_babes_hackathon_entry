// ============================================================================
// theme_engine.js — Dynamic color system driven by ThemeVector
// ----------------------------------------------------------------------------
// Sets CSS custom properties based on player's theme element.
// Drives: accent colors, glow intensity, particle speed, background darkness.
// ============================================================================

(function () {
    'use strict';
    let currentElement = 'fire';
    let currentIntensity = 0.5;

    const ELEMENT_COLORS = {
        fire: { hex: '#FF6B35', rgb: '255, 107, 53' },
        water: { hex: '#00D4FF', rgb: '0, 212, 255' },
        earth: { hex: '#7CB342', rgb: '124, 179, 66' },
        air: { hex: '#B0BEC5', rgb: '176, 190, 197' },
        aether: { hex: '#9C27B0', rgb: '156, 39, 176' },
        machine: { hex: '#00BCD4', rgb: '0, 188, 212' },
        neutral: { hex: '#607D8B', rgb: '96, 125, 139' },
    };

    // ── The SERVED theme vector is the authority ────────────────────────────────
    // `handleThemeVector` (theme_engine.go) computes the vector from the engine's own terms:
    // tone, element, intensity, entropy and the MoodTag sourced from the wallet's NON-LOCKED
    // bound assets (§27.6/§27.7). Before this, the palette was driven ONLY by
    // `localStorage.theme_element`, so the §27 contract existed and was consumed by ten SCSS
    // partials while nothing ever fed it a real value — the layer was INERT, not missing.
    // The element enum is the server's: 0 none, 1 fire, 2 water, 3 earth, 4 air, 5 aether,
    // 6 machine.
    const SERVER_ELEMENTS = ['neutral', 'fire', 'water', 'earth', 'air', 'aether', 'machine'];

    function applyServerVector(wallet) {
        if (!wallet) return Promise.resolve(false);
        const url = '/api/theme/vector?wallet=' + encodeURIComponent(wallet);
        return fetch(url)
            .then(function (r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.json(); })
            .then(function (v) {
                if (!v || v.success === false) throw new Error(v && v.error ? v.error : 'refused');
                const name = SERVER_ELEMENTS[Number(v.element) || 0] || 'neutral';
                currentElement = name;
                applyTheme(name);
                // intensity is 0..100 on the wire; the CSS token is a 0..1 view factor.
                setIntensity(Math.max(0, Math.min(100, Number(v.intensity) || 0)) / 100);
                window.__themeVector = {
                    source: 'server', element: name, element_id: Number(v.element) || 0,
                    tone: Number(v.tone) || 0, intensity: Number(v.intensity) || 0,
                    entropy: Number(v.entropy) || 0, mood_tag: Number(v.mood_tag) || 0,
                };
                document.documentElement.setAttribute('data-theme-source', 'server');
                return true;
            })
            .catch(function (e) {
                // A refused read KEEPS the last known theme and STATES the refusal — it never
                // silently reverts to a default that would look like the player's real element.
                window.__themeVector = { source: 'fallback', element: currentElement, reason: String(e && e.message || e) };
                document.documentElement.setAttribute('data-theme-source', 'fallback');
                return false;
            });
    }

    function init() {
        // Load saved theme or default to fire
        try {
            const saved = localStorage.getItem('theme_element');
            if (saved && ELEMENT_COLORS[saved]) {
                currentElement = saved;
            }
        } catch (e) {}
        applyTheme(currentElement);
        // Then ask the engine. The server value wins when it answers.
        try {
            const wallet = window.currentWallet || (window.getActiveWallet && window.getActiveWallet()) || '';
            if (wallet) applyServerVector(wallet);
        } catch (e) {}
    }

    function setElement(element) {
        if (!ELEMENT_COLORS[element]) return;
        currentElement = element;
        applyTheme(element);
        try { localStorage.setItem('theme_element', element); } catch (e) {}
    }

    function applyTheme(element) {
        const colors = ELEMENT_COLORS[element];
        const root = document.documentElement;
        
        root.style.setProperty('--theme-accent', colors.hex);
        root.style.setProperty('--theme-accent-rgb', colors.rgb);
        root.style.setProperty('--theme-accent-dim', `rgba(${colors.rgb}, 0.2)`);
        root.style.setProperty('--theme-accent-glow', `rgba(${colors.rgb}, 0.4)`);
        
        // Update audio intensity if available
        if (typeof window.audioEngine !== 'undefined') {
            window.audioEngine.setIntensity(currentIntensity);
        }
    }

    function setIntensity(intensity) {
        currentIntensity = Math.max(0, Math.min(1, intensity));
        const root = document.documentElement;
        root.style.setProperty('--arena-glow-intensity', currentIntensity);
        
        // Update particle speed
        const canvas = document.getElementById('constellation-particle-canvas');
        if (canvas) {
            canvas.style.opacity = 0.3 + (currentIntensity * 0.7);
        }
    }

    function getElement() { return currentElement; }
    function getIntensity() { return currentIntensity; }

    // Parallax mouse tracking
    function initParallax() {
        const container = document.querySelector('.parallax-container');
        if (!container) return;

        document.addEventListener('mousemove', (e) => {
            const x = (e.clientX / window.innerWidth - 0.5) * 10;
            const y = (e.clientY / window.innerHeight - 0.5) * 10;
            
            const layers = container.querySelectorAll('.parallax-layer');
            layers.forEach((layer, i) => {
                const depth = (i + 1) * 0.5;
                layer.style.transform = `translate(${x * depth}px, ${y * depth}px)`;
            });
        });
    }

    // Number counting animation
    function animateCount(element, targetValue, duration) {
        duration = duration || 400;
        const startValue = parseFloat(element.textContent) || 0;
        const startTime = performance.now();
        
        function update(currentTime) {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const eased = 1 - Math.pow(1 - progress, 3); // ease-out cubic
            const current = startValue + (targetValue - startValue) * eased;
            element.textContent = current.toFixed(2);
            
            if (progress < 1) {
                requestAnimationFrame(update);
            }
        }
        
        requestAnimationFrame(update);
    }

    window.themeEngine = {
        init: init,
        setElement: setElement,
        setIntensity: setIntensity,
        getElement: getElement,
        getIntensity: getIntensity,
        initParallax: initParallax,
        animateCount: animateCount,
        // Re-read the SERVED vector (call after a wallet connect, or to refresh). Returns a
        // promise resolving to true when the server answered and false when the read was refused
        // (in which case the last known theme is kept and the reason is in window.__themeVector).
        applyServerVector: applyServerVector,
        elements: SERVER_ELEMENTS.slice(),
    };

    if (document.body) init();
})();
