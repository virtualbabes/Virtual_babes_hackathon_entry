// ============================================================================
// card_animations.js — 3D Card Flip & Capture Animations
// ----------------------------------------------------------------------------
// Ported from Triple Triad: animation-engine.js + card-flip patterns
// Handles: card flip (Y-axis rotation), capture burst, combo chain effects,
// elemental glow, mood aura pulses
// ============================================================================

const CARD_ANIMS = {
    perspective: 1000,
    flipDuration: 600,
    captureBurstCount: 12
};

// --- Card Flip Animation ---
export function animateCardFlip(slotEl, newOwner, callback) {
    if (!slotEl) return;
    
    // Add flipping class for CSS animation
    slotEl.classList.add('card-flipping');
    
    // Change mid-animation (at 50% rotation)
    setTimeout(() => {
        slotEl.dataset.owner = newOwner;
        slotEl.classList.remove('card-flipping');
        slotEl.classList.add('card-flipped');
        
        // Cleanup after animation
        setTimeout(() => {
            slotEl.classList.remove('card-flipped');
            if (callback) callback();
        }, CARD_ANIMS.flipDuration / 2);
    }, CARD_ANIMS.flipDuration / 2);
}

// --- Capture Particle Burst ---
export function triggerCaptureBurst(gridIndex, owner) {
    const board = document.getElementById('board-container');
    if (!board) return;
    
    const slot = board.querySelector(`[data-index="${gridIndex}"]`);
    if (!slot) return;
    
    const rect = slot.getBoundingClientRect();
    const color = owner === 0 ? '#3b82f6' : '#ef4444'; // Blue for P1, Red for P2
    
    // Create burst particles
    for (let i = 0; i < CARD_ANIMS.captureBurstCount; i++) {
        createParticle(rect.left + rect.width / 2, rect.top + rect.height / 2, color, i);
    }
}

function createParticle(x, y, color, index) {
    const particle = document.createElement('div');
    particle.className = 'card-particle';
    particle.style.cssText = `
        position: fixed;
        left: ${x}px;
        top: ${y}px;
        width: 8px;
        height: 8px;
        background: ${color};
        border-radius: 50%;
        pointer-events: none;
        z-index: 1000;
        box-shadow: 0 0 6px ${color};
    `;
    
    document.body.appendChild(particle);
    
    // Random direction
    const angle = (index / CARD_ANIMS.captureBurstCount) * Math.PI * 2;
    const velocity = 50 + Math.random() * 50;
    const dx = Math.cos(angle) * velocity;
    const dy = Math.sin(angle) * velocity;
    
    particle.animate([
        { transform: 'translate(0, 0) scale(1)', opacity: 1 },
        { transform: `translate(${dx}px, ${dy}px) scale(0)`, opacity: 0 }
    ], {
        duration: 400 + Math.random() * 200,
        easing: 'ease-out'
    }).onfinish = () => particle.remove();
}

// --- Combo Chain Animation ---
export function animateComboChain(flippedIndices, owner) {
    flippedIndices.forEach((index, i) => {
        setTimeout(() => {
            const slot = document.querySelector(`[data-index="${index}"]`);
            if (slot) {
                slot.classList.add('combo-flash');
                setTimeout(() => slot.classList.remove('combo-flash'), 300);
            }
        }, i * 150); // Stagger each flip
    });
}

// --- Elemental Glow ---
export function applyElementalGlow(slotEl, element) {
    const glowColors = {
        'Fire': '0 0 20px rgba(239, 68, 68, 0.6)',
        'Water': '0 0 20px rgba(59, 130, 246, 0.6)',
        'Earth': '0 0 20px rgba(34, 197, 94, 0.6)',
        'Air': '0 0 20px rgba(255, 255, 255, 0.4)',
        'Neutral': 'none'
    };
    
    if (slotEl && glowColors[element]) {
        slotEl.style.boxShadow = glowColors[element];
    }
}

