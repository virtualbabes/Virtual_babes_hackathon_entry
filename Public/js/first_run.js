// first_run.js — First-Run Quick-Start (onboarding flow, UI-Flow pass 1).
// Classic IIFE (sibling of early_tasks.js). Guides a brand-new player through the
// minimum viable demo loop: Connect Wallet → Claim Faucet $VBV → Enter the Hub.
// Live completion: step 0 auto-completes when a wallet connects; step 1 when the
// Faucet Dashboard opens; step 2 closes the flow (hub opened). Contained overlay:
// position:fixed / inset:0 / overflow:hidden — the page itself never scrolls.
(function () {
    'use strict';

    var FLAG = 'vbt_first_run_seen';
    var BOOT_DELAY_MS = 1400; // after load, before we consider showing

    var STEPS = [
        {
            id: 'wallet', label: 'Connect your wallet', hint: 'Link a Voi / Algorand wallet (WalletConnect, Kibisis, Nautilus) — everything is anchored to YOUR wallet.',
            cta: 'Connect Wallet',
            go: function () { return typeof window.handleWalletAction === 'function' && window.handleWalletAction(); }
        },
        {
            id: 'claim', label: 'Claim your $VBV grant', hint: 'Every new player starts with a free $VBV faucet claim — the only in-game token today.',
            cta: 'Open the Faucet',
            go: function () { return typeof window.openFaucetDashboard === 'function' && window.openFaucetDashboard(); }
        },
        {
            id: 'hub', label: 'Enter the Hub', hint: 'The constellation is your command deck — matches, market, citizens, territory.',
            cta: 'Enter the Hub',
            go: function () { return typeof window.openConstellationHub === 'function' && window.openConstellationHub(); }
        }
    ];

    var overlayEl = null, stepIdx = 0, shown = false, pollTimer = null;
    var launched = [false, false, false]; // user initiated step CTA
    var skipped = false;                  // user explicitly skipped this visit

    function walletConnected() {
        try {
            if (typeof window.getActiveWallet === 'function' && window.getActiveWallet()) return true;
            if (typeof window.getWalletAddress === 'function' && window.getWalletAddress()) return true;
            return !!(typeof window.userAddress !== 'undefined' && window.userAddress);
        } catch (_) { return false; }
    }

    function suppressForever() { try { localStorage.setItem(FLAG, '1'); } catch (_) {} }
    function suppressed() { try { return localStorage.getItem(FLAG) === '1'; } catch (_) { return true; } }

    function ensureStyle() {
        if (document.getElementById('fr-style')) return;
        var s = document.createElement('style');
        s.id = 'fr-style';
        s.textContent = [
            '#first-run-overlay{position:fixed;inset:0;overflow:hidden;background:rgba(8,10,18,.82);z-index:9999;display:flex;align-items:center;justify-content:center;}',
            '#first-run-overlay .fr-panel{width:min(540px,92vw);max-height:86vh;overflow-y:auto;border:1px solid rgba(212,175,55,.55);border-radius:14px;background:linear-gradient(160deg,#10141f,#1a2030);color:#dfe7f2;padding:22px 26px;box-shadow:0 0 26px rgba(0,242,254,.22);}',
            '#first-run-overlay h2{color:#ffd34d;margin:0 0 4px;font-size:19px;letter-spacing:.4px;}',
            '#first-run-overlay .fr-sub{color:#8fa3bd;font-size:12.5px;margin:0 0 14px;}',
            '#first-run-overlay .fr-step{border:1px solid #2b3c4e;border-radius:10px;padding:10px 12px;margin:8px 0;display:flex;align-items:flex-start;gap:10px;opacity:.45;}',
            '#first-run-overlay .fr-step.fr-active{border-color:rgba(0,242,254,.75);opacity:1;box-shadow:0 0 10px rgba(0,242,254,.18);}',
            '#first-run-overlay .fr-step.fr-done{opacity:1;border-color:rgba(57,217,138,.6);}',
            '#first-run-overlay .fr-num{width:24px;height:24px;border-radius:12px;background:#1b2a3e;color:#dfe7f2;display:flex;align-items:center;justify-content:center;font-size:12px;flex:0 0 24px;}',
            '#first-run-overlay .fr-step.fr-done .fr-num{background:#39d98a;color:#06240f;}',
            '#first-run-overlay .fr-body{flex:1;min-width:0;}',
            '#first-run-overlay .fr-label{font-weight:600;font-size:14px;margin:0;}',
            '#first-run-overlay .fr-hint{color:#9db1c9;font-size:12px;margin:3px 0 10px;}',
            '#first-run-overlay .fr-cta{display:inline-block;border:1px solid rgba(212,175,55,.7);border-radius:7px;padding:5px 14px;background:rgba(212,175,55,.14);color:#ffd34d;cursor:pointer;font-size:12.5px;}',
            '#first-run-overlay .fr-cta:hover{background:rgba(212,175,55,.32);}',
            '#first-run-overlay .fr-done-badge{color:#39d98a;font-size:13px;font-weight:700;}',
            '#first-run-overlay .fr-actions{margin-top:14px;display:flex;gap:10px;flex-wrap:wrap;align-items:center;}',
            '#first-run-overlay .fr-skip{background:transparent;border:1px solid #40526b;color:#b6c4d8;border-radius:7px;padding:5px 12px;cursor:pointer;font-size:12px;}',
            // The wallet-onboarding door (/api/bridge/onboard) placed in this flow.
            '#first-run-overlay .fr-onboard{margin-top:12px;padding-top:12px;border-top:1px solid #2b3c4e;display:flex;align-items:center;gap:10px;flex-wrap:wrap;}',
            '#first-run-overlay .fr-cta2{border:1px solid rgba(0,242,254,.6);border-radius:7px;padding:5px 14px;background:rgba(0,242,254,.12);color:#8ff3ff;cursor:pointer;font-size:12.5px;}',
            '#first-run-overlay .fr-cta2:hover{background:rgba(0,242,254,.26);}',
            '#first-run-overlay .fr-onboard-status{color:#9db1c9;font-size:12px;}'
        ].join('\n');
        document.head.appendChild(s);
    }

    function render() {
        if (!overlayEl) return;
        var stepsHtml = STEPS.map(function (st, i) {
            var cls = i === stepIdx ? 'fr-active' : (i < stepIdx ? 'fr-done' : '');
            var badge = i < stepIdx
                ? '<span class="fr-done-badge">✓ done</span>'
                : '<button class="fr-cta" data-step="' + i + '">' + st.cta + '</button>';
            return '<div class="fr-step ' + cls + '">' +
                '<div class="fr-num">' + (i + 1) + '</div>' +
                '<div class="fr-body"><p class="fr-label">' + st.label + '</p>' +
                '<p class="fr-hint">' + st.hint + '</p>' + badge + '</div></div>';
        }).join('');
        overlayEl.innerHTML =
            '<div class="fr-panel">' +
            '<h2>🚀 Quick Start</h2>' +
            '<p class="fr-sub">Three steps to your first $VBV. Checkpoints unlock live as you complete each one.</p>' +
            stepsHtml +
            // POST /api/bridge/onboard (OnboardingService.HandleVoiOnboarding) had NO owner anywhere
            // in the client. It is the arena's WALLET ONBOARDING door — it registers the connected
            // wallet (safety/Sybil checks + opt-in) — so its owner is this flow, not the Bridge
            // Router it happens to be routed under. Kept as an explicit control rather than an
            // automatic side effect of connecting, because it writes on-chain state.
            '<div class="fr-onboard">' +
            '<button class="fr-cta2" id="fr-onboard-btn">🪪 Register this wallet on Voi</button>' +
            '<span class="fr-onboard-status" id="fr-onboard-status"></span>' +
            '</div>' +
            '<div class="fr-actions">' +
            '<button class="fr-skip" id="fr-skip">Skip for now</button>' +
            '<label style="color:#8fa3bd;font-size:11.5px;display:flex;align-items:center;gap:6px;cursor:pointer;"><input type="checkbox" id="fr-dontshow" style="accent-color:#00f2fe;"> Don\'t show on next visit</label>' +
            '</div></div>';

        var ctas = overlayEl.querySelectorAll('.fr-cta');
        for (var i = 0; i < ctas.length; i++) {
            (function (idx, btn) {
                btn.addEventListener('click', function () { goStep(idx); });
            })(parseInt(ctas[i].getAttribute('data-step'), 10), ctas[i]);
        }
        var skip = document.getElementById('fr-skip');
        if (skip) skip.addEventListener('click', function () {
            skipped = true;
            if (document.getElementById('fr-dontshow') && document.getElementById('fr-dontshow').checked) suppressForever();
            close();
        });
        var onboard = document.getElementById('fr-onboard-btn');
        if (onboard) onboard.addEventListener('click', function () { registerOnVoi(); });
    }

    // The wallet-onboarding door. The body carries ONLY the wallet, which is what the handler
    // decodes; the outcome (or the refusal) is stated in place.
    function registerOnVoi() {
        var status = document.getElementById('fr-onboard-status');
        var wallet = (typeof window.getActiveWallet === 'function' && window.getActiveWallet()) || window.currentWallet || window.userAddress || '';
        if (!wallet) { if (status) status.textContent = 'Connect a wallet first.'; return; }
        if (status) status.textContent = 'Registering…';
        fetch('/api/bridge/onboard', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ wallet: wallet })
        }).then(function (r) {
            return r.json().catch(function () { return {}; }).then(function (body) {
                if (!r.ok) throw new Error(body.error || ('HTTP ' + r.status));
                return body;
            });
        }).then(function () {
            if (status) status.textContent = 'This wallet is registered.';
        }).catch(function (e) {
            if (status) status.textContent = 'Not registered: ' + e.message;
        });
    }

    function goStep(i) {
        if (i !== stepIdx || i > stepIdx) return;
        launched[i] = true;
        var st = STEPS[i];
        // Never mask the target UI (connect selector / faucet / hub): close the
        // quick-start BEFORE opening it. State advances via the global poll watch.
        close();
        if (typeof st.go === 'function') { try { st.go(); } catch (_) {} }

        if (i === 2) {
            // Enter Hub completes the flow.
            if (typeof window.markHubVisited === 'function') { try { window.markHubVisited(); } catch (_) {} }
            if (typeof window.showToast === 'function') { try { window.showToast('Welcome — your $VBV story begins.', 'success'); } catch (_) {} }
            suppressForever();
        }
    }

    var lastNudge = 0;
    function nudge() {
        var now = (typeof Date !== 'undefined' && Date.now) ? Date.now() : 0;
        if (now - lastNudge < 2500) return; // debounce: never stack nudge reopens
        lastNudge = now;
        window.setTimeout(function () { open(); }, 450);
    }

    // Global watch (runs until suppressed): advances step state on live signals and
    // (first-run only, unless skipped) gently reopens at the NEXT step after the
    // user's action completed — so targets are never masked by the wizard.
    function poll() {
        if (suppressed() || skipped) return;
        if (stepIdx === 0 && walletConnected()) {
            stepIdx = 1;
            if (shown) render();
            else if (launched[0]) nudge();
            return;
        }
        if (stepIdx === 1) {
            if (document.getElementById('faucet-dashboard')) { stepIdx = 2; if (shown) render(); }
            else if (launched[1]) nudge();
            return;
        }
        if (stepIdx === 2 && launched[1] && !document.getElementById('faucet-dashboard')) {
            nudge(); // faucet closed → offer the final "Enter the Hub" step
        }
    }

    function open() {
        ensureStyle();
        if (!overlayEl) {
            overlayEl = document.createElement('div');
            overlayEl.id = 'first-run-overlay';
            document.body.appendChild(overlayEl);
        }
        render();
        overlayEl.style.display = 'flex';
        shown = true;
        if (typeof window.hideAllOverlays === 'function') { try { window.hideAllOverlays(); } catch (_) {} }
        return overlayEl;
    }

    function close() {
        shown = false;
        if (overlayEl) overlayEl.style.display = 'none';
    }

    // Auto-show for a brand-new player (no wallet yet, never suppressed).
    window.setTimeout(function () {
        try {
            if (!pollTimer) pollTimer = window.setInterval(poll, 1500);
            if (!suppressed() && !walletConnected() && document.readyState !== 'loading') open();
        } catch (_) { /* never break boot */ }
    }, BOOT_DELAY_MS);

    // Manual reopen (Dev Hub / admin) + reset for demos/testing.
    window.openFirstRun = function () { return open(); };
    window.resetFirstRun = function () { try { localStorage.removeItem(FLAG); } catch (_) {} close(); };
    window.__firstRunState = function () { return { shown: shown, stepIdx: stepIdx, wallet: !!walletConnected() }; };
})();