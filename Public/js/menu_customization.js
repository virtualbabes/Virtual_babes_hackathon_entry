// ============================================================================
// menu_customization.js — Button shapes, sizes, grid layouts, and controller nav
// ----------------------------------------------------------------------------
// 60+ button shapes, 6 size tiers, 9 grid types, per-button overrides
// Controller-friendly: each grid type maps to a navigation model
// ============================================================================

// ============================================================
// 1. SHAPE LIBRARY
// ============================================================
const SHAPES = {
    circle: {
        name: 'Circle',
        icon: '●',
        clip: 'circle(50% at 50% 50%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Omnidirectional — all 4 controller directions valid',
    },
    oval: {
        name: 'Oval',
        icon: '⬭',
        clip: 'ellipse(50% 35% at 50% 50%)',
        hitDirs: ['left', 'right'],
        desc: 'Horizontal emphasis — left/right primary',
    },
    star: {
        name: 'Star',
        icon: '★',
        clip: 'polygon(50% 0%, 61% 35%, 98% 35%, 68% 57%, 79% 91%, 50% 70%, 21% 91%, 32% 57%, 2% 35%, 39% 35%)',
        hitDirs: ['up', 'down'],
        desc: 'Top/bottom points — hero button feel',
    },
    triangle: {
        name: 'Triangle',
        icon: '▲',
        clip: 'polygon(50% 0%, 0% 100%, 100% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Apex up — progression/growth metaphor',
    },
    triangleDown: {
        name: 'Triangle Down',
        icon: '▼',
        clip: 'polygon(0% 0%, 100% 0%, 50% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Apex down — descent/warning metaphor',
    },
    oblong: {
        name: 'Oblong',
        icon: '▬',
        clip: 'inset(0% 0% 0% 0% round 25px)',
        hitDirs: ['left', 'right'],
        desc: 'Pill shape — horizontal action bar',
    },
    hexagon: {
        name: 'Hexagon',
        icon: '⬡',
        clip: 'polygon(25% 0%, 75% 0%, 100% 50%, 75% 100%, 25% 100%, 0% 50%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Tactical/industrial — P-Justice, P-AOS pathways',
    },
    octagon: {
        name: 'Octagon',
        icon: '⯃',
        clip: 'polygon(30% 0%, 70% 0%, 100% 30%, 100% 70%, 70% 100%, 30% 100%, 0% 70%, 0% 30%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Stop sign — P-Lockdown, P-Warden pathways',
    },
    rectangle: {
        name: 'Rectangle',
        icon: '■',
        clip: 'inset(0% 0% 0% 0% round 6px)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Standard — neutral, clean',
    },
    diamond: {
        name: 'Diamond',
        icon: '◆',
        clip: 'polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Diagonal emphasis — P-Syndicate, P-Boss pathways',
    },
    pentagon: {
        name: 'Pentagon',
        icon: '⬠',
        clip: 'polygon(50% 0%, 100% 38%, 82% 100%, 18% 100%, 0% 38%)',
        hitDirs: ['up', 'down'],
        desc: 'Five-point — P-Commissioner, P-Forensic pathways',
    },
    shield: {
        name: 'Shield',
        icon: '🛡',
        clip: 'polygon(50% 0%, 100% 20%, 100% 65%, 50% 100%, 0% 65%, 0% 20%)',
        hitDirs: ['up', 'down'],
        desc: 'Defensive — P-Peace, P-AOS, P-Warden pathways',
    },
    arrowRight: {
        name: 'Arrow Right',
        icon: '▶',
        clip: 'polygon(0% 50%, 50% 0%, 50% 35%, 100% 35%, 100% 65%, 50% 65%, 50% 100%)',
        hitDirs: ['left', 'right'],
        desc: 'Forward action — primary CTA',
    },
    arrowLeft: {
        name: 'Arrow Left',
        icon: '◀',
        clip: 'polygon(50% 0%, 50% 35%, 0% 35%, 0% 65%, 50% 65%, 50% 100%, 100% 50%)',
        hitDirs: ['left', 'right'],
        desc: 'Back action — secondary/return',
    },
    cross: {
        name: 'Cross/Plus',
        icon: '✚',
        clip: 'polygon(35% 0%, 65% 0%, 65% 35%, 100% 35%, 100% 65%, 65% 65%, 65% 100%, 35% 100%, 35% 65%, 0% 65%, 0% 35%, 35% 35%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Four-directional — P-Justice, P-Commissioner pathways',
    },
    parallelogram: {
        name: 'Parallelogram',
        icon: '▱',
        clip: 'polygon(25% 0%, 100% 0%, 75% 100%, 0% 100%)',
        hitDirs: ['left', 'right'],
        desc: 'Dynamic slant — P-Intel, P-Tax pathways',
    },
    // --- PATHWAY-SPECIFIC SHAPES ---
    justiceScales: {
        name: 'Justice Scales',
        icon: '⚖',
        clip: 'polygon(50% 0%, 55% 30%, 100% 30%, 100% 45%, 55% 45%, 60% 100%, 40% 100%, 45% 45%, 0% 45%, 0% 30%, 45% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Justice scales — P-Justice pathway exclusive',
    },
    shadowCrescent: {
        name: 'Shadow Crescent',
        icon: '🌙',
        clip: 'circle(50% at 60% 50%)',
        hitDirs: ['left', 'right'],
        desc: 'Crescent moon — P-Shadow pathway exclusive',
    },
    skull: {
        name: 'Skull',
        icon: '💀',
        clip: 'polygon(25% 0%, 75% 0%, 100% 30%, 100% 70%, 75% 100%, 65% 85%, 35% 85%, 25% 100%, 0% 70%, 0% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Death/syndicate — P-Syndicate, P-Boss pathways',
    },
    ledgerBook: {
        name: 'Ledger Book',
        icon: '📒',
        clip: 'polygon(10% 0%, 90% 0%, 100% 15%, 100% 100%, 0% 100%, 0% 15%)',
        hitDirs: ['up', 'down'],
        desc: 'Book of records — P-Ledger, P-Tax, P-Forensic pathways',
    },
    fingerprint: {
        name: 'Fingerprint',
        icon: '🖐',
        clip: 'polygon(50% 0%, 80% 15%, 100% 50%, 80% 85%, 50% 100%, 20% 85%, 0% 50%, 20% 15%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Identity/forensic — P-Forensic, P-Intel pathways',
    },
    dove: {
        name: 'Dove',
        icon: '🕊',
        clip: 'polygon(50% 0%, 70% 20%, 100% 40%, 80% 60%, 60% 100%, 50% 80%, 40% 100%, 20% 60%, 0% 40%, 30% 20%)',
        hitDirs: ['up', 'down'],
        desc: 'Peace — P-Peace pathway exclusive',
    },
    flame: {
        name: 'Flame',
        icon: '🔥',
        clip: 'polygon(50% 0%, 70% 20%, 100% 50%, 80% 80%, 60% 100%, 50% 80%, 40% 100%, 20% 80%, 0% 50%, 30% 20%)',
        hitDirs: ['up', 'down'],
        desc: 'Energy/passion — P-Intel, P-Justice pathways',
    },
    chain: {
        name: 'Chain Link',
        icon: '⛓',
        clip: 'polygon(30% 0%, 70% 0%, 100% 30%, 100% 70%, 70% 100%, 30% 100%, 0% 70%, 0% 30%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Lockdown/bonds — P-Lockdown pathway exclusive',
    },
    crown: {
        name: 'Crown',
        icon: '👑',
        clip: 'polygon(0% 100%, 0% 50%, 20% 70%, 35% 30%, 50% 60%, 65% 30%, 80% 70%, 100% 50%, 100% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Authority — P-Boss, P-Commissioner, P-Governor pathways',
    },
    eye: {
        name: 'Eye',
        icon: '👁',
        clip: 'ellipse(50% 35% at 50% 50%)',
        hitDirs: ['left', 'right'],
        desc: 'Surveillance — P-Intel, P-Shadow pathways',
    },
    mask: {
        name: 'Theater Mask',
        icon: '🎭',
        clip: 'polygon(20% 0%, 80% 0%, 100% 30%, 90% 70%, 70% 100%, 50% 90%, 30% 100%, 10% 70%, 0% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Deception — P-Shadow, P-Syndicate pathways',
    },
    gavel: {
        name: 'Gavel',
        icon: '🔨',
        clip: 'polygon(40% 0%, 60% 0%, 70% 30%, 100% 50%, 70% 70%, 60% 100%, 40% 100%, 30% 70%, 0% 50%, 30% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Judgment — P-Justice, P-Commissioner pathways',
    },
    beast: {
        name: 'Beast Claw',
        icon: '🦁',
        clip: 'polygon(50% 0%, 65% 25%, 100% 20%, 80% 50%, 100% 80%, 65% 75%, 50% 100%, 35% 75%, 0% 80%, 20% 50%, 0% 20%, 35% 25%)',
        hitDirs: ['up', 'down'],
        desc: 'Power/beast — P-Boss, P-Syndicate pathways',
    },
    serpent: {
        name: 'Serpent',
        icon: '🐍',
        clip: 'polygon(30% 0%, 70% 0%, 100% 20%, 80% 50%, 100% 80%, 70% 100%, 30% 100%, 0% 80%, 20% 50%, 0% 20%)',
        hitDirs: ['left', 'right'],
        desc: 'Cunning/underworld — P-Shadow, P-Syndicate pathways',
    },
    atom: {
        name: 'Atom',
        icon: '⚛',
        clip: 'circle(40% at 50% 50%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Science/forensic — P-Forensic, P-Intel pathways',
    },
    heart: {
        name: 'Heart',
        icon: '❤',
        clip: 'polygon(50% 100%, 0% 35%, 0% 15%, 25% 0%, 50% 25%, 75% 0%, 100% 15%, 100% 35%)',
        hitDirs: ['up', 'down'],
        desc: 'Domestic/love — P-Peace, Domestic system',
    },
    house: {
        name: 'House',
        icon: '🏠',
        clip: 'polygon(50% 0%, 100% 40%, 100% 100%, 0% 100%, 0% 40%)',
        hitDirs: ['up', 'down'],
        desc: 'Home/domestic — Domestic system, Orphans',
    },
    coins: {
        name: 'Coins',
        icon: '🪙',
        clip: 'circle(40% at 50% 60%)',
        hitDirs: ['up', 'down'],
        desc: 'Economy — P-Ledger, P-Tax, Markets',
    },
    chart: {
        name: 'Chart',
        icon: '📊',
        clip: 'polygon(0% 100%, 0% 60%, 25% 60%, 25% 40%, 50% 40%, 50% 20%, 75% 20%, 75% 50%, 100% 50%, 100% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Analytics — P-Intel, P-Tax, P-Forensic',
    },
    people: {
        name: 'People',
        icon: '👥',
        clip: 'polygon(25% 0%, 75% 0%, 100% 30%, 100% 100%, 0% 100%, 0% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Social/rivalry — Rivalry system, Clubs',
    },
    sword: {
        name: 'Sword',
        icon: '⚔',
        clip: 'polygon(50% 0%, 55% 60%, 100% 70%, 100% 80%, 50% 100%, 0% 80%, 0% 70%, 45% 60%)',
        hitDirs: ['up', 'down'],
        desc: 'Combat/battle — Battle system, Tournaments',
    },
    shieldAlt: {
        name: 'Battle Shield',
        icon: '🛡',
        clip: 'polygon(50% 0%, 100% 15%, 100% 60%, 50% 100%, 0% 60%, 0% 15%)',
        hitDirs: ['up', 'down'],
        desc: 'Defense — P-AOS, P-Lockdown, P-Peace',
    },
    flag: {
        name: 'Flag',
        icon: '🚩',
        clip: 'polygon(0% 0%, 100% 0%, 100% 100%, 50% 75%, 0% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Territory/governance — Territory, Governor',
    },
    key: {
        name: 'Key',
        icon: '🔑',
        clip: 'polygon(20% 30%, 80% 30%, 80% 45%, 100% 45%, 100% 55%, 80% 55%, 80% 70%, 60% 70%, 60% 55%, 20% 55%)',
        hitDirs: ['left', 'right'],
        desc: 'Access/unlock — Unlock system, DLC',
    },
    rocket: {
        name: 'Rocket',
        icon: '🚀',
        clip: 'polygon(50% 0%, 70% 30%, 100% 60%, 80% 100%, 50% 80%, 20% 100%, 0% 60%, 30% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Launch/creator — Creator Economy, Launches',
    },
    dna: {
        name: 'DNA',
        icon: '🧬',
        clip: 'polygon(50% 0%, 60% 20%, 70% 10%, 80% 30%, 90% 20%, 100% 50%, 90% 80%, 80% 70%, 70% 90%, 60% 80%, 50% 100%, 40% 80%, 30% 90%, 20% 70%, 10% 80%, 0% 50%, 10% 20%, 20% 30%, 30% 10%, 40% 20%)',
        hitDirs: ['up', 'down'],
        desc: 'Breeding/genetics — Breeding system, Pets',
    },
    ghost: {
        name: 'Ghost',
        icon: '👻',
        clip: 'polygon(20% 0%, 80% 0%, 100% 20%, 100% 80%, 80% 100%, 70% 85%, 60% 100%, 50% 85%, 40% 100%, 30% 85%, 20% 100%, 0% 80%, 0% 20%)',
        hitDirs: ['up', 'down'],
        desc: 'Stealth — P-Shadow, Underworld, Ghost Protocol item',
    },
    bomb: {
        name: 'Bomb',
        icon: '💣',
        clip: 'circle(40% at 50% 60%)',
        hitDirs: ['up', 'down'],
        desc: 'Sabotage — Underworld, Sabotage contracts',
    },
    syringe: {
        name: 'Syringe',
        icon: '💉',
        clip: 'polygon(40% 0%, 60% 0%, 60% 40%, 100% 40%, 100% 60%, 60% 60%, 60% 100%, 40% 100%, 40% 60%, 0% 60%, 0% 40%, 40% 40%)',
        hitDirs: ['up', 'down'],
        desc: 'Enhancement — Mutation, Items, P-Forensic',
    },
    recycle: {
        name: 'Recycle',
        icon: '♻',
        clip: 'polygon(50% 0%, 80% 20%, 100% 50%, 80% 80%, 50% 100%, 20% 80%, 0% 50%, 20% 20%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Industrial loop — Industrial system, Economy',
    },
    scalesTri: {
        name: 'Triangle Scales',
        icon: '⚖',
        clip: 'polygon(50% 0%, 100% 100%, 0% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Balance/justice — Justice system, P-Justice',
    },
    network: {
        name: 'Network',
        icon: '🕸',
        clip: 'polygon(50% 0%, 100% 25%, 100% 75%, 50% 100%, 0% 75%, 0% 25%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Connections — P-Intel, ArcNet, Matchmaking',
    },
    hourglass: {
        name: 'Hourglass',
        icon: '⏳',
        clip: 'polygon(20% 0%, 80% 0%, 50% 50%, 80% 100%, 20% 100%, 50% 50%)',
        hitDirs: ['up', 'down'],
        desc: 'Time/season — Seasonal events, Tournaments',
    },
    dice: {
        name: 'Dice',
        icon: '🎲',
        clip: 'polygon(10% 10%, 90% 10%, 90% 90%, 10% 90%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Chance/luck — Gambling, Events, Quick Play',
    },
    trophy: {
        name: 'Trophy',
        icon: '🏆',
        clip: 'polygon(30% 0%, 70% 0%, 80% 30%, 100% 50%, 80% 80%, 20% 80%, 0% 50%, 20% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Achievement — Achievements, Tournaments',
    },
    medal: {
        name: 'Medal',
        icon: '🎖',
        clip: 'circle(40% at 50% 50%)',
        hitDirs: ['up', 'down'],
        desc: 'Honor — Achievements, Career progression',
    },
    ribbon: {
        name: 'Ribbon',
        icon: '🎗',
        clip: 'polygon(30% 0%, 70% 0%, 100% 30%, 70% 100%, 50% 70%, 30% 100%, 0% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Cause/compliance — Compliance, Justice',
    },
    lock: {
        name: 'Lock',
        icon: '🔒',
        clip: 'polygon(20% 40%, 80% 40%, 80% 100%, 20% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Security — P-Lockdown, Counterfeit, Security',
    },
    unlock: {
        name: 'Unlock',
        icon: '🔓',
        clip: 'polygon(20% 40%, 80% 40%, 80% 100%, 20% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Access — Unlock system, DLC, Features',
    },
    bell: {
        name: 'Bell',
        icon: '🔔',
        clip: 'polygon(50% 0%, 100% 40%, 100% 70%, 80% 100%, 20% 100%, 0% 70%, 0% 40%)',
        hitDirs: ['up', 'down'],
        desc: 'Notification — System messages, Events',
    },
    megaphone: {
        name: 'Megaphone',
        icon: '📢',
        clip: 'polygon(0% 30%, 30% 0%, 100% 0%, 100% 100%, 30% 100%, 0% 70%)',
        hitDirs: ['left', 'right'],
        desc: 'Broadcast — Admin, Governance, Ads',
    },
    palette: {
        name: 'Palette',
        icon: '🎨',
        clip: 'circle(50% at 50% 50%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Creator — Creator Economy, DLC, Styling',
    },
    gear: {
        name: 'Gear',
        icon: '⚙',
        clip: 'polygon(40% 0%, 60% 0%, 60% 15%, 80% 15%, 80% 0%, 100% 0%, 100% 20%, 85% 20%, 85% 40%, 100% 40%, 100% 60%, 85% 60%, 85% 80%, 100% 80%, 100% 100%, 80% 100%, 80% 85%, 60% 85%, 60% 100%, 40% 100%, 40% 85%, 20% 85%, 20% 100%, 0% 100%, 0% 80%, 15% 80%, 15% 60%, 0% 60%, 0% 40%, 15% 40%, 15% 20%, 0% 20%, 0% 0%, 20% 0%, 20% 15%, 40% 15%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Settings — Infrastructure, Gaming OS, Config',
    },
    robot: {
        name: 'Robot',
        icon: '🤖',
        clip: 'polygon(20% 20%, 80% 20%, 80% 80%, 20% 80%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'AI/Local — Local LLM, AI Citizens',
    },
    brain: {
        name: 'Brain',
        icon: '🧠',
        clip: 'circle(45% at 50% 50%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Intelligence — P-Intel, Local LLM, AI',
    },
    chartLine: {
        name: 'Line Chart',
        icon: '📈',
        clip: 'polygon(0% 100%, 0% 60%, 25% 60%, 25% 40%, 50% 40%, 50% 20%, 75% 20%, 75% 50%, 100% 50%, 100% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Growth — Markets, Economy, Career XP',
    },
    bank: {
        name: 'Bank',
        icon: '🏦',
        clip: 'polygon(0% 40%, 50% 0%, 100% 40%, 100% 100%, 0% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Finance — Economy, Loans, P-Ledger',
    },
    courthouse: {
        name: 'Courthouse',
        icon: '🏛',
        clip: 'polygon(0% 30%, 50% 0%, 100% 30%, 100% 100%, 0% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Justice — Justice system, P-Justice, P-Commissioner',
    },
    church: {
        name: 'Church',
        icon: '⛪',
        clip: 'polygon(40% 0%, 60% 0%, 60% 30%, 100% 30%, 100% 100%, 0% 100%, 0% 30%, 40% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Faith — Faith system, Church, P-Peace',
    },
    mosque: {
        name: 'Mosque',
        icon: '🕌',
        clip: 'polygon(40% 0%, 60% 0%, 60% 20%, 100% 40%, 100% 100%, 0% 100%, 0% 40%, 40% 20%)',
        hitDirs: ['up', 'down'],
        desc: 'Faith — Faith system, 24 religions',
    },
    temple: {
        name: 'Temple',
        icon: '🕍',
        clip: 'polygon(30% 0%, 70% 0%, 100% 30%, 100% 100%, 0% 100%, 0% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Faith — Faith system, P-Peace, P-Justice',
    },
    tent: {
        name: 'Tent',
        icon: '⛺',
        clip: 'polygon(50% 0%, 100% 100%, 0% 100%)',
        hitDirs: ['up', 'down'],
        desc: 'Events — Seasonal events, Treasure hunts',
    },
    campfire: {
        name: 'Campfire',
        icon: '🔥',
        clip: 'polygon(50% 0%, 80% 50%, 100% 100%, 0% 100%, 20% 50%)',
        hitDirs: ['up', 'down'],
        desc: 'Social — Clubs, Rivalry, Community',
    },
    map: {
        name: 'Map',
        icon: '🗺',
        clip: 'polygon(0% 0%, 100% 0%, 100% 100%, 0% 100%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Territory — Territory, Regions, 3D World',
    },
    globe: {
        name: 'Globe',
        icon: '🌍',
        clip: 'circle(50% at 50% 50%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'World — World Dashboard, 3D World, Multi-chain',
    },
    satellite: {
        name: 'Satellite',
        icon: '🛰',
        clip: 'polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Intel — P-Intel, Surveillance, ArcNet',
    },
    radar: {
        name: 'Radar',
        icon: '📡',
        clip: 'circle(50% at 50% 50%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Detection — Counterfeit, Justice, P-Intel',
    },
    handcuffs: {
        name: 'Handcuffs',
        icon: '⛓',
        clip: 'polygon(20% 30%, 40% 30%, 40% 70%, 20% 70%)',
        hitDirs: ['left', 'right'],
        desc: 'Arrest — Justice, P-Lockdown, P-AOS',
    },
    magnifier: {
        name: 'Magnifier',
        icon: '🔍',
        clip: 'circle(35% at 45% 45%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Investigate — P-Forensic, P-Intel, Justice',
    },
    scalesBalanced: {
        name: 'Balanced Scales',
        icon: '⚖',
        clip: 'polygon(50% 0%, 55% 35%, 100% 35%, 100% 50%, 55% 50%, 50% 100%, 45% 50%, 0% 50%, 0% 35%, 45% 35%)',
        hitDirs: ['up', 'down'],
        desc: 'Justice — Justice system, P-Justice, P-Commissioner',
    },
    pill: {
        name: 'Pill',
        icon: '💊',
        clip: 'ellipse(50% 25% at 50% 50%)',
        hitDirs: ['left', 'right'],
        desc: 'Enhancement — Items, Mutation, Mood Catalyst',
    },
    sparkles: {
        name: 'Sparkles',
        icon: '✨',
        clip: 'polygon(50% 0%, 55% 45%, 100% 50%, 55% 55%, 50% 100%, 45% 55%, 0% 50%, 45% 45%)',
        hitDirs: ['up', 'down'],
        desc: 'Special — Achievements, Rewards, Premium',
    },
    clover: {
        name: 'Clover',
        icon: '🍀',
        clip: 'circle(30% at 35% 35%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Luck — Events, Gambling, Treasure',
    },
    spade: {
        name: 'Spade',
        icon: '♠',
        clip: 'polygon(50% 0%, 100% 50%, 75% 100%, 25% 100%, 0% 50%)',
        hitDirs: ['up', 'down'],
        desc: 'Strategy — P-Shadow, Underworld, Cards',
    },
    heartCard: {
        name: 'Heart Card',
        icon: '♥',
        clip: 'polygon(50% 100%, 0% 35%, 0% 15%, 25% 0%, 50% 25%, 75% 0%, 100% 15%, 100% 35%)',
        hitDirs: ['up', 'down'],
        desc: 'Domestic — Marriage, Breeding, Pets',
    },
    club: {
        name: 'Club Card',
        icon: '♣',
        clip: 'polygon(50% 0%, 80% 30%, 80% 70%, 50% 100%, 20% 70%, 20% 30%)',
        hitDirs: ['up', 'down'],
        desc: 'Social — Clubs, Community, Rivalry',
    },
    joker: {
        name: 'Joker',
        icon: '🃏',
        clip: 'polygon(20% 0%, 80% 0%, 100% 20%, 100% 80%, 80% 100%, 20% 100%, 0% 80%, 0% 20%)',
        hitDirs: ['up', 'down', 'left', 'right'],
        desc: 'Wildcard — Events, Gambling, Quick Play',
    },
};

// ============================================================
// 2. SIZE LIBRARY
// ============================================================
const SIZES = {
    xs:  { name: 'Tiny',        px: 32,  font: 8,  icon: 14, hitPad: 4  },
    sm:  { name: 'Small',       px: 48,  font: 9,  icon: 18, hitPad: 4  },
    md:  { name: 'Medium',      px: 72,  font: 11, icon: 24, hitPad: 6  },
    lg:  { name: 'Large',       px: 96,  font: 13, icon: 32, hitPad: 6  },
    xl:  { name: 'Extra Large', px: 128, font: 15, icon: 40, hitPad: 8  },
    xxl: { name: 'Huge',        px: 160, font: 18, icon: 48, hitPad: 8  },
};

// ============================================================
// 3. GRID LAYOUT LIBRARY
// ============================================================
const GRIDS = {
    grid: {
        name: 'Standard Grid',
        desc: 'Row-major grid — 2D directional nav',
        navType: '2d',
        compute: function(items, containerW, containerH, layout) {
            const cols = layout.gridCols || Math.ceil(Math.sqrt(items.length));
            const rows = Math.ceil(items.length / cols);
            const gap = parseInt(layout.gridGap) || 12;
            const cellW = (containerW - gap * (cols + 1)) / cols;
            const cellH = (containerH - gap * (rows + 1)) / rows;
            return items.map((item, i) => {
                const col = i % cols;
                const row = Math.floor(i / cols);
                return {
                    x: gap + col * (cellW + gap),
                    y: gap + row * (cellH + gap),
                    w: cellW,
                    h: cellH,
                };
            });
        },
    },
    circle: {
        name: 'Radial Wheel',
        desc: 'Items on circle circumference — rotate CW/CCW',
        navType: 'radial',
        compute: function(items, containerW, containerH, layout) {
            const cx = containerW / 2;
            const cy = containerH / 2;
            const r = Math.min(containerW, containerH) * (parseFloat(layout.gridRadius) || 0.4);
            const rot = (parseFloat(layout.gridRotation) || 0) * Math.PI / 180;
            return items.map((item, i) => {
                const angle = rot + (2 * Math.PI * i) / items.length - Math.PI / 2;
                const size = SIZES[item.size || 'md'].px;
                return {
                    x: cx + r * Math.cos(angle) - size / 2,
                    y: cy + r * Math.sin(angle) - size / 2,
                    w: size,
                    h: size,
                };
            });
        },
    },
    spokes: {
        // HUB & SPOKES — the first item is the hub at the centre, the rest ride the
        // spoke ring. DISTINCT from `circle` (a plain ring of peers) by having a
        // centre, which is what makes an unambiguous "spin the wheel" gesture
        // possible: the ring turns around a fixed hub.
        // POSITION-ORDER CONTRACT: the returned array is index-aligned with `items`
        // (hub first, then the ring in order) because every consumer paints
        // `positions[i]` over `items[i]`.
        name: 'Hub & Spokes',
        desc: 'First item is the hub, the rest ride the spokes — drag to spin',
        navType: 'radial',
        compute: function (items, containerW, containerH, layout) {
            const cx = containerW / 2;
            const cy = containerH / 2;
            const r = Math.min(containerW, containerH) * (parseFloat(layout.gridRadius) || 0.4);
            const rot = (parseFloat(layout.gridRotation) || 0) * Math.PI / 180;
            const out = [];
            const hub = items[0];
            if (hub) {
                const hs = SIZES[hub.size || 'md'].px;
                out.push({ x: cx - hs / 2, y: cy - hs / 2, w: hs, h: hs, hub: true });
            }
            const ring = items.slice(1);
            ring.forEach((item, i) => {
                const angle = rot + (2 * Math.PI * i) / (ring.length || 1) - Math.PI / 2;
                const size = SIZES[item.size || 'md'].px;
                out.push({
                    x: cx + r * Math.cos(angle) - size / 2,
                    y: cy + r * Math.sin(angle) - size / 2,
                    w: size,
                    h: size,
                    spoke: i,
                });
            });
            return out;
        },
    },
    triangle: {
        name: 'Pyramid',
        desc: '1-2-3-4... triangle — layer by layer',
        navType: 'layered',
        compute: function(items, containerW, containerH, layout) {
            const positions = [];
            let itemIdx = 0;
            let row = 1;
            const gap = parseInt(layout.gridGap) || 12;
            while (itemIdx < items.length) {
                const count = Math.min(row, items.length - itemIdx);
                const rowWidth = count * 72 + (count - 1) * gap;
                const startX = (containerW - rowWidth) / 2;
                const y = 60 + (row - 1) * (72 + gap);
                for (let col = 0; col < count; col++) {
                    positions.push({
                        x: startX + col * (72 + gap),
                        y: y,
                        w: 72,
                        h: 72,
                    });
                    itemIdx++;
                }
                row++;
            }
            return positions;
        },
    },
    cross: {
        name: 'Plus/Cross',
        desc: 'Center + 4 cardinal arms — D-pad perfect',
        navType: 'cardinal',
        compute: function(items, containerW, containerH, layout) {
            const cx = containerW / 2;
            const cy = containerH / 2;
            const r = Math.min(containerW, containerH) * 0.3;
            const dirs = [
                { x: 0, y: -r },
                { x: r, y: 0 },
                { x: 0, y: r },
                { x: -r, y: 0 },
            ];
            return items.slice(0, 5).map((item, i) => {
                const size = SIZES[item.size || 'md'].px;
                if (i === 0) {
                    return { x: cx - size / 2, y: cy - size / 2, w: size, h: size };
                }
                const d = dirs[i - 1] || dirs[0];
                return { x: cx + d.x - size / 2, y: cy + d.y - size / 2, w: size, h: size };
            });
        },
    },
    arc: {
        name: 'Arc/Semicircle',
        desc: 'Items in semicircle — left/right sweep',
        navType: 'sweep',
        compute: function(items, containerW, containerH, layout) {
            const cx = containerW / 2;
            const cy = containerH * 0.7;
            const r = Math.min(containerW, containerH) * (parseFloat(layout.gridRadius) || 0.4);
            return items.map((item, i) => {
                const angle = Math.PI + (Math.PI * i) / (items.length - 1 || 1);
                const size = SIZES[item.size || 'md'].px;
                return {
                    x: cx + r * Math.cos(angle) - size / 2,
                    y: cy + r * Math.sin(angle) - size / 2,
                    w: size,
                    h: size,
                };
            });
        },
    },
    diamond: {
        name: 'Diamond',
        desc: 'Items in diamond pattern — diagonal nav',
        navType: 'diagonal',
        compute: function(items, containerW, containerH, layout) {
            const cx = containerW / 2;
            const cy = containerH / 2;
            const r = Math.min(containerW, containerH) * 0.35;
            const dirs = [
                { x: 0, y: -r },
                { x: r, y: 0 },
                { x: 0, y: r },
                { x: -r, y: 0 },
            ];
            return items.slice(0, 4).map((item, i) => {
                const size = SIZES[item.size || 'md'].px;
                const d = dirs[i] || { x: 0, y: 0 };
                return { x: cx + d.x - size / 2, y: cy + d.y - size / 2, w: size, h: size };
            });
        },
    },
    linearH: {
        name: 'Horizontal Line',
        desc: 'Single row — left/right only',
        navType: 'linear',
        compute: function(items, containerW, containerH, layout) {
            const gap = parseInt(layout.gridGap) || 12;
            const totalW = items.reduce((sum, item) => sum + SIZES[item.size || 'md'].px, 0) + gap * (items.length - 1);
            let x = (containerW - totalW) / 2;
            const y = containerH / 2;
            return items.map(item => {
                const size = SIZES[item.size || 'md'].px;
                const pos = { x: x, y: y - size / 2, w: size, h: size };
                x += size + gap;
                return pos;
            });
        },
    },
    linearV: {
        name: 'Vertical Line',
        desc: 'Single column — up/down only',
        navType: 'linear',
        compute: function(items, containerW, containerH, layout) {
            const gap = parseInt(layout.gridGap) || 12;
            const totalH = items.reduce((sum, item) => sum + SIZES[item.size || 'md'].px, 0) + gap * (items.length - 1);
            const x = containerW / 2;
            let y = (containerH - totalH) / 2;
            return items.map(item => {
                const size = SIZES[item.size || 'md'].px;
                const pos = { x: x - size / 2, y: y, w: size, h: size };
                y += size + gap;
                return pos;
            });
        },
    },
    freeform: {
        name: 'Free Position',
        desc: 'Absolute coordinates — custom tab order',
        navType: 'freeform',
        compute: function(items, containerW, containerH, layout) {
            return items.map(item => {
                const override = layout.buttonOverrides?.[item.id] || {};
                const size = SIZES[override.size || item.size || 'md'].px;
                return {
                    x: override.x ? parseFloat(override.x) * containerW / 100 : 50,
                    y: override.y ? parseFloat(override.y) * containerH / 100 : 50,
                    w: size,
                    h: size,
                };
            });
        },
    },
};

