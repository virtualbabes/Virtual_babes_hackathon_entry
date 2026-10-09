// ============================================================================
// pathway_avenues.js — 12 Career Pathways with unique styling & tier tracking
// ----------------------------------------------------------------------------
// Each pathway has: color palette, icon set, animation style, 5-tier unlocks
// Synced with /api/player/progression for live unlock state
// ============================================================================

// --- 12 Pathway Definitions ---
const PATHWAYS = {
    'P-Shadow': {
        name: 'Shadow Ops',
        domain: 'Gossip · Justice Recruiter',
        faction: 'JUSTICE',
        colors: { primary: '#4a148c', secondary: '#7b1fa2', accent: '#e040fb', glow: '#aa00ff' },
        icon: '🕵️',
        animation: 'steady-pulse',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Shadow Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['+5% Shadow Bonus', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Shadow Bonus', 'Unique Ritual'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Shadow Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Shadow Bonus', 'Governor Eligible'] },
        ]
    },
    'P-Lockdown': {
        name: 'Lockdown',
        domain: 'Kidnapper · AOS · Warden',
        faction: 'JUSTICE',
        colors: { primary: '#b71c1c', secondary: '#d32f2f', accent: '#ff5252', glow: '#ff1744' },
        icon: '🔒',
        animation: 'steady-pulse',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Lockdown Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['+5% Lockdown Bonus', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Lockdown Bonus', 'Unique Contract'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Lockdown Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Lockdown Bonus', 'Warden Elite'] },
        ]
    },
    'P-Ledger': {
        name: 'Ledger',
        domain: 'Launderer · Mutation Auditor',
        faction: 'UNDERWORLD',
        colors: { primary: '#1b5e20', secondary: '#2e7d32', accent: '#69f0ae', glow: '#00e676' },
        icon: '📒',
        animation: 'sharp-flicker',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Ledger Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['-2% Loan Interest', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Ledger Bonus', 'Unique Audit'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Ledger Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Ledger Bonus', 'Boss Eligible'] },
        ]
    },
    'P-Syndicate': {
        name: 'Syndicate',
        domain: 'Underworld Boss · Judge',
        faction: 'UNDERWORLD',
        colors: { primary: '#ff6f00', secondary: '#ff8f00', accent: '#ffab40', glow: '#ff9100' },
        icon: '💀',
        animation: 'sharp-flicker',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Syndicate Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['+5% Syndicate Bonus', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Syndicate Bonus', 'Unique Heist'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Syndicate Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Syndicate Bonus', 'Boss Elite'] },
        ]
    },
    'P-Tax': {
        name: 'Tax',
        domain: 'Tax Auditor · Launderer',
        faction: 'HYBRID',
        colors: { primary: '#f57f17', secondary: '#fbc02d', accent: '#fff176', glow: '#ffd600' },
        icon: '💰',
        animation: 'breathing-wave',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Tax Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['+10% Dividend Yield', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Tax Bonus', 'Unique Audit'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Tax Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Tax Bonus', 'Commissioner Elite'] },
        ]
    },
    'P-Peace': {
        name: 'Peace',
        domain: 'Sector Peacekeeper · Smuggler',
        faction: 'JUSTICE',
        colors: { primary: '#0d47a1', secondary: '#1565c0', accent: '#448aff', glow: '#2979ff' },
        icon: '🕊️',
        animation: 'steady-pulse',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Peace Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['-50% Orphan Fee', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Peace Bonus', 'Unique Ritual'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Peace Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Peace Bonus', 'Peacekeeper Elite'] },
        ]
    },
    'P-Intel': {
        name: 'Intel',
        domain: 'Intel Agent · ArcNet Operative',
        faction: 'JUSTICE',
        colors: { primary: '#006064', secondary: '#00838f', accent: '#18ffff', glow: '#00e5ff' },
        icon: '📡',
        animation: 'steady-pulse',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Intel Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['+15% Search-Rescue', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Intel Bonus', 'Unique Scan'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Intel Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Intel Bonus', 'Multi-Instance LLM'] },
        ]
    },
    'P-Justice': {
        name: 'Justice',
        domain: 'Justice Recruiter · Bounty Hunter',
        faction: 'JUSTICE',
        colors: { primary: '#1a237e', secondary: '#283593', accent: '#536dfe', glow: '#3d5afe' },
        icon: '⚖️',
        animation: 'steady-pulse',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Justice Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['+5% Justice Bonus', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Justice Bonus', 'Unique Mission'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Justice Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Justice Bonus', 'Judge Elite'] },
        ]
    },
    'P-AOS': {
        name: 'AOS',
        domain: 'AOS · Sector Peacekeeper',
        faction: 'JUSTICE',
        colors: { primary: '#33691e', secondary: '#558b2f', accent: '#b2ff59', glow: '#76ff03' },
        icon: '🛡️',
        animation: 'steady-pulse',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['AOS Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['+5% AOS Bonus', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% AOS Bonus', 'Unique Defense'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% AOS Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% AOS Bonus', 'AOS Commander'] },
        ]
    },
    'P-Commissioner': {
        name: 'Commissioner',
        domain: 'Justice Commissioner · Tax Auditor',
        faction: 'HYBRID',
        colors: { primary: '#880e4f', secondary: '#ad1457', accent: '#ff4081', glow: '#f50057' },
        icon: '🏛️',
        animation: 'breathing-wave',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Commissioner Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['Verified Creator', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Commissioner Bonus', 'Unique Proposal'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Commissioner Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Commissioner Bonus', 'Governor Elite'] },
        ]
    },
    'P-Forensic': {
        name: 'Forensic',
        domain: 'Mutation Auditor · Launderer',
        faction: 'UNDERWORLD',
        colors: { primary: '#3e2723', secondary: '#4e342e', accent: '#bcaaa4', glow: '#8d6e63' },
        icon: '🔬',
        animation: 'sharp-flicker',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Forensic Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['+5% Forensic Bonus', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Forensic Bonus', 'Unique Audit'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Forensic Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Forensic Bonus', 'Forensic Elite'] },
        ]
    },
    'P-Boss': {
        name: 'Boss',
        domain: 'Underworld Boss · Judge',
        faction: 'UNDERWORLD',
        colors: { primary: '#263238', secondary: '#37474f', accent: '#90a4ae', glow: '#607d8b' },
        icon: '👑',
        animation: 'sharp-flicker',
        tiers: [
            { name: 'Initiate', xp: 0, unlocks: ['Boss Badge', 'T1 Card Pool'] },
            { name: 'Apprentice', xp: 100, unlocks: ['+5% Boss Bonus', 'T2 Items'] },
            { name: 'Journeyman', xp: 500, unlocks: ['+10% Boss Bonus', 'Unique Heist'] },
            { name: 'Expert', xp: 2000, unlocks: ['+15% Boss Bonus', 'T4 Card Pool'] },
            { name: 'Master', xp: 10000, unlocks: ['+20% Boss Bonus', 'Underworld Kingpin'] },
        ]
    },
};

// --- Tier Calculation ---
function calculateTier(pathwayId, xp, achievementCount) {
    const pathway = PATHWAYS[pathwayId];
    if (!pathway) return 0;
    let tier = 0;
    for (let i = pathway.tiers.length - 1; i >= 0; i--) {
        if (xp >= pathway.tiers[i].xp) {
            tier = i + 1;
            break;
        }
    }
    return tier;
}

function getTierProgress(pathwayId, xp) {
    const pathway = PATHWAYS[pathwayId];
    if (!pathway) return { current: 0, next: 0, pct: 0 };
    let currentTier = 0;
    for (let i = pathway.tiers.length - 1; i >= 0; i--) {
        if (xp >= pathway.tiers[i].xp) {
            currentTier = i;
            break;
        }
    }
    const current = pathway.tiers[currentTier].xp;
    const next = currentTier < pathway.tiers.length - 1 ? pathway.tiers[currentTier + 1].xp : current;
    const pct = next > current ? Math.min(100, ((xp - current) / (next - current)) * 100) : 100;
    return { current, next, pct, tier: currentTier + 1 };
}

// --- CSS Variable Generation ---
function getCSSVariables(pathwayId) {
    const p = PATHWAYS[pathwayId];
    if (!p) return {};
    return {
        '--pathway-primary': p.colors.primary,
        '--pathway-secondary': p.colors.secondary,
        '--pathway-accent': p.colors.accent,
        '--pathway-glow': p.colors.glow,
    };
}

// --- Animation Keyframes ---
function injectAnimationStyles() {
    if (document.getElementById('pathway-animations')) return;
    const style = document.createElement('style');
    style.id = 'pathway-animations';
    style.textContent = `
        @keyframes pathway-pulse {
            0%, 100% { box-shadow: 0 0 8px var(--pathway-glow); }
            50% { box-shadow: 0 0 20px var(--pathway-glow), 0 0 40px var(--pathway-glow); }
        }
        @keyframes pathway-flicker {
            0%, 100% { box-shadow: 0 0 5px var(--pathway-glow); }
            25% { box-shadow: 0 0 15px var(--pathway-glow); }
            50% { box-shadow: 0 0 3px var(--pathway-glow); }
            75% { box-shadow: 0 0 20px var(--pathway-glow); }
        }
        @keyframes pathway-breath {
            0%, 100% { box-shadow: 0 0 6px var(--pathway-glow); transform: scale(1); }
            50% { box-shadow: 0 0 18px var(--pathway-glow); transform: scale(1.02); }
        }
        .pathway-node { animation: pathway-pulse 2s ease-in-out infinite; }
        .pathway-node.faction-justice { animation-name: pathway-pulse; }
        .pathway-node.faction-underworld { animation-name: pathway-flicker; }
        .pathway-node.faction-hybrid { animation-name: pathway-breath; }
    `;
    document.head.appendChild(style);
}

// --- Expose ---
export {
    PATHWAYS,
    calculateTier,
    getTierProgress,
    getCSSVariables,
    injectAnimationStyles,
};

export function getPathway(id) {
    return PATHWAYS[id] || null;
}

export function getAllPathways() {
    return PATHWAYS;
}

// Auto-inject animation styles
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', injectAnimationStyles);
} else {
    injectAnimationStyles();
}
