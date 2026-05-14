import { syncUI } from '../app.js';
import { CONFIG } from './config.js';

export let masterVolume = parseFloat(localStorage.getItem('masterVolume') || '0.5');
export let musicVolume = parseFloat(localStorage.getItem('musicVolume') || '0.5');
export let sfxVolume = parseFloat(localStorage.getItem('sfxVolume') || '0.5');

// --- Low-Latency Audio Subsystem (Web Audio API) ---
let audioCtx = null;
let sfxGainNode = null;
const bufferCache = new Map();

/**
 * Initializes the high-performance SFX engine.
 * Must be triggered by a user gesture (e.g., login or connect button) to satisfy browser policies.
 */
export function initAudioContext() {
    if (audioCtx) return;
    try {
        const AudioContextClass = window.AudioContext || window.webkitAudioContext;
        if (!AudioContextClass) return;

        audioCtx = new AudioContextClass();
        sfxGainNode = audioCtx.createGain();
        sfxGainNode.connect(audioCtx.destination);
        
        syncSFXGain();
        console.log("[AUDIO] High-performance SFX engine initialized.");
    } catch (e) {
        console.warn("[AUDIO] AudioContext initialization failed. Falling back to legacy audio.");
    }
}

/**
 * Updates the SFX GainNode to match master and sfx volume settings.
 */
export function syncSFXGain() {
    if (!sfxGainNode || !audioCtx) return;
    const gain = masterVolume * sfxVolume;
    // use setTargetAtTime to avoid audio pops during volume changes
    sfxGainNode.gain.setTargetAtTime(gain, audioCtx.currentTime, 0.05);
}

/**
 * Centralized setter for SFX volume to ensure persistence and gain synchronization.
 */
export function updateSfxVolume(value) {
    sfxVolume = parseFloat(value);
    localStorage.setItem('sfxVolume', sfxVolume);
    syncSFXGain();
}

/**
 * Centralized setter for Master volume.
 */
export function updateMasterVolume(value) {
    masterVolume = parseFloat(value);
    localStorage.setItem('masterVolume', masterVolume);
    syncSFXGain();
}

/**
 * Centralized setter for Music volume.
 */
export function updateMusicVolume(value) {
    musicVolume = parseFloat(value);
    localStorage.setItem('musicVolume', musicVolume);
}

/**
 * Fetches and decodes an audio file into an AudioBuffer for zero-latency playback.
 */
async function getSFXBuffer(path) {
    const url = path.startsWith('http') ? path : `${CONFIG.ASSET_URL}Assets/Audio/${path}`;
    if (bufferCache.has(url)) return bufferCache.get(url);

    try {
        const response = await fetch(url);
        const arrayBuffer = await response.arrayBuffer();
        const buffer = await audioCtx.decodeAudioData(arrayBuffer);
        bufferCache.set(url, buffer);
        return buffer;
    } catch (err) {
        console.warn(`[AUDIO] Buffer load failed: ${url}`);
        return null;
    }
}

/**
 * Plays a subtle audio cue for ambient mood motes.
 * Uses 'Toggle_bip.mp3' from Game_Feedback as a soft 'spark' sound to accompany visual particles.
 */
export function playMoodMoteSFX(mood) {
    if (sfxVolume <= 0 || masterVolume <= 0) return;
    
    // Throttling: Mood motes are very frequent. Audio triggers on only ~5% of visual events
    // to maintain a subtle, immersive "hum" rather than a cacophony.
    if (Math.random() > 0.05) return;

    playSFX('Game_Feedback/Toggle_bip.mp3');
}

/**
 * Plays a high-intensity audio cue for match connections.
 * Accompanies the visual 'triggerConnectionPulse' effect.
 */
export function playConnectionSFX() {
    playSFX('Connected.mp3');
}

/**
 * Plays a high-intensity battle start audio cue.
 * Accompanies the transition to the active combat phase.
 */
export function playBattleStartSFX() {
    playSFX('Start_intense.mp3');
}

/**
 * Unified low-latency play function.
 * Overrides the legacy window.PlaySound to provide polyphony and better performance.
 */
export async function playSFX(path) {
    if (sfxVolume <= 0 || masterVolume <= 0) return;

    // Lazy-init if not already called by a UI gesture, or resume if suspended
    if (!audioCtx) initAudioContext();
    if (audioCtx && audioCtx.state === 'suspended') audioCtx.resume();

    if (!audioCtx) {
        // Fallback to legacy Audio path if engine not ready
        const url = path.startsWith('http') ? path : `${CONFIG.ASSET_URL}Assets/Audio/${path}`;
        const audio = new Audio(url);
        audio.volume = masterVolume * sfxVolume;
        audio.play().catch(() => {});
        return;
    }

    const buffer = await getSFXBuffer(path);
    if (!buffer) return;

    const source = audioCtx.createBufferSource();
    source.buffer = buffer;
    source.connect(sfxGainNode);
    source.start(0);
}

window.PlaySound = playSFX;
window.initAudioContext = initAudioContext;
window.playMoodMoteSFX = playMoodMoteSFX;
window.playConnectionSFX = playConnectionSFX;
window.playBattleStartSFX = playBattleStartSFX;