// ============================================================
// 4. PATHWAY-SHAPE MAPPING
// ============================================================
const PATHWAY_SHAPES = {
    'P-Shadow':      'shadowCrescent',
    'P-Lockdown':    'chain',
    'P-Ledger':      'ledgerBook',
    'P-Syndicate':   'skull',
    'P-Tax':         'coins',
    'P-Peace':       'dove',
    'P-Intel':       'satellite',
    'P-Justice':     'justiceScales',
    'P-AOS':         'shieldAlt',
    'P-Commissioner':'crown',
    'P-Forensic':    'fingerprint',
    'P-Boss':        'beast',
};

// ============================================================
// 5. SYSTEM-SHAPE MAPPING (default shapes per system)
// ============================================================
const SYSTEM_SHAPES = {
    world:        'globe',
    quickplay:    'dice',
    profile:      'circle',
    tutorial:     'house',
    career:       'chart',
    battle:       'sword',
    justice:      'scalesBalanced',
    underworld:   'mask',
    faith:        'church',
    domestic:     'heart',
    economy:      'bank',
    markets:      'chartLine',
    governance:   'courthouse',
    territory:    'map',
    rivalry:      'people',
    pets:         'dna',
    breeding:     'dna',
    items:        'pill',
    mutation:     'syringe',
    clubs:        'campfire',
    creator:      'palette',
    dlc:          'key',
    achievements: 'trophy',
    leaderboard:  'medal',
    events:       'tent',
    seasonal:     'hourglass',
    tournament:   'trophy',
    contracts:    'lock',
    counterfeit:  'magnifier',
    blackmarket:  'ghost',
    loans:        'bank',
    dividends:    'coins',
    orphans:      'house',
    vehicles:     'rocket',
    localmodel:   'brain',
    ai:           'robot',
    industrial:   'recycle',
    compliance:   'ribbon',
    report:       'megaphone',
    settings:     'gear',
    ads:          'megaphone',
    launches:     'rocket',
    church:       'church',
    npc:          'sparkles',
};

