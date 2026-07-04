import { CONFIG } from './config.js';

export let masterVolume = parseFloat(localStorage.getItem('masterVolume') || '0.5');
export let musicVolume = parseFloat(localStorage.getItem('musicVolume') || '0.5');
export let sfxVolume = parseFloat(localStorage.getItem('sfxVolume') || '0.5');
export let lastMasterVolume = parseFloat(localStorage.getItem('lastMasterVolume') || (masterVolume > 0 ? masterVolume : '0.5')); // PILLAR 4: Load last non-zero volume
export let lastSfxVolume = parseFloat(localStorage.getItem('lastSfxVolume') || (sfxVolume > 0 ? sfxVolume : '0.5')); // PILLAR 4: Load last non-zero volume

// --- Low-Latency Audio Subsystem (Web Audio API) ---
let audioCtx = null;
let sfxGainNode = null;
let musicGainNode = null;
let currentMutationSoundscapeSource = null;

// PILLAR 6: Phase-Based Track Mapping
export const MUSIC_PHASE_MAP = {
    "DISCONNECTED": "Not_connected_ambient",
    "Setup": "Unbuilt_deck_ambient",
    "Lobby": "ambient_menu_music_2",
    "TournamentLobby": "Tournament_game_ambient",
    "Active_Casual": "2_player_ambient_1",
    "Active_Quick": "quick_play_ambient_1",
    "Active_Tournament": "Tournament_game_ambient_2",
    "Finished": "ambient_menu_music_4"
};

let currentMusicGain = null;
let isMusicTransitioning = false;
let currentMutationInsuranceHumSource = null;
let currentDistrictStabilizerThrumSource = null;
const bufferCache = new Map(); // url -> Promise<AudioBuffer>
const sfxCooldowns = new Map(); // path -> timestamp
const MIN_SFX_INTERVAL = 50; // ms

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
        musicGainNode = audioCtx.createGain();

        // PILLAR 5: Audio Hardening.
        // Implement a master compressor to prevent digital clipping during high-frequency combat events.
        const compressor = audioCtx.createDynamicsCompressor();
        compressor.threshold.setValueAtTime(-24, audioCtx.currentTime);
        compressor.knee.setValueAtTime(40, audioCtx.currentTime);
        compressor.ratio.setValueAtTime(12, audioCtx.currentTime);
        compressor.attack.setValueAtTime(0, audioCtx.currentTime);
        compressor.release.setValueAtTime(0.25, audioCtx.currentTime);

        sfxGainNode.connect(compressor);
        musicGainNode.connect(compressor);
        compressor.connect(audioCtx.destination);
        
        syncSFXGain();
        syncMusicGain();

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
 * Updates the Music GainNode to match master and music volume settings.
 */
export function syncMusicGain() {
    if (!musicGainNode || !audioCtx) return;
    const gain = masterVolume * musicVolume;
    musicGainNode.gain.setTargetAtTime(gain, audioCtx.currentTime, 0.05);
}

/**
 * Centralized setter for SFX volume to ensure persistence and gain synchronization.
 */
export function updateSfxVolume(value) {
    sfxVolume = parseFloat(value);
    localStorage.setItem('sfxVolume', sfxVolume);
    if (sfxVolume > 0) {
        lastSfxVolume = sfxVolume;
        localStorage.setItem('lastSfxVolume', lastSfxVolume);
    }
    syncSFXGain();
}

/**
 * Centralized setter for Master volume.
 */
export function updateMasterVolume(value) {
    masterVolume = parseFloat(value);
    localStorage.setItem('masterVolume', masterVolume);
    if (masterVolume > 0) {
        lastMasterVolume = masterVolume;
        localStorage.setItem('lastMasterVolume', lastMasterVolume);
    }
    syncSFXGain();
    syncMusicGain(); // PILLAR 4: Authoritative Master. Sync both gain nodes.
}

/**
 * Centralized setter for Music volume.
 */
export function updateMusicVolume(value) {
    musicVolume = parseFloat(value);
    localStorage.setItem('musicVolume', musicVolume);
    if (musicVolume > 0) { // PILLAR 4: Persist last non-zero volume
        lastMusicVolume = musicVolume;
        localStorage.setItem('lastMusicVolume', lastMusicVolume);
    }
    syncMusicGain();
}

/**
 * Toggles the master volume between 0 and its last non-zero value.
 */