// --- Mood Aura Pulse ---
export function applyMoodAura(slotEl, mood) {
    const auraColors = {
        'Volatile': 'rgba(239, 68, 68, 0.3)',
        'Serene': 'rgba(59, 130, 246, 0.3)',
        'Spirited': 'rgba(245, 158, 11, 0.3)',
        'Grounded': 'rgba(34, 197, 94, 0.3)',
        'Neutral': 'transparent'
    };
    
    if (slotEl && auraColors[mood]) {
        slotEl.style.background = auraColors[mood];
        slotEl.classList.add('mood-aura');
    }
}

// --- Card Placement Animation ---
export function animateCardPlace(slotEl) {
    if (!slotEl) return;
    
    slotEl.animate([
        { transform: 'scale(0) rotateZ(180deg)', opacity: 0 },
        { transform: 'scale(1.1) rotateZ(0deg)', opacity: 1 },
        { transform: 'scale(1) rotateZ(0deg)', opacity: 1 }
    ], {
        duration: 300,
        easing: 'ease-out'
    });
}

// --- Game Over Fanfare ---
export function animateGameOver(winner) {
    const board = document.getElementById('board-container');
    if (!board) return;
    
    const color = winner === 0 ? '#3b82f6' : '#ef4444';
    
    // Flash the board
    board.animate([
        { boxShadow: `0 0 0px ${color}` },
        { boxShadow: `0 0 60px ${color}` },
        { boxShadow: `0 0 0px ${color}` }
    ], {
        duration: 1000,
        easing: 'ease-in-out'
    });
}

// --- Initialize board animations ---
export function initCardAnimations() {
    // Inject CSS if not already present
    if (!document.getElementById('card-anim-styles')) {
        const style = document.createElement('style');
        style.id = 'card-anim-styles';
        style.textContent = CARD_ANIMATION_CSS;
        document.head.appendChild(style);
    }
}

// --- CSS Injection ---
const CARD_ANIMATION_CSS = `
/* Card Flip Animation */
.card-flipping {
    animation: cardFlip 0.6s ease-in-out;
    transform-style: preserve-3d;
}

@keyframes cardFlip {
    0% { transform: rotateY(0deg) scale(1); }
    50% { transform: rotateY(90deg) scale(1.1); }
    100% { transform: rotateY(180deg) scale(1); }
}

.card-flipped {
    transition: transform 0.3s ease-out;
}

/* Combo Flash */
.combo-flash {
    animation: comboFlash 0.3s ease-out;
}

@keyframes comboFlash {
    0% { background: rgba(255, 215, 0, 0.8); }
    100% { background: transparent; }
}

/* Mood Aura */
.mood-aura {
    animation: moodPulse 2s ease-in-out infinite;
}

@keyframes moodPulse {
    0%, 100% { opacity: 0.3; }
    50% { opacity: 0.6; }
}

/* Card Particle */
.card-particle {
    position: fixed;
    pointer-events: none;
    z-index: 1000;
}

/* Elemental Glow */
.elemental-fire { box-shadow: 0 0 20px rgba(239, 68, 68, 0.6); }
.elemental-water { box-shadow: 0 0 20px rgba(59, 130, 246, 0.6); }
.elemental-earth { box-shadow: 0 0 20px rgba(34, 197, 94, 0.6); }
.elemental-air { box-shadow: 0 0 20px rgba(255, 255, 255, 0.4); }

/* Card Place Animation */
.card-placing {
    animation: cardPlace 0.3s ease-out;
}

@keyframes cardPlace {
    0% { transform: scale(0) rotateZ(180deg); opacity: 0; }
    50% { transform: scale(1.1) rotateZ(0deg); opacity: 1; }
    100% { transform: scale(1) rotateZ(0deg); opacity: 1; }
}
`;

// --- Globals ---
window.initCardAnimations = initCardAnimations;
window.triggerCaptureBurst = triggerCaptureBurst;
window.animateCardFlip = animateCardFlip;
window.animateComboChain = animateComboChain;
window.applyElementalGlow = applyElementalGlow;
window.applyMoodAura = applyMoodAura;
window.animateCardPlace = animateCardPlace;
window.animateGameOver = animateGameOver;
