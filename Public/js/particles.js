// Inside Public/js/particles.js

/**
 * FX Subsystem for Virtualbabes Arena
 * Handles high-performance canvas particles for captures and victories.
 */

let particles = [];
let particleCanvas = null;
let particleCtx = null;
let particleAnimationId = null;

/**
 * Initializes the particle system and scales the canvas to fit the battle grid.
 */
export function initParticleSystem() {
    particleCanvas = document.getElementById("particle-canvas");
    if (!particleCanvas) return;

    particleCtx = particleCanvas.getContext("2d");
    
    const resize = () => {
        const board = document.getElementById("board-container");
        if (board) {
            const rect = board.getBoundingClientRect();
            // Match logical resolution to physical pixels for precision scaling
            particleCanvas.width = rect.width;
            particleCanvas.height = rect.height;
            // Sync position with the glassmorphism grid container
            particleCanvas.style.left = board.offsetLeft + "px";
            particleCanvas.style.top = board.offsetTop + "px";
        }
    };

    window.addEventListener('resize', resize);
    resize();
    
    console.log("[PARTICLES] Visual Effects Subsystem Ready.");
}

/**
 * Internal loop to process particle physics and drawing.
 */
export function animateParticles() {
    if (!particleCtx) return;

    particleCtx.clearRect(0, 0, particleCanvas.width, particleCanvas.height);

    for (let i = particles.length - 1; i >= 0; i--) {
        const p = particles[i];

        // 1. Apply Physics based on Particle Type
        p.x += p.vx;
        p.y += p.vy;
        
        if (p.type === "gravity") {
            p.vy += 0.15; // Standard gravity for rewards/sparks
        } else if (p.type === "ambient") {
            p.vx += Math.sin(Date.now() / 500) * 0.05; // Drifting motes
        }

        p.life--;
        const alpha = p.life / p.initialLife;
        
        // Apply Wanted Glitch: High infamy causes visual distortion and static jitter
        let drawX = p.x;
        let drawY = p.y;
        if ((p.wantedLevel > 10 || p.isGlitchy) && Math.random() > 0.94) {
            drawX += (Math.random() - 0.5) * 8;
        }

        // 2. Specialized Rendering
        if (p.type === "data-line") {
            // Heist digital lines
            let alphaMod = alpha * 0.6;
            if (p.isGlitchy && Math.random() > 0.8) alphaMod = 1.0; // Flickering

            particleCtx.beginPath();
            particleCtx.moveTo(drawX, drawY);
            particleCtx.lineTo(drawX, drawY - p.size * 10);
            particleCtx.strokeStyle = `rgba(${p.color.r}, ${p.color.g}, ${p.color.b}, ${alphaMod})`;
            particleCtx.lineWidth = p.isGlitchy ? 2 : 1;
            particleCtx.stroke();
            if (p.life <= 0) particles.splice(i, 1);
            continue;
        }

        // Draw spark line for tactical 'impact' feel
        particleCtx.beginPath();
        particleCtx.moveTo(drawX, drawY);
        particleCtx.lineTo(drawX - p.vx * 1.5, drawY - p.vy * 1.5);
        particleCtx.strokeStyle = `rgba(${p.color.r}, ${p.color.g}, ${p.color.b}, ${alpha})`;
        particleCtx.lineWidth = p.size;
        particleCtx.lineCap = "round";
        particleCtx.stroke();

        // Apply Rank Trail: Diamond tier particles leave a secondary energy trail
        if (p.hasTrail && p.life > p.initialLife * 0.5) {
            particleCtx.beginPath();
            particleCtx.moveTo(drawX - p.vx * 1.5, drawY - p.vy * 1.5);
            particleCtx.lineTo(drawX - p.vx * 4, drawY - p.vy * 4);
            particleCtx.strokeStyle = `rgba(0, 242, 254, ${alpha * 0.3})`; // Neon Cyan trail
            particleCtx.lineWidth = p.size * 0.5;
            particleCtx.stroke();
        }

        if (p.life <= 0) {
            particles.splice(i, 1);
        }
    }

    if (particles.length > 0) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    } else {
        particleAnimationId = null;
    }
}

/**
 * Spawns a burst of sparks at the center of a captured slot.
 * @param {number} gridIndex - The 0-8 slot position.
 * @param {number} owner - The new owner (0=Cyan, 1=Red).
 */