export function toggleMuteMaster() {
    if (masterVolume > 0) {
        lastMasterVolume = masterVolume;
        updateMasterVolume(0);
    } else {
        updateMasterVolume(lastMasterVolume > 0 ? lastMasterVolume : 0.5); // Restore or default
    }
}

/**
 * Toggles the SFX volume between 0 and its last non-zero value.
 */
export function toggleMuteSfx() {
    if (sfxVolume > 0) { lastSfxVolume = sfxVolume; updateSfxVolume(0); }
    else { updateSfxVolume(lastSfxVolume > 0 ? lastSfxVolume : 0.5); } // Restore or default
}

/**
 * Toggles the music volume between 0 and its last non-zero value.
 * PILLAR 4: Persistence Hardening.
 */
export function toggleMuteMusic() {
    if (musicVolume > 0) { lastMusicVolume = musicVolume; updateMusicVolume(0); }
    else { updateMusicVolume(lastMusicVolume > 0 ? lastMusicVolume : 0.5); } // Restore or default
}

/**
 * Fetches and decodes an audio file into an AudioBuffer for zero-latency playback.
 */
async function getSFXBuffer(path) {
    const url = path.startsWith('http') ? path : `${CONFIG.ASSET_URL}Assets/Audio/${path}`;
    if (bufferCache.has(url)) return bufferCache.get(url);

    const promise = (async () => {
        try {
            const response = await fetch(url);
            const arrayBuffer = await response.arrayBuffer();
            // PILLAR 5: Deterministic Audio Decoding. 
            return await audioCtx.decodeAudioData(arrayBuffer);
        } catch (err) {
            console.warn(`[AUDIO] Buffer load failed: ${url}`);
            bufferCache.delete(url); // Clear failed load from cache
            return null;
        }
    })();

    bufferCache.set(url, promise);
    return promise;
}

/**
 * Plays a subtle audio cue for ambient mood motes.
 * Uses 'Toggle_bip.mp3' from Game_Feedback as a soft 'spark' sound to accompany visual particles.
 */
export function playMoodMoteSFX(mood) {
    if (sfxVolume <= 0 || masterVolume <= 0) return;
    
    // Throttling: Mood motes are very frequent. Audio triggers on only ~5% of visual events
    // to maintain a subtle, immersive "hum" rather than a cacophony.
    if (Math.random() > 0.05) return; // PILLAR 5: Throttled for performance

    playSFX('Toggle_bip.mp3');
}

/**
 * Plays a high-intensity audio cue for match connections.
 * Accompanies the visual 'triggerConnectionPulse' effect.
 */
export function playConnectionSFX() {
    playSFX('Connected.mp3');
}

/**
 * Plays the glitchy 'Cyber-Pulse' sound for regional blackouts.
 * PILLAR 1: Regional Warfare Feedback.
 */
export function playBlackoutSFX() {
    playSFX('Cyber-Pulse.mp3');
}

/**
 * Plays the intense 'Chain_reaction' sound for combo flips.
 * PILLAR 5: Atmospheric Immersion.
 */
export function playComboSFX() {
    playSFX('Chain_reaction.mp3');
}

/**
 * Plays the 'Cyber-Pulse' sound for Bounty Board de-cloaking.
 * PILLAR 3: Criminality & Intelligence.
 */
export function playCloakFailureSFX() {
    playSFX('Cyber-Pulse.mp3');
}

/**
 * Plays a constant low-frequency electronic thrum for the District Stabilizer.
 * PILLAR 1: Infrastructure Prestige.
 */
export async function playDistrictStabilizerThrum() {
    if (sfxVolume <= 0 || masterVolume <= 0) return;
    if (currentDistrictStabilizerThrumSource) return;

    if (!audioCtx) initAudioContext();
    if (audioCtx && audioCtx.state === 'suspended') audioCtx.resume();

    const buffer = await getSFXBuffer('Industrial_Hum.mp3');
    if (!buffer || !audioCtx) return;

    currentDistrictStabilizerThrumSource = audioCtx.createBufferSource();
    currentDistrictStabilizerThrumSource.buffer = buffer;
    currentDistrictStabilizerThrumSource.loop = true;
    
    // PILLAR 6: Differentiate via pitch shifting. 
    // 0.45 rate creates a constant low-frequency thrum.
    currentDistrictStabilizerThrumSource.playbackRate.value = 0.45;
    
    const thrumGain = audioCtx.createGain();
    thrumGain.gain.value = 0.4; // Subtle ambient thrum
    currentDistrictStabilizerThrumSource.connect(thrumGain);
    thrumGain.connect(sfxGainNode);
    currentDistrictStabilizerThrumSource.start(0);
}