// ============================================================
// 6. CONTROLLER NAVIGATION
// ============================================================
const CONTROLLER_MAP = {
    grid:     { up: [-1, 0], down: [1, 0], left: [0, -1], right: [0, 1] },
    circle:   { up: 'center', down: 'outer', left: 'cw', right: 'ccw' },
    spokes:   { up: 'hub', down: 'outer', left: 'cw', right: 'ccw' },
    triangle: { up: 'prev-layer', down: 'next-layer', left: 'sibling-prev', right: 'sibling-next' },
    cross:    { up: 0, right: 1, down: 2, left: 3 },
    arc:      { left: 'prev', right: 'next' },
    diamond:  { up: 0, right: 1, down: 2, left: 3 },
    linearH:  { left: 'prev', right: 'next' },
    linearV:  { up: 'prev', down: 'next' },
    freeform: { up: 'tab-prev', down: 'tab-next' },
};

// Grids that place items on a ring around a centre. ONE owner for this list: the
// customization panel used to spell `['circle','arc']` inline, so a new radial grid
// would silently lose its Radius control.
const RADIAL_GRIDS = Object.freeze(['circle', 'arc', 'spokes']);

// Grids that lay items out in FLOW (no absolute coordinates). ONE owner for this
// too: both the panel and the constellation tested `['grid','linear-h','linear-v']`
// — ids that do not exist (the real keys are `linearH`/`linearV`) — so those two
// layouts were painted with absolute positioning they do not use.
const FLOW_GRIDS = Object.freeze(['grid', 'linearH', 'linearV']);