export function triggerCaptureParticles(gridIndex, owner) {
    if (!particleCtx) return;

    // Access authoritative game state to derive card and player context
    const state = window.GetGameState();
    const card = state.board[gridIndex];
    if (!card) return;

    const boardContainer = document.getElementById("board-container");
    if (!boardContainer) return;

    const slotSize = boardContainer.offsetWidth / 3;
    const col = gridIndex % 3;
    const row = Math.floor(gridIndex / 3);

    const centerX = col * slotSize + slotSize / 2;
    const centerY = row * slotSize + slotSize / 2;

    // 1. Resolve Dynamic Colors based on Mood
    const moodColors = {
        "Volatile": { r: 255, g: 100, b: 0 },   // Fire
        "Serene": { r: 0, g: 180, b: 255 },     // Water
        "Spirited": { r: 180, g: 0, b: 255 },   // Lightning
        "Grounded": { r: 100, g: 255, b: 50 }   // Earth
    };

    let color = owner === 0 ? { r: 0, g: 242, b: 254 } : { r: 255, g: 75, b: 75 };
    if (card.mood && moodColors[card.mood]) {
        color = moodColors[card.mood];
    }

    // 2. Resolve Stats for Effects (Mojo, Loyalty, Wanted)
    const wantedLevel = (owner === 0 ? state.p1_wanted_level : state.p2_wanted_level) || 0;
    const mojo = (owner === 0 ? state.mojo : 0) || 0; // P1 focus for local simulation feedback
    const loyalty = card.loyalty || 0;

    const particleCount = 40 + Math.floor(mojo / 10); // Higher Mojo = more dense bursts
    const speedBoost = 1 + (loyalty / 200);           // Higher Loyalty = snappier particles

    for (let i = 0; i < particleCount; i++) {
        const angle = Math.random() * Math.PI * 2;
        const speed = (Math.random() * 4 + 1.5) * speedBoost;
        particles.push({
            x: centerX,
            y: centerY,
            vx: Math.cos(angle) * speed,
            vy: Math.sin(angle) * speed,
            size: Math.random() * 2 + 1,
            color: color,
            life: Math.random() * 40 + 20,
            initialLife: 60,
            wantedLevel: wantedLevel,
            type: "gravity"
        });
    }

    if (!particleAnimationId) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    }
}

/**
 * Triggers a golden trophy burst for Arena winners.
 */
export function triggerVictoryEffect() {
    if (!particleCtx) return;

    const state = window.GetGameState();
    const winnerIdx = state.winner;
    if (winnerIdx === -1) return;

    // 1. Resolve Rank/Reputation for local player or opponent
    const rep = (winnerIdx === 0) ? state.reputation : (state.p2_reputation || 0);
    
    // 2. Define Rank-Based Fanfare Tiers
    let particleCount = 60;
    let velocityMult = 1.0;
    let isDiamond = false;

    if (rep >= 1000) { // Diamond
        particleCount = 180;
        velocityMult = 1.5;
        isDiamond = true;
    } else if (rep >= 600) { // Gold
        particleCount = 120;
        velocityMult = 1.2;
    } else if (rep >= 300) { // Silver
        particleCount = 90;
    }

    const width = particleCanvas.width;
    const height = particleCanvas.height;

    for (let i = 0; i < particleCount; i++) {
        particles.push({
            x: Math.random() * width,
            y: height + 10,
            vx: (Math.random() - 0.5) * 6,
            vy: (-Math.random() * 12 - 5) * velocityMult,
            size: Math.random() * 3 + 1,
            color: { r: 255, g: 215, b: 0 }, // Gold
            life: Math.random() * 100 + 50,
            initialLife: 150,
            hasTrail: isDiamond,
            wantedLevel: 0,
            type: "gravity"
        });
    }

    if (!particleAnimationId) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    }
}

/**
 * Triggers digital "Data Breach" lines for the Heist Planning Terminal.
 * Now scales with the player's Cunning attribute.
 */
export function triggerHeistPulse() {
    if (!particleCtx) return;
    const state = window.GetGameState();
    const cunning = state.cunning || 0;

    const width = particleCanvas.width;
    const height = particleCanvas.height;

    // High cunning = faster, stealthier lines. Low cunning = chaotic, glitchy breach.
    const particleCount = 50 + (cunning < 10 ? 30 : 0);
    const speedBase = 2 + (cunning / 5);
    const isGlitchy = cunning < 10;

    for (let i = 0; i < particleCount; i++) {
        particles.push({
            x: Math.random() * width,
            y: Math.random() * height,
            vx: 0,
            vy: Math.random() * 5 + speedBase,
            size: Math.random() * 3 + 2,
            color: isGlitchy ? { r: 255, g: 150, b: 0 } : { r: 255, g: 75, b: 75 }, // Orange/Red shift for instability
            life: 30,
            initialLife: 30,
            type: "data-line",
            isGlitchy: isGlitchy
        });
    }

    if (!particleAnimationId) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    }
}

/**
 * Triggers a shower of VBV motes for Salary payouts or Rewards.
 * Density and prestige colors shift based on Player Mojo.
 */