/**
 * Stops the District Stabilizer thrum.
 */
export function stopDistrictStabilizerThrum() {
    if (currentDistrictStabilizerThrumSource) {
        try {
            currentDistrictStabilizerThrumSource.stop();
        } catch (e) {}
        currentDistrictStabilizerThrumSource = null;
    }
}

/**
 * Plays a rhythmic data-processing sound effect for Staff Training activation.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
export function playStaffTrainingSFX() {
    playSFX('Cyber-Pulse.mp3');
}

/**
 * Plays a specialized 'ka-ching' variant for Sabotage Reparations.
 * PILLAR 1: Trust Layer Feedback.
 */
export async function playSabotageReparationSFX() {
    if (sfxVolume <= 0 || masterVolume <= 0) return;

    if (!audioCtx) initAudioContext();
    if (audioCtx && audioCtx.state === 'suspended') audioCtx.resume();

    const buffer = await getSFXBuffer('Pay_out-in-2.mp3');
    if (!buffer || !audioCtx) return;

    const source = audioCtx.createBufferSource();
    source.buffer = buffer;

    // PILLAR 6: Audio Variation. 
    // 1.2 rate creates a higher, more satisfying "metallic" ring.
    source.playbackRate.value = 1.2;

    const sfxGain = audioCtx.createGain();
    sfxGain.gain.value = 1.0; 
    source.connect(sfxGain);
    sfxGain.connect(sfxGainNode);
    
    source.start(0);
}

/**
 * Plays a high-priority system alarm for ecosystem milestones.
 * PILLAR 1: Industrial Loop Feedback.
 */
export function playEcosystemAlertSFX() {
    playSFX('Warning_long.mp3');
}

/**
 * Plays the quick 'interupt_warning' sound for botched procedures.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
export function playProcedureInterruptedSFX() {
    playSFX('interupt_warning.mp3');
}

/**
 * Plays the 'Warning_long' sound for critical systemic alerts.
 * PILLAR 1: Infrastructure Security.
 */
export function playLongWarningSFX() {
    playSFX('Warning_long.mp3');
}

/**
 * Plays a high-intensity battle start audio cue.
 * Accompanies the transition to the active combat phase.
 */
export function playBattleStartSFX() {
    playSFX('Start_intense.mp3');
}

/**
 * Plays a high-pitched static discharge for cloak disruption events.
 * PILLAR 3: Criminality & Intelligence Feedback.
 */
export function playCloakDisruptorSFX() {
    playSFX('High-pitched_static_discharge.mp3');
}

/**
 * Plays a successful mutation chain reaction for procedure completion.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
export function playMutationSuccessSFX() {
    playSFX('Chain_reaction.mp3');
}

/**
 * Plays the low-frequency industrial background for the Mutation Foundry.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
export async function playMutationSoundscape() {
    if (sfxVolume <= 0 || masterVolume <= 0) return;
    if (currentMutationSoundscapeSource) return;

    if (!audioCtx) initAudioContext();
    if (audioCtx && audioCtx.state === 'suspended') audioCtx.resume();

    const buffer = await getSFXBuffer('Industrial_Hum.mp3');
    if (!buffer || !audioCtx) return;

    currentMutationSoundscapeSource = audioCtx.createBufferSource();
    currentMutationSoundscapeSource.buffer = buffer;
    currentMutationSoundscapeSource.loop = true;
    currentMutationSoundscapeSource.connect(sfxGainNode);
    currentMutationSoundscapeSource.start(0);
}

/**
 * Plays a deep, low-frequency power hum for insured procedures.
 * Layers over the soundscape to provide tactical confirmation.
 */
export async function playMutationInsuranceHum() {
    if (sfxVolume <= 0 || masterVolume <= 0) return;
    if (currentMutationInsuranceHumSource) return;

    if (!audioCtx) initAudioContext();
    if (audioCtx && audioCtx.state === 'suspended') audioCtx.resume();

    const buffer = await getSFXBuffer('Industrial_Hum.mp3');
    if (!buffer || !audioCtx) return;

    currentMutationInsuranceHumSource = audioCtx.createBufferSource();
    currentMutationInsuranceHumSource.buffer = buffer;
    currentMutationInsuranceHumSource.loop = true;
    
    // PILLAR 6: Differentiate via pitch shifting. 
    // 0.55 rate creates a deep "Power Grid" resonance.
    currentMutationInsuranceHumSource.playbackRate.value = 0.55;
    
    const humGain = audioCtx.createGain();
    humGain.gain.value = 0.7; // Subtle layering
    currentMutationInsuranceHumSource.connect(humGain);
    humGain.connect(sfxGainNode);
    currentMutationInsuranceHumSource.start(0);
}