// ============================================================
// 7. PUBLIC API
// ============================================================

function getShapeForSystem(systemId) {
    return SYSTEM_SHAPES[systemId] || 'circle';
}

function getPathwayShape(pathwayId) {
    return PATHWAY_SHAPES[pathwayId] || 'circle';
}

function getSize(sizeId) {
    return SIZES[sizeId] || SIZES.md;
}

function getGrid(gridId) {
    return GRIDS[gridId] || GRIDS.grid;
}

function computePositions(gridType, items, containerW, containerH, layout) {
    const grid = GRIDS[gridType] || GRIDS.grid;
    return grid.compute(items, containerW, containerH, layout);
}

function getClipPath(shapeId) {
    const shape = SHAPES[shapeId];
    return shape ? shape.clip : SHAPES.circle.clip;
}

function getControllerNav(gridType) {
    return CONTROLLER_MAP[gridType] || CONTROLLER_MAP.grid;
}

function getAllShapeIds() {
    return Object.keys(SHAPES);
}

function getAllGridIds() {
    return Object.keys(GRIDS);
}

function isRadialGrid(gridId) {
    return RADIAL_GRIDS.indexOf(gridId) >= 0;
}

// true when the layout paints items at computed coordinates (everything except the
// flow layouts), so consumers know whether to set `position:relative` + a height.
function isPositionedGrid(gridId) {
    return FLOW_GRIDS.indexOf(gridId) < 0;
}

function getAllSizeIds() {
    return Object.keys(SIZES);
}


export {
    SHAPES,
    SIZES,
    GRIDS,
    RADIAL_GRIDS,
    FLOW_GRIDS,
    PATHWAY_SHAPES,
    SYSTEM_SHAPES,
    CONTROLLER_MAP,
    getShapeForSystem,
    getPathwayShape,
    getSize,
    getGrid,
    computePositions,
    getClipPath,
    getControllerNav,
    getAllShapeIds,
    getAllGridIds,
    getAllSizeIds,
    isRadialGrid,
    isPositionedGrid,
};