export function triggerRewardRain() {
    if (!particleCtx) return;
    const state = window.GetGameState();
    const mojo = state.mojo || 0;

    const width = particleCanvas.width;
    const particleCount = 80 + Math.floor(mojo / 5);

    for (let i = 0; i < particleCount; i++) {
        const isGold = mojo > 500 && Math.random() > 0.7; // 30% chance for gold motes for Icons
        particles.push({
            x: Math.random() * width,
            y: -20,
            vx: (Math.random() - 0.5) * 2,
            vy: Math.random() * 4 + 2,
            size: Math.random() * 2 + 1,
            color: isGold ? { r: 255, g: 215, b: 0 } : { r: 0, g: 242, b: 254 },
            life: 200,
            initialLife: 200,
            type: "gravity"
        });
    }

    if (!particleAnimationId) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    }
}

/**
 * Triggers intense radial forge sparks for Club founding and restock.
 * Colors adapt to the Club's specialized Industry Type.
 */
export function triggerFoundryFusion(clubType = "Standard") {
    if (!particleCtx) return;
    const centerX = particleCanvas.width / 2;
    const centerY = particleCanvas.height / 2;

    const typeColors = {
        "Elemental": { r: 0, g: 242, b: 254 }, // Cyan
        "Tactical": { r: 180, g: 0, b: 255 },  // Purple
        "Vitality": { r: 50, g: 255, b: 100 }, // Green
        "Standard": { r: 255, g: 166, b: 87 }  // Orange
    };

    const color = typeColors[clubType] || typeColors["Standard"];

    for (let i = 0; i < 150; i++) {
        const angle = Math.random() * Math.PI * 2;
        const speed = Math.random() * 8 + 2;
        particles.push({
            x: centerX,
            y: centerY,
            vx: Math.cos(angle) * speed,
            vy: Math.sin(angle) * speed,
            size: Math.random() * 3 + 1,
            color: color,
            life: Math.random() * 30 + 20,
            initialLife: 50,
            type: "gravity"
        });
    }

    if (!particleAnimationId) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    }
}

/**
 * Spawns low-density ambient motes for tiles with Elemental Moods.
 * @param {number} gridIndex - The 0-8 slot position.
 * @param {string} mood - The mood type (Volatile, Serene, etc).
 */
export function triggerMoodMote(gridIndex, mood) {
    if (!particleCtx || mood === "Neutral" || !mood) return;

    const board = document.getElementById("board-container");
    if (!board) return;

    const rect = board.getBoundingClientRect();
    const slotSize = rect.width / 3;
    const col = gridIndex % 3;
    const row = Math.floor(gridIndex / 3);

    const centerX = col * slotSize + slotSize / 2;
    const centerY = row * slotSize + slotSize / 2;

    const moodColors = {
        "Volatile": { r: 255, g: 100, b: 0 },
        "Serene": { r: 0, g: 180, b: 255 },
        "Spirited": { r: 180, g: 0, b: 255 },
        "Grounded": { r: 100, g: 255, b: 50 }
    };

    const color = moodColors[mood] || { r: 255, g: 255, b: 255 };

    // Spawn 1-2 motes per trigger for a subtle drifting effect
    const count = Math.floor(Math.random() * 2) + 1;
    for (let i = 0; i < count; i++) {
        particles.push({
            x: centerX + (Math.random() - 0.5) * (slotSize * 0.6),
            y: centerY + (Math.random() - 0.5) * (slotSize * 0.6),
            vx: (Math.random() - 0.5) * 0.5,
            vy: (Math.random() - 0.5) * 0.5 - 0.2, // Drifting upwards
            size: Math.random() * 2 + 0.5,
            color: color,
            life: Math.random() * 100 + 50,
            initialLife: 150,
            type: "ambient",
            wantedLevel: 0
        });
    }

    if (!particleAnimationId) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    }
}

/**
 * Triggers a one-time "Connection Pulse" when an opponent successfully joins a match.
 * Radiates from the center of the board with a tactical energy burst.
 */
export function triggerConnectionPulse() {
    if (!particleCtx) return;
    const centerX = particleCanvas.width / 2;
    const centerY = particleCanvas.height / 2;

    // High-density burst to signal match start
    for (let i = 0; i < 120; i++) {
        const angle = Math.random() * Math.PI * 2;
        const speed = Math.random() * 8 + 3;
        particles.push({
            x: centerX,
            y: centerY,
            vx: Math.cos(angle) * speed,
            vy: Math.sin(angle) * speed,
            size: Math.random() * 4 + 1,
            color: { r: 180, g: 0, b: 255 }, // Tactical Purple
            life: 50,
            initialLife: 50,
            type: "gravity",
            wantedLevel: 0
        });
    }

    if (!particleAnimationId) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    }
}

// Global exposure for WASM engine and legacy HTML event handlers
window.triggerCaptureParticles = triggerCaptureParticles;
window.initParticleSystem = initParticleSystem;
window.triggerVictoryEffect = triggerVictoryEffect;
window.triggerHeistPulse = triggerHeistPulse;
window.triggerRewardRain = triggerRewardRain;
window.triggerFoundryFusion = triggerFoundryFusion;
window.triggerMoodMote = triggerMoodMote;
window.triggerConnectionPulse = triggerConnectionPulse;