/**
 * Plays character-specific victory/defeat voice lines or generic sounds.
 * @param {string} characterType - The NPC archetype (e.g., "Witch", "Boss", "Lady", "cute", "Mini-Boss").
 * @param {boolean} isPlayerVictory - True if the player won, false if the opponent (NPC or human) won.
 * @param {boolean} isMultiplayer - True if it's a multiplayer match, false for single-player vs NPC.
 */
export function playCharacterVoiceLine(characterType, isPlayerVictory, isMultiplayer) {
    if (sfxVolume <= 0 || masterVolume <= 0) return;

    let audioFile = '';

    if (isPlayerVictory) {
        // Player wins (either vs NPC or Human)
        audioFile = 'Crowd/applause_player_win.mp3';
    } else {
        // Opponent wins (either NPC or Human)
        if (isMultiplayer) {
            // Generic opponent win sound for human vs human
            audioFile = 'opponent_win.wav';
        } else {
            // NPC wins vs Player
            switch (characterType) {
                case "Witch": audioFile = 'Witch/evil-witch-laugh-140135.mp3'; break;
                case "Boss": audioFile = 'Boss/evil-laugh-47891.mp3'; break;
                case "Lady": audioFile = 'Lady/soft-laughing-6445.mp3'; break;
                case "cute": audioFile = 'cute/hehehehe-288404.mp3'; break;
                case "Mini-Boss": audioFile = 'Mini-Boss/sinister-laugh-146634.mp3'; break;
                default: audioFile = 'opponent_win.wav'; // Fallback generic NPC win
            }
        }
    }

    if (audioFile) {
        playSFX(audioFile);
    }
}

/**
 * Unified low-latency play function.
 * Overrides the legacy window.PlaySound to provide polyphony and better performance.
 */
