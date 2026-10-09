// ============================================================================
// constellation_tutorial.js — 7-step guided onboarding journey
// ----------------------------------------------------------------------------
// Welcome → First Node → Found Church → Meet Pet → First Battle → Claim Reward → Explore
// Required interactions to proceed. Skip button always available. localStorage flag.
// ============================================================================

(function () {
    'use strict';
    let overlayEl = null;
    let currentStep = 0;
    let tutorialActive = false;
    let actionRequired = false;

    const TUTORIAL_STEPS = [
        {
            npc: 'anya',
            name: 'Anya',
            message: "Welcome to the Constellation Hub, traveler. This is where your journey begins. Each node represents a system you can unlock. Let's explore together!",
            highlight: null,
            action: null,
            required: false,
        },
        {
            npc: 'vbabes',
            name: 'Vbabes',
            message: "Hey! I'm Vbabes. That glowing node is Faith Church — the heart of your social territory. Click it to learn more!",
            highlight: '.feature-node[data-feature="faith"]',
            action: 'click-node',
            required: true,
        },
        {
            npc: 'anya',
            name: 'Anya',
            message: "Perfect! Every church generates VBV over time. Let's found your first church — click the 'Open' tab to get started.",
            highlight: '.ch-tab[data-tab="open"]',
            action: 'click-tab',
            required: true,
        },
        {
            npc: 'vbabes',
            name: 'Vbabes',
            message: "Great! Now let's meet your first pet. Pets battle in events for stats and VBV. Click the Pet Battle Arena node!",
            highlight: '.feature-node[data-feature="entity"]',
            action: 'click-node',
            required: true,
        },
        {
            npc: 'anya',
            name: 'Anya',
            message: "In the arena, you'll challenge other pets to battles. For this tutorial, we'll auto-win your first fight. Click 'Find Match'!",
            highlight: '#pba-find-match-btn',
            action: 'click-match',
            required: true,
        },
        {
            npc: 'vbabes',
            name: 'Vbabes',
            message: "Victory! You earned VBV and XP. Your pet will grow stronger with each win. Check your VBV balance at the top-right!",
            highlight: '.vbx-balance-widget',
            action: null,
            required: false,
        },
        {
            npc: 'anya',
            name: 'Anya',
            message: "You're ready to explore freely. Unlock nodes, battle pets, grow your churches, and climb the leaderboard. Good luck, traveler!",
            highlight: null,
            action: null,
            required: false,
        },
    ];

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('constellation-tutorial-overlay');
        if (!overlayEl) {
            // SELF-MOUNT. Nothing in the shell provided this root, so the null-check below made the
            // whole tutorial a silent no-op: `init()` returned and `openConstellationTutorial()` had
            // no surface to render into.
            document.body.insertAdjacentHTML('beforeend', '<div id="constellation-tutorial-overlay"></div>');
            overlayEl = document.getElementById('constellation-tutorial-overlay');
            if (!overlayEl) return;
        }
        overlayEl.className = 'overlay constellation-tutorial-overlay';
        overlayEl.style.display = 'none';
        overlayEl.innerHTML = `
            <div class="tutorial-dialog" id="tutorial-dialog">
                <div class="tutorial-npc-avatar" id="tutorial-npc-avatar">👩</div>
                <div class="tutorial-content">
                    <div class="tutorial-npc-name" id="tutorial-npc-name">Anya</div>
                    <div class="tutorial-message" id="tutorial-message"></div>
                </div>
                <div class="tutorial-actions">
                    <button class="vbt-btn vbt-btn-secondary" onclick="window.constellationTutorialSkip()">Skip Tutorial</button>
                    <button class="vbt-btn vbt-btn-primary" id="tutorial-next-btn" onclick="window.constellationTutorialNext()">Next →</button>
                </div>
            </div>
            <div class="tutorial-highlight-ring" id="tutorial-highlight-ring"></div>
            <div class="tutorial-step-indicator" id="tutorial-step-indicator"></div>
        `;
    }

    function showStep(step) {
        if (step >= TUTORIAL_STEPS?.length ?? 0) {
            closeTutorial();
            return;
        }
        const data = TUTORIAL_STEPS[step];
        currentStep = step;
        actionRequired = data.required;

        const avatar = document.getElementById('tutorial-npc-avatar');
        const name = document.getElementById('tutorial-npc-name');
        const message = document.getElementById('tutorial-message');
        const ring = document.getElementById('tutorial-highlight-ring');
        const nextBtn = document.getElementById('tutorial-next-btn');
        const indicator = document.getElementById('tutorial-step-indicator');

        if (avatar) avatar.textContent = data.npc === 'anya' ? '👩' : '👩‍🦰';
        if (name) name.textContent = data?.name;
        if (message) message.textContent = data.message;

        // Highlight element
        if (ring) {
            if (data.highlight) {
                const target = document.querySelector(data.highlight);
                if (target) {
                    const rect = target.getBoundingClientRect();
                    ring.style.display = 'block';
                    ring.style.left = (rect.left - 8) + 'px';
                    ring.style.top = (rect.top - 8) + 'px';
                    ring.style.width = (rect.width + 16) + 'px';
                    ring.style.height = (rect.height + 16) + 'px';
                } else {
                    ring.style.display = 'none';
                }
            } else {
                ring.style.display = 'none';
            }
        }

        // Disable next button if action required
        if (nextBtn) {
            nextBtn.disabled = data.required;
            nextBtn.textContent = data.required ? 'Complete Action' : 'Next →';
        }

        // Update step indicator
        if (indicator) {
            indicator.innerHTML = TUTORIAL_STEPS.map((_, i) => `
                <span class="step-dot ${i === step ? 'active' : i < step ? 'done' : ''}"></span>
            `).join('');
        }

        // Play sound
        if (typeof window.audioEngine !== 'undefined') {
            window.audioEngine.click();
        }
    }

    window.openConstellationTutorial = function () {
        init();
        if (!overlayEl) return;
        overlayEl.style.display = 'flex';
        tutorialActive = true;
        currentStep = 0;
        showStep(0);
    };

    window.constellationTutorialNext = function () {
        if (actionRequired) return;
        showStep(currentStep + 1);
    };

    window.constellationTutorialSkip = function () {
        closeTutorial();
    };

    // Called by game when required action is completed
    window.tutorialActionComplete = function () {
        if (!tutorialActive) return;
        actionRequired = false;
        const nextBtn = document.getElementById('tutorial-next-btn');
        if (nextBtn) {
            nextBtn.disabled = false;
            nextBtn.textContent = 'Next →';
        }
    };

    function closeTutorial() {
        tutorialActive = false;
        if (overlayEl) overlayEl.style.display = 'none';
        const ring = document.getElementById('tutorial-highlight-ring');
        if (ring) ring.style.display = 'none';
        try { localStorage.setItem('constellation_tutorial_seen', 'true'); } catch (e) {}
        
        // Show contextual tip
        if (typeof window.showToast === 'function') {
            window.showToast("Tip: Click any node to explore, or check Daily Challenges for rewards!", 'success');
        }
    }

    // Auto-show on first visit
    window.addEventListener('load', function () {
        try {
            if (!localStorage.getItem('constellation_tutorial_seen')) {
                setTimeout(function () {
                    if (window.openConstellationTutorial) window.openConstellationTutorial();
                }, 1000);
            }
        } catch (e) {}
    });

    if (document.getElementById('constellation-tutorial-overlay')) init();
})();
