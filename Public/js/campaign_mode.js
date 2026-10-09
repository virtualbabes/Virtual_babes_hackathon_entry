// ============================================================================
// campaign_mode.js — Campaign Mode UI
// ----------------------------------------------------------------------------
// Ported from Triple Triad: campaign.js (797 lines)
// Single-player campaign with lives, stage progression, unlocks
// ============================================================================

var API_BASE = '/api';

// Campaign definitions
const CAMPAIGNS = [
    { id: 'playground', name: 'Playground', icon: '🎪', desc: 'Learn the basics', stages: 5, unlocked: true },
    { id: 'street_brawl', name: 'Street Brawl', icon: '👊', desc: 'Rough neighborhood fights', stages: 8, unlocked: false },
    { id: 'arena', name: 'The Arena', icon: '🏟️', desc: 'Professional competition', stages: 10, unlocked: false },
    { id: 'underworld', name: 'Underworld', icon: '🕳️', desc: 'Criminal underworld', stages: 12, unlocked: false },
    { id: 'heaven', name: 'Heaven\'s Gate', icon: '⛪', desc: 'Divine trials', stages: 15, unlocked: false }
];

// Stage template
const STAGE_TEMPLATE = (campaignId, stageNum) => ({
    id: `${campaignId}_${stageNum}`,
    name: `Stage ${stageNum}`,
    difficulty: Math.min(10, 3 + stageNum),
    reward: 1000 * stageNum,
    unlocked: stageNum === 1,
    completed: false
});

// State
let selectedCampaign = 'playground';
let campaignLives = 3;
let currentStage = 1;

// --- Init ---
export function initCampaignMode() {
    const el = document.getElementById('wd-campaign');
    if (!el) return;
    renderCampaignMode(el);
}

function renderCampaignMode(el) {
    el.innerHTML = `
        <div class="campaign-container">
            <div class="campaign-header">
                <span class="campaign-icon">📖</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Campaign Mode</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Progress through stages, earn rewards</p>
                </div>
                <div class="campaign-lives">
                    <span class="lives-label">Lives</span>
                    <span class="lives-value">${'❤️'.repeat(campaignLives)}${'🖤'.repeat(Math.max(0, 3 - campaignLives))}</span>
                </div>
            </div>

            <!-- Campaign Selection -->
            <div class="campaign-list">
                ${CAMPAIGNS.map(c => `
                    <div class="campaign-card ${selectedCampaign === c.id ? 'active' : ''} ${c.unlocked ? '' : 'locked'}" data-campaign="${c.id}">
                        <span class="campaign-card-icon">${c.icon}</span>
                        <span class="campaign-card-name">${c.name}</span>
                        <span class="campaign-card-desc">${c.desc}</span>
                        <span class="campaign-card-stages">${c.stages} stages</span>
                        ${c.unlocked ? '' : '<span class="campaign-card-lock">🔒</span>'}
                    </div>
                `).join('')}
            </div>

            <!-- Stage Map -->
            <div class="stage-map">
                <h4 style="color:#e5e7eb;">${CAMPAIGNS.find(c => c.id === selectedCampaign)?.name || 'Select Campaign'}</h4>
                <div class="stage-grid">
                    ${renderStageGrid()}
                </div>
            </div>

            <!-- Selected Stage Info -->
            <div class="stage-info" id="stage-info">
                <p style="color:#90a4ae;">Select a stage to begin</p>
            </div>
        </div>
    `;
    attachCampaignListeners(el);
}

function renderStageGrid() {
    const campaign = CAMPAIGNS.find(c => c.id === selectedCampaign);
    if (!campaign) return '';
    
    let html = '';
    for (let i = 1; i <= campaign.stages; i++) {
        const unlocked = i === 1 || (i > 1 && i <= currentStage);
        const completed = i < currentStage;
        html += `
            <div class="stage-slot ${unlocked ? 'unlocked' : 'locked'} ${completed ? 'completed' : ''}" data-stage="${i}">
                <span class="stage-num">${i}</span>
                ${completed ? '<span class="stage-check">✓</span>' : ''}
                ${!unlocked ? '<span class="stage-lock">🔒</span>' : ''}
            </div>
        `;
    }
    return html;
}

function attachCampaignListeners(el) {
    el.querySelectorAll('.campaign-card').forEach(card => {
        card.addEventListener('click', () => {
            if (card.classList.contains('locked')) {
                if (window.showToast) window.showToast('Complete previous campaign to unlock', 'warning');
                return;
            }
            selectedCampaign = card.dataset.campaign;
            renderCampaignMode(el);
        });
    });

    el.querySelectorAll('.stage-slot.unlocked').forEach(slot => {
        slot.addEventListener('click', () => {
            currentStage = parseInt(slot.dataset.stage);
            renderCampaignMode(el);
            if (window.showToast) window.showToast(`Stage ${currentStage} selected!`, 'info');
        });
    });
}

// --- Globals ---
window.initCampaignMode = initCampaignMode;