export async function playSFX(path) {
    if (sfxVolume <= 0 || masterVolume <= 0) return;

    // PILLAR 5: Audio Hardening.
    // Prevent buffer exhaustion and UX cacophony by enforcing a minimum interval per sound type.
    // This is critical for high-frequency item usage and chain reactions.
    const now = Date.now();
    if (sfxCooldowns.has(path) && (now - sfxCooldowns.get(path)) < MIN_SFX_INTERVAL) {
        return;
    }
    sfxCooldowns.set(path, now);

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

// --- Challenge Audio Handlers ---
let activeChallengeWaitSource = null;

/**
 * Plays the challenge wait loop for the sender while awaiting a response.
 */
export async function playChallengeWaitSFX() {
    if (sfxVolume <= 0 || masterVolume <= 0) return;
    if (activeChallengeWaitSource) return;

    if (!audioCtx) initAudioContext();
    if (audioCtx && audioCtx.state === 'suspended') audioCtx.resume();

    const buffer = await getSFXBuffer('Challenge_wait.mp3');
    if (!buffer || !audioCtx) return;

    activeChallengeWaitSource = audioCtx.createBufferSource();
    activeChallengeWaitSource.buffer = buffer;
    activeChallengeWaitSource.loop = true;
    activeChallengeWaitSource.connect(sfxGainNode);
    activeChallengeWaitSource.start(0);
}

/**
 * Stops the active challenge wait loop.
 */
export function stopChallengeWaitSFX() {
    if (activeChallengeWaitSource) {
        try {
            activeChallengeWaitSource.stop();
        } catch (e) {
            // Source might have already stopped or not started
        }
        activeChallengeWaitSource = null;
    }
}

/**
 * Stops the Mutation Insurance power hum.
 */
export function stopMutationInsuranceHum() {
    if (currentMutationInsuranceHumSource) {
        try {
            currentMutationInsuranceHumSource.stop();
        } catch (e) {}
        currentMutationInsuranceHumSource = null;
    }
}

/**
 * Plays the challenge accepted sound.
 */
export function playChallengeAcceptedSFX() {
    playSFX('Challenge_accepted.mp3');
}

/**
 * Plays a randomized challenge declined sound variant.
 */
export function playChallengeDeclinedSFX() {
    const variants = [
        'Chalenge_declined.mp3',
        'Chalenge_declined1.mp3',
        'Chalenge_declined2.mp3',
        'Chalenge_declined3.mp3'
    ];
    const randomVariant = variants[Math.floor(Math.random() * variants.length)];
    playSFX(randomVariant);
}

// --- Music Management ---
let currentMusicSource = null;

/**
 * transitionMusic handles seamless cross-fading between ambient tracks.
 * PILLAR 6: Phase-Based Atmosphere.
 */
export async function transitionMusic(trackName) {
    if (musicVolume <= 0 || masterVolume <= 0 || isMusicTransitioning) return;
    
    const currentTrack = localStorage.getItem('currentMusicTrack');
    if (currentTrack === trackName) return;

    isMusicTransitioning = true;

    if (!audioCtx) initAudioContext();
    if (audioCtx && audioCtx.state === 'suspended') {
        try { await audioCtx.resume(); } catch(e) { return; }
    }

    const path = trackName.includes('.') ? trackName : `${trackName}.mp3`;
    const buffer = await getSFXBuffer(path);
    if (!buffer || !audioCtx) return;

    const fadeTime = 1.5; // 1.5s linear cross-fade
    const now = audioCtx.currentTime;

    // 1. Fade out current track if active
    if (currentMusicSource && currentMusicGain) {
        const oldGain = currentMusicGain;
        const oldSource = currentMusicSource;
        oldGain.gain.setValueAtTime(oldGain.gain.value, now);
        oldGain.gain.linearRampToValueAtTime(0, now + fadeTime);
        
        setTimeout(() => {
            try { oldSource.stop(); oldSource.disconnect(); oldGain.disconnect(); } catch(e) {}
        }, fadeTime * 1000);
    }

    // 2. Start new track with fade in
    const newGain = audioCtx.createGain();
    newGain.gain.setValueAtTime(0, now);
    newGain.gain.linearRampToValueAtTime(musicVolume * masterVolume, now + fadeTime);
    newGain.connect(audioCtx.destination); // Connect to master destination

    currentMusicSource = audioCtx.createBufferSource();
    currentMusicSource.buffer = buffer;
    currentMusicSource.loop = true;
    currentMusicSource.connect(newGain);
    currentMusicSource.start(now);

    currentMusicGain = newGain;
    localStorage.setItem('currentMusicTrack', trackName);
    isMusicTransitioning = false;
    console.log(`[AUDIO] Transitioned to: ${trackName}`);
}

/**
 * Stops the Mutation Foundry soundscape.
 */
export function stopMutationSoundscape() {
    if (currentMutationSoundscapeSource) {
        currentMutationSoundscapeSource.stop();
        currentMutationSoundscapeSource = null;
    }
}

window.PlaySound = playSFX;
window.initAudioContext = initAudioContext;
window.playMoodMoteSFX = playMoodMoteSFX;
window.playCharacterVoiceLine = playCharacterVoiceLine;
window.playConnectionSFX = playConnectionSFX;
window.playBlackoutSFX = playBlackoutSFX;
window.playCloakFailureSFX = playCloakFailureSFX;
window.playDistrictStabilizerThrum = playDistrictStabilizerThrum;
window.stopDistrictStabilizerThrum = stopDistrictStabilizerThrum;
window.playComboSFX = playComboSFX;
window.playSabotageReparationSFX = playSabotageReparationSFX;
window.playStaffTrainingSFX = playStaffTrainingSFX;
window.playEcosystemAlertSFX = playEcosystemAlertSFX;
window.playMutationSuccessSFX = playMutationSuccessSFX;
window.playCloakDisruptorSFX = playCloakDisruptorSFX;
window.playBattleStartSFX = playBattleStartSFX;
window.playChallengeWaitSFX = playChallengeWaitSFX;
window.stopChallengeWaitSFX = stopChallengeWaitSFX;
window.playChallengeAcceptedSFX = playChallengeAcceptedSFX;
window.playChallengeDeclinedSFX = playChallengeDeclinedSFX;
window.transitionMusic = transitionMusic;
window.playMutationSoundscape = playMutationSoundscape;
window.playMutationInsuranceHum = playMutationInsuranceHum;
window.stopMutationInsuranceHum = stopMutationInsuranceHum;
window.stopMutationSoundscape = stopMutationSoundscape;
window.playProcedureInterruptedSFX = playProcedureInterruptedSFX;
window.playLongWarningSFX = playLongWarningSFX;
