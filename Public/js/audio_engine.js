// ============================================================================
// audio_engine.js — UI sounds, game sounds, ambient drone
// ----------------------------------------------------------------------------
// All sounds synthesized via Web Audio API (tiny files, instant load).
// Preloaded on wallet connect, not page load.
// Volume tied to ThemeEngine intensity.
// ============================================================================

(function () {
    'use strict';
    let audioCtx = null;
    let masterGain = null;
    let ambientOsc = null;
    let ambientGain = null;
    let isMuted = false;
    let volume = 0.6;

    function init() {
        if (audioCtx) return;
        try {
            audioCtx = new (window.AudioContext || window.webkitAudioContext)();
            masterGain = audioCtx.createGain();
            masterGain.gain.value = volume;
            masterGain.connect(audioCtx.destination);
        } catch (e) {
            console.warn('Audio not available');
        }
    }

    function createOscillator(freq, type, duration, gainVal) {
        if (!audioCtx || isMuted) return;
        const osc = audioCtx.createOscillator();
        const gain = audioCtx.createGain();
        osc.type = type;
        osc.frequency.setValueAtTime(freq, audioCtx.currentTime);
        gain.gain.setValueAtTime(gainVal, audioCtx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.001, audioCtx.currentTime + duration);
        osc.connect(gain);
        gain.connect(masterGain);
        osc.start();
        osc.stop(audioCtx.currentTime + duration);
    }

    // UI Sounds
    function playClick() { createOscillator(800, 'sine', 0.08, 0.15); }
    function playHover() { createOscillator(1200, 'sine', 0.05, 0.08); }
    function playSuccess() { createOscillator(523, 'sine', 0.15, 0.2); setTimeout(() => createOscillator(659, 'sine', 0.15, 0.2), 100); setTimeout(() => createOscillator(784, 'sine', 0.3, 0.2), 200); }
    function playError() { createOscillator(200, 'sawtooth', 0.3, 0.15); }
    function playUnlock() { createOscillator(440, 'sine', 0.1, 0.2); setTimeout(() => createOscillator(660, 'sine', 0.1, 0.2), 80); setTimeout(() => createOscillator(880, 'sine', 0.4, 0.25), 160); }
    function playCoin() { createOscillator(1200, 'square', 0.05, 0.1); setTimeout(() => createOscillator(1600, 'square', 0.08, 0.1), 50); }

    // Game sounds
    function playBattleStart() { createOscillator(300, 'sawtooth', 0.2, 0.15); setTimeout(() => createOscillator(400, 'sawtooth', 0.2, 0.15), 150); setTimeout(() => createOscillator(500, 'sawtooth', 0.4, 0.2), 300); }
    function playHit() { createOscillator(150, 'square', 0.15, 0.2); }
    function playVictory() { [523, 659, 784, 1047].forEach((f, i) => setTimeout(() => createOscillator(f, 'sine', 0.3, 0.2), i * 150)); }
    function playDefeat() { [400, 350, 300, 250].forEach((f, i) => setTimeout(() => createOscillator(f, 'sine', 0.4, 0.15), i * 200)); }

    // Ambient drone
    function startAmbient() {
        if (!audioCtx || ambientOsc) return;
        ambientOsc = audioCtx.createOscillator();
        ambientGain = audioCtx.createGain();
        ambientOsc.type = 'sine';
        ambientOsc.frequency.setValueAtTime(60, audioCtx.currentTime);
        ambientGain.gain.setValueAtTime(isMuted ? 0 : volume * 0.05, audioCtx.currentTime);
        ambientOsc.connect(ambientGain);
        ambientGain.connect(masterGain);
        ambientOsc.start();
    }

    function stopAmbient() {
        if (ambientOsc) { ambientOsc.stop(); ambientOsc = null; }
    }

    function setIntensity(intensity) {
        if (ambientGain) {
            ambientGain.gain.value = isMuted ? 0 : volume * (0.03 + intensity * 0.07);
        }
    }

    function setVolume(val) {
        volume = val;
        if (masterGain) masterGain.gain.value = isMuted ? 0 : val;
        if (ambientGain) ambientGain.gain.value = val * 0.05;
    }

    function toggleMute() {
        isMuted = !isMuted;
        if (masterGain) masterGain.gain.value = isMuted ? 0 : volume;
        return isMuted;
    }

    function preload() {
        init();
        // Create silent buffer to unlock audio context
        if (audioCtx && audioCtx.state === 'suspended') {
            audioCtx.resume();
        }
    }

    window.audioEngine = {
        init: init,
        preload: preload,
        click: playClick,
        hover: playHover,
        success: playSuccess,
        error: playError,
        unlock: playUnlock,
        coin: playCoin,
        battleStart: playBattleStart,
        hit: playHit,
        victory: playVictory,
        defeat: playDefeat,
        startAmbient: startAmbient,
        stopAmbient: stopAmbient,
        setIntensity: setIntensity,
        setVolume: setVolume,
        toggleMute: toggleMute,
    };
})();
