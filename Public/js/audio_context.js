// AudioContextManager - Contextual Ambient Music System (Phase 6)
// Manages game-phase-aware ambient audio transitions with fade-in/fade-out cross-fading.
// Bridges syncUI phase transitions to transitionMusic() in audio.js.

class AudioContextManager {
    constructor() {
        this.currentContext = 'menu'; // Current audio context phase
        this.enabled = localStorage.getItem('contextualAmbients') === 'true';
        this.pendingTransition = null;
        
        // Context-to-trackName mapping matching MUSIC_PHASE_MAP in audio.js
        this.contextToPhase = {
            'menu':           'DISCONNECTED',
            'lobby':          'Lobby',
            'casual_2p':      'Active_Casual',
            'quick_play':     'Active_Quick',
            'tournament':     'Active_Tournament',
            'tournament_lobby': 'TournamentLobby',
            'combat':         'Active',
            'finished':       'Finished'
        };

        this._boundPhaseHandler = null;
        this._boundConnHandler = null;
        this._bindPhaseListeners();
    }
    
    _bindPhaseListeners() {
        this._boundPhaseHandler = (e) => {
            if (!this.enabled) return;
            this.transitionToContext(e.detail.context, e.detail.reason);
        };
        
        this._boundConnHandler = (e) => {
            if (e.detail.state === 'disconnected' && this.currentContext !== 'menu') {
                this.transitionToContext('menu', 'disconnect');
            } else if (e.detail.state === 'connected' && this.currentContext === 'menu') {
                this.transitionToContext('lobby', 'reconnect');
            }
        };
        
        window.addEventListener('game_phase_change', this._boundPhaseHandler);
        window.addEventListener('connection_state_change', this._boundConnHandler);
    }
    
    /**
     * Transition to a new audio context. Uses transitionMusic from audio.js with track resolution.
     * @param {string} context - Target context key
     * @param {string} reason - Why transition occurred
     */
    transitionToContext(context, reason = 'manual') {
        if (!this.enabled) return;
        if (context === this.currentContext) return;
        
        // Resolve the phase string from music_phase_map for transitionMusic
        const phaseKey = this.contextToPhase[context];
        if (!phaseKey) {
            console.warn(`[AudioContextManager] Unknown context: ${context}`);
            return;
        }

        const trackName = phaseKey; // transitionMusic uses MUSIC_PHASE_MAP internally
        
        console.log(`[AudioContextManager] Transitioning to '${context}' (${reason}) -> phase: ${trackName}`);
        
        if (window.transitionMusic) {
            window.transitionMusic(trackName);
        }
        
        this.currentContext = context;
    }
    
    /**
     * Get current playing context info (for debugging/state inspection)
     */
    getCurrentContext() {
        return {
            context: this.currentContext,
            enabled: this.enabled
        };
    }
    
    /**
     * Toggle contextual ambients on/off and persist to localStorage
     */
    toggle(enabled) {
        this.enabled = !!enabled;
        localStorage.setItem('contextualAmbients', String(this.enabled));
        
        if (!this.enabled) {
            if (window.stopMutationSoundscape) window.stopMutationSoundscape();
            console.log('[AudioContextManager] Contextual ambients disabled');
        } else {
            console.log('[AudioContextManager] Contextual ambients enabled');
        }
        
        return this.enabled;
    }
    
    /**
     * Mute/unmute all ambient audio (respects global mute state)
     */
    toggleMute(muted) {
        if (muted) {
            this.stopAll();
        }
        return muted;
    }
    
    /**
     * Stop all ambient audio (music only, not SFX)
     */
    stopAll() {
        // Music tracks are auto-stopped by transitionMusic cross-fade logic
        // No explicit stop needed for looping music sources in this architecture
    }
    
    /**
     * Cleanup and destroy the manager
     */
    destroy() {
        window.removeEventListener('game_phase_change', this._boundPhaseHandler);
        window.removeEventListener('connection_state_change', this._boundConnHandler);
        this._boundPhaseHandler = null;
        this._boundConnHandler = null;
    }
}

// Global singleton instance
let audioContextManagerInstance = null;

/**
 * Initialize the global AudioContextManager
 * @returns {AudioContextManager} The manager instance
 */
function initAudioContextManager() {
    if (audioContextManagerInstance) return audioContextManagerInstance;
    audioContextManagerInstance = new AudioContextManager();
    console.log('[AudioContextManager] Initialized');
    return audioContextManagerInstance;
}

/**
 * Get the current AudioContextManager instance
 * @returns {AudioContextManager|null} The manager instance or null
 */
function getAudioContextManager() {
    return audioContextManagerInstance;
}

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        AudioContextManager,
        initAudioContextManager,
        getAudioContextManager
    };
}
