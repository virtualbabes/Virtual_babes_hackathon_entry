// Public/js/particles.js
// PILLAR 5: Visual Feedback & Immersion.

let particles = [];
const MAX_PARTICLES = 150;
let isAnimating = false; // PILLAR 5: CPU Optimization for Mobile

/**
 * startAnimationLoop ensures only one requestAnimationFrame cycle is active.
 */
function startAnimationLoop() {
    if (isAnimating) return;
    isAnimating = true;
    requestAnimationFrame(animateParticles);
}

/**
 * triggerMoodMote generates small, short-lived particles at a specific grid index, colored by mood.
 * PILLAR 5: Visual Feedback & Immersion.
 */
export function triggerMoodMote(gridIndex, mood) {
    const canvas = document.getElementById("particle-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext('2d');

    // Map gridIndex to approximate canvas coordinates
    const cellSize = canvas.width / 3;
    const row = Math.floor(gridIndex / 3);
    const col = gridIndex % 3;
    const x = col * cellSize + cellSize / 2;
    const y = row * cellSize + cellSize / 2;

    const moodColors = {
        "Volatile": "#ff4b4b", // Red
        "Serene": "#00f2fe",   // Cyan
        "Spirited": "#ffd700", // Gold
        "Grounded": "#3fb950", // Green
        "Neutral": "#888888"   // Grey
    };

    for (let i = 0; i < 5; i++) { // Generate a few motes per trigger
        particles.push({
            x: x + (Math.random() - 0.5) * 20, y: y + (Math.random() - 0.5) * 20,
            vx: (Math.random() - 0.5) * 2, vy: (Math.random() - 0.5) * 2,
            size: Math.random() * 2 + 1, color: moodColors[mood] || "#ffffff",
            life: 0.8, decay: Math.random() * 0.05 + 0.02,
        });
    }
    startAnimationLoop();
}

/**
 * triggerCloakFailureParticles generates a purple glitch effect.
 * Triggers when Ghost Protocol expires for an outlaw.
 */
export function triggerCloakFailureParticles() {
    const canvas = document.getElementById("particle-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const rect = canvas.getBoundingClientRect();

    // Generate 40-60 erratic purple glitch particles
    for (let i = 0; i < 50; i++) {
        particles.push({
            x: rect.width / 2,
            y: 0, // Burst from the top bar area
            vx: (Math.random() - 0.5) * 20,
            vy: Math.random() * 15,
            size: Math.random() * 4 + 2,
            color: Math.random() > 0.5 ? "#9b51e0" : "#ff00ff", // Purple / Magenta
            life: 1.0,
            decay: Math.random() * 0.05 + 0.02,
            glitch: true
        });
    }
    
    if (particles.length > MAX_PARTICLES) {
        particles = particles.slice(-MAX_PARTICLES);
    }
    
    startAnimationLoop();
}

/**
 * triggerMutationInsuranceEffect renders or removes the golden shield bubble.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
export function triggerMutationInsuranceEffect(active) {
    const canvas = document.getElementById("particle-canvas");
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();

    // Remove existing shield instances
    particles = particles.filter(p => !p.isShield);

    if (active) {
        particles.push({
            x: rect.width / 2,
            y: rect.height / 2,
            size: 130, // Radius for the hexagon
            color: "rgba(255, 215, 0, 0.1)", // Faint gold fill
            borderColor: "#ffd700",         // Neon gold border
            life: 1.0,
            decay: 0, // Persistent until manually removed
            isShield: true,
            pulse: 0
        });
        startAnimationLoop();
    }
}

/**
 * triggerStaffTrainingEffect initiates a horizontal laser scan across the arena.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
export function triggerStaffTrainingEffect() {
    const canvas = document.getElementById("particle-canvas");
    if (!canvas) return;

    // Reset existing scanners to prevent overlap
    particles = particles.filter(p => !p.isScanner);

    particles.push({
        y: 0,
        vy: 12, // High velocity scan
        color: "#00f2fe", // Neon Cyan
        life: 1.0,
        decay: 0,
        isScanner: true
    });
    
    startAnimationLoop();
}

/**
 * triggerMutationSuccessParticles generates an emerald-green geometric burst.
 * Triggers on successful gene-editing procedures.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
export function triggerMutationSuccessParticles() {
    const canvas = document.getElementById("particle-canvas");
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();

    // Generate 80 high-velocity radial geometric particles
    for (let i = 0; i < 80; i++) {
        const angle = Math.random() * Math.PI * 2;
        const speed = Math.random() * 12 + 4;
        particles.push({
            x: rect.width / 2,
            y: rect.height / 2,
            vx: Math.cos(angle) * speed,
            vy: Math.sin(angle) * speed,
            size: Math.random() * 4 + 2,
            color: "#50c878", // Emerald Green
            life: 1.0,
            decay: Math.random() * 0.03 + 0.015,
            isGeometric: true,
            sides: Math.random() > 0.5 ? 3 : 4, // Triangles and Diamonds
            rotation: Math.random() * Math.PI * 2,
            spin: (Math.random() - 0.5) * 0.2
        });
    }

    if (particles.length > MAX_PARTICLES) particles = particles.slice(-MAX_PARTICLES);
    startAnimationLoop();
}

/**
 * triggerDistrictStabilizerEffect renders or removes the shimmering cyan grid.
 * PILLAR 1: Infrastructure Prestige.
 */
export function triggerDistrictStabilizerEffect(active) {
    const canvas = document.getElementById("particle-canvas");
    if (!canvas) return;

    // Reset existing instances to prevent duplicates
    particles = particles.filter(p => !p.isDistrictGrid);

    if (active) {
        particles.push({
            color: "#00f2fe", // Neon Cyan
            life: 1.0,
            decay: 0,         // Persistent until manually removed
            isDistrictGrid: true,
            pulse: 0
        });
        startAnimationLoop();
    }
}

/**
 * triggerCloakDisruptorParticles generates a cyan lightning burst effect.
 * Triggers when a Hunter manually disrupts an outlaw's signal via tactical item.
 * PILLAR 5: Visual Feedback.
 */
export function triggerCloakDisruptorParticles() {
    const canvas = document.getElementById("particle-canvas");
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();

    // Generate high-velocity cyan "lightning" streaks
    for (let i = 0; i < 80; i++) {
        particles.push({
            x: rect.width / 2,
            y: rect.height / 2, 
            vx: (Math.random() - 0.5) * 70, // Sharp intensity
            vy: (Math.random() - 0.5) * 70,
            size: Math.random() * 2 + 1,
            color: "#00f2fe", // Neon Cyan
            life: 0.6,
            decay: Math.random() * 0.08 + 0.05,
            glitch: true,
            isLightning: true
        });
    }
    
    if (particles.length > MAX_PARTICLES) {
        particles = particles.slice(-MAX_PARTICLES);
    }
    
    startAnimationLoop();
}

export function initParticleSystem() {
    const canvas = document.getElementById("particle-canvas");
    if (!canvas) return;
    const resize = () => {
        canvas.width = canvas.parentElement.clientWidth;
        canvas.height = canvas.parentElement.clientHeight;
    };
    window.addEventListener('resize', resize);
    resize();
}

export function animateParticles() {
    const canvas = document.getElementById("particle-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext('2d');

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    for (let i = particles.length - 1; i >= 0; i--) {
        const p = particles[i];
        
        // PILLAR 5: Visual Hardening. Capture previous state for lightning streaks.
        const lx = p.x;
        const ly = p.y;

        p.x += p.vx;
        p.y += p.vy;
        p.life -= p.decay;

        if (p.spin) p.rotation += p.spin;

        if (p.life <= 0) {
            particles.splice(i, 1);
            continue;
        }

        if (p.glitch && Math.random() > 0.8) {
            p.x += (Math.random() - 0.5) * 10; // Random jitter
        }

        ctx.globalAlpha = p.life;
        if (p.isLightning) {
            ctx.strokeStyle = p.color;
            ctx.lineWidth = p.size;
            ctx.beginPath();
            ctx.moveTo(lx, ly);
            ctx.lineTo(p.x, p.y);
            ctx.stroke();
        } else if (p.isGeometric) {
            // PILLAR 6: Geometric Expansion Renderer.
            ctx.fillStyle = p.color;
            ctx.beginPath();
            const sides = p.sides || 3;
            const rotation = p.rotation || 0;
            for (let s = 0; s < sides; s++) {
                const angle = rotation + (s * 2 * Math.PI / sides);
                const px = p.x + p.size * Math.cos(angle);
                const py = p.y + p.size * Math.sin(angle);
                if (s === 0) ctx.moveTo(px, py);
                else ctx.lineTo(px, py);
            }
            ctx.closePath();
            ctx.fill();
        } else if (p.isShield) {
            // PILLAR 6: Mutation Insurance Shield Renderer.
            p.pulse += 0.02;
            const pulseFactor = 1 + Math.sin(p.pulse) * 0.05;
            const radius = p.size * pulseFactor;

            ctx.save();
            ctx.translate(p.x, p.y);
            ctx.rotate(p.pulse * 0.05); // Majestic slow rotation

            ctx.strokeStyle = p.borderColor;
            ctx.lineWidth = 3;
            ctx.fillStyle = p.color;
            ctx.shadowBlur = 15;
            ctx.shadowColor = p.borderColor;

            ctx.beginPath();
            for (let s = 0; s < 6; s++) {
                const angle = s * 2 * Math.PI / 6;
                const sx = radius * Math.cos(angle);
                const sy = radius * Math.sin(angle);
                if (s === 0) ctx.moveTo(sx, sy);
                else ctx.lineTo(sx, sy);
            }
            ctx.closePath();
            ctx.fill();
            ctx.stroke();
            ctx.restore();
        } else if (p.isScanner) {
            // PILLAR 6: Staff Training Scan Line Renderer.
            p.y += p.vy;
            
            if (p.y > canvas.height) {
                particles.splice(i, 1);
                continue;
            }

            ctx.save();
            ctx.strokeStyle = p.color;
            ctx.lineWidth = 3;
            ctx.shadowBlur = 20;
            ctx.shadowColor = p.color;
            ctx.globalAlpha = p.life;

            // Main scanning laser
            ctx.beginPath();
            ctx.moveTo(0, p.y);
            ctx.lineTo(canvas.width, p.y);
            ctx.stroke();

            // Leading glow edge
            ctx.lineWidth = 1;
            ctx.globalAlpha = 0.5;
            ctx.strokeRect(0, p.y - 2, canvas.width, 4);

            // Digital distortion/noise (flickering bits)
            if (Math.random() > 0.4) {
                ctx.fillStyle = p.color;
                ctx.globalAlpha = 0.4;
                for(let j=0; j<5; j++) {
                    const bx = Math.random() * canvas.width;
                    const bw = Math.random() * 40 + 10;
                    ctx.fillRect(bx, p.y - 1, bw, 2);
                }
            }
            ctx.restore();
        } else if (p.isDistrictGrid) {
            // PILLAR 1: District Stabilizer Grid Renderer.
            p.pulse += 0.015;
            const shimmer = 0.05 + Math.sin(p.pulse) * 0.03;
            
            ctx.save();
            ctx.strokeStyle = p.color;
            ctx.lineWidth = 1;
            ctx.globalAlpha = shimmer;
            
            const cellSize = 50;
            ctx.beginPath();
            // Vertical lines
            for (let x = 0; x <= canvas.width; x += cellSize) {
                ctx.moveTo(x, 0);
                ctx.lineTo(x, canvas.height);
            }
            // Horizontal lines
            for (let y = 0; y <= canvas.height; y += cellSize) {
                ctx.moveTo(0, y);
                ctx.lineTo(canvas.width, y);
            }
            ctx.stroke();
            ctx.restore();
        } else {
            ctx.fillStyle = p.color;
            ctx.fillRect(p.x, p.y, p.size, p.size);
        }
    }

    if (particles.length > 0) {
        requestAnimationFrame(animateParticles);
    } else {
        isAnimating = false;
        ctx.clearRect(0, 0, canvas.width, canvas.height); // Final clear
    }
}

// --- Placeholders for existing system requirements ---
export function triggerFoundryFusion(type) {
    // Implementation for club-specific fusion effects
    startAnimationLoop();
}

export function triggerCaptureParticles(idx, owner) {
    startAnimationLoop();
}

export function triggerGlobalKidnapEffect() {
    startAnimationLoop();
}

export function triggerMutationScarEffect() {
    const canvas = document.getElementById("particle-canvas");
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();

    // PILLAR 6: Specialized Gene-Editing Feedback.
    // Generate high-intensity blood-red jagged streaks radiating from center.
    for (let i = 0; i < 100; i++) {
        particles.push({
            x: rect.width / 2,
            y: rect.height / 2,
            vx: (Math.random() - 0.5) * 100, // Violent, wide-reaching velocity
            vy: (Math.random() - 0.5) * 100,
            size: Math.random() * 3 + 2,
            color: "#ff0000", // Blood Red
            life: 0.5,
            decay: Math.random() * 0.1 + 0.05,
            glitch: true,
            isLightning: true
        });
    }

    // Screen-shake emulation: add horizontal glitch lines that span the viewport
    for (let i = 0; i < 10; i++) {
        particles.push({
            x: 0,
            y: Math.random() * rect.height,
            vx: rect.width,
            vy: 0,
            size: 1,
            color: "#880000",
            life: 0.3,
            decay: 0.1,
            isLightning: true
        });
    }

    if (particles.length > MAX_PARTICLES) {
        particles = particles.slice(-MAX_PARTICLES);
    }

    startAnimationLoop();
}

window.triggerCloakFailureParticles = triggerCloakFailureParticles;
window.triggerCloakDisruptorParticles = triggerCloakDisruptorParticles;
window.triggerDistrictStabilizerEffect = triggerDistrictStabilizerEffect;
window.triggerStaffTrainingEffect = triggerStaffTrainingEffect;
window.triggerMutationInsuranceEffect = triggerMutationInsuranceEffect;
window.triggerMutationSuccessParticles = triggerMutationSuccessParticles;
window.triggerFoundryFusion = triggerFoundryFusion;
window.triggerGlobalKidnapEffect = triggerGlobalKidnapEffect;
window.triggerMutationScarEffect = triggerMutationScarEffect;
window.triggerCaptureParticles = triggerCaptureParticles;
window.initParticleSystem = initParticleSystem;