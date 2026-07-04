import { CONFIG } from './config.js';
import { socket, myClientId } from './network.js';
import { showToast, hideAllOverlays, showMatchPreview, renderCardHTML, movePowerTooltip, hidePowerTooltip, showQuickCastMenu, handleLocalBanUI, tooltipEl } from './ui.js';
import { userAddress, walletProvider, signClient } from './wallet.js';
import { collectiveIntelligence } from '../collective-intelligence.js';
import { getCachedEnvoiName, resolveEnvoiName } from './utils.js';
import { initAudioContext } from './audio.js';

// --- Game State Variables ---
export let activeCardId = null; // Tracks the card you clicked in your hand
export let aiThinking = false; // To track if AI is currently performing a move
export let pendingQuickCastId = null; // PILLAR 3: Prevent redundant interaction
export let lastBoardState = Array(9).fill(null); // Track state to detect captures
export let currentChallengerId = null; // Stores the ID of the player who sent the current challenge
export let currentOpponentId = null;   // The player we are currently battling
export let spectatorMatchState = null; // Stores P1/P2 mapping for spectators
export let myPlayerIndex = 0;          // 0 for P1, 1 for P2
export let matchHistorySaved = false;
export let lastLobbyPlayers = []; // Cache for portfolio valuation, also used for player list
export let lastTauntPhase = null;      // Tracks narrative state to prevent duplicate taunts
export let lastTauntTurn = null;

// --- Game State Setters ---
export const setMyPlayerIndex = (index) => { myPlayerIndex = index; };
export const setCurrentOpponentId = (id) => { currentOpponentId = id; };
export const setSpectatorMatchState = (state) => { spectatorMatchState = state; };
export const setMatchHistorySaved = (saved) => { matchHistorySaved = saved; };
export const setLastLobbyPlayers = (players) => { lastLobbyPlayers = players; };
export const setLastTauntPhase = (phase) => { lastTauntPhase = phase; };
export const setLastTauntTurn = (turn) => { lastTauntTurn = turn; };
export const setPendingQuickCastId = (id) => { pendingQuickCastId = id; };

// --- Game Logic Functions ---

export function buildEmptyBoard() {
    const boardContainer = document.getElementById("board-container");
    boardContainer.innerHTML = "";
    spectatorMatchState = null; // PILLAR 5: Clear spectator state on game reset.
    for(let i=0; i<9; i++) {
        const slot = document.createElement("div");
        slot.className = "grid-slot";
        slot.onclick = () => clickGrid(i);
        boardContainer.appendChild(slot);
    }
}

export function toggleMatchmakingQueue() {
    if (!userAddress) { showToast("Connect wallet first", "error"); return; }
    initAudioContext();

    // PILLAR 5: Deterministic Decision Logic. 
    // Use the authoritative WASM state to determine which action to dispatch.
    const state = window.GetGameState("all");
    if (!state) return;

    if (state.deck.length < 5) { showToast("Deck must have 5 cards", "error"); return; }

    const btn = document.getElementById("btn-matchmaking");
    if (btn) btn.disabled = true; // Throttle UI during transition

    if (!state.in_matchmaking_queue) {
        socket.send(JSON.stringify({
            type: "join_queue",
            payload: { 
                deck: state.deck.map(c => c.id),
                deck_rating: state.deck_rating
            }
        }));
        if (btn) btn.innerText = "Joining Queue...";
    } else {
        socket.send(JSON.stringify({ type: "leave_queue" }));
        if (btn) btn.innerText = "Leaving Queue...";
    }
}

export function handleMatchmakingUpdate(data) {
    if (data.status === "queued") {
        window.SetInMatchmakingQueue(true); // Update WASM state
        showToast("🛰️ Entered global matchmaking pool.", "info");
    } else if (data.status === "idle") {
        window.SetInMatchmakingQueue(false); // Update WASM state
        showToast("🛰️ Left matchmaking pool.", "info");
    } else if (data.status === "match_found") {
        window.SetInMatchmakingQueue(false); // Update WASM state
        const opp = data.opponent ? data.opponent.substring(0, 8) : "Human";
        showToast(`⚔️ MATCH FOUND! Battle vs ${opp}...`, "success");
        window.SetPhase("Active"); // Optional: logic to transition visual state
    }
}

export function updatePlayerList(players) {
    const list = document.getElementById("active-players");
    list.innerHTML = "";
    
    // Check if current user is banned
    const me = players.find(p => p.id === myClientId);
    handleLocalBanUI(me ? me.ban_expires : null);
    const iAmBanned = me && me.ban_expires && new Date(me.ban_expires) > Date.now();

    players.forEach(p => {
        const li = document.createElement("li");
        li.className = "player-item";
        const isMe = p.id === myClientId;
        
        // PILLAR 3: Factional Icon Identification.
        const factionIcon = p.faction === "JUSTICE" ? "⚖️ " : (p.faction === "UNDERWORLD" ? "💀 " : "");

        const targetBanned = p.ban_expires && new Date(p.ban_expires) > Date.now();
        const isDisabled = !isMe && (iAmBanned || targetBanned);
        const adminBadge = p.is_admin ? `<span style="color: var(--neon-cyan); font-weight: bold; font-size: 0.8em; margin-left: 5px;">[ADMIN]</span>` : '';
        const btnTitle = targetBanned ? "Player Banned" : (iAmBanned ? "You are Banned" : "Challenge");

        li.innerHTML = `<span>${factionIcon}${p.id} ${isMe ? '(You)' : ''} ${adminBadge}</span>
                        <div style="display: flex; gap: 5px;">
                            ${!isMe ? `<button class="outline" style="padding: 5px 10px; font-size: 10px;" ${isDisabled ? 'disabled' : ''} title="${btnTitle}" onclick="sendChallenge('${p.id}')">Challenge</button>` : ''}
                            ${!isMe ? `<button class="outline" style="padding: 5px 10px; font-size: 10px; border-color: var(--neon-purple); color: var(--neon-purple);" onclick="sendSpectate('${p.id}')">Watch</button>` : ''}
                        </div>`;
        list.appendChild(li);
    });

    renderMatchHistory(); // Ensure history is refreshed when lobby data updates
}

export function sendChatMessage() {
    const input = document.getElementById("chat-input");
    const text = input.value.trim();
    if (!text || !socket) return;

    const envelope = {
        type: "chat",
        payload: { text: text }
    };
    socket.send(JSON.stringify(envelope));
    input.value = "";
}

export function handleChatKey(e) {
    if (e.key === 'Enter') sendChatMessage();
}

export function renderChatMessage(sender, text) {
    const display = document.getElementById("chat-display");
    if (!display) return;

    const msgDiv = document.createElement("div");
    msgDiv.className = "chat-msg";
    
    const isNpcTaunt = (sender === "SERVER" || sender === "SYSTEM") && text.includes('"');
    if (sender === "SERVER" || sender === "SYSTEM") msgDiv.classList.add("system");

    // PILLAR 5: Visual Feedback. Apply specialized themes for economic enforcement events.
    if (text.includes("STAGNATION FEE")) msgDiv.classList.add("stagnation-fee");
    if (text.includes("GHOST RECLAIM")) msgDiv.classList.add("ghost-reclaim");

    if (isNpcTaunt) {
        msgDiv.innerHTML = `<b>${sender}:</b> <span class="typewriter-content"></span>`;
        display.appendChild(msgDiv);
        const content = msgDiv.querySelector(".typewriter-content");
        let i = 0;
        const typeWriter = () => {
            if (i < text.length) {
                content.textContent += text.charAt(i);
                i++;
                display.scrollTop = display.scrollHeight;
                setTimeout(typeWriter, 30);
            }
        };
        typeWriter();
    } else {
        msgDiv.innerHTML = `<b>${sender}:</b> ${text}`;
        display.appendChild(msgDiv);
        display.scrollTop = display.scrollHeight;
    }
}

export async function saveMatchResult(state) {
    const history = JSON.parse(localStorage.getItem("vbabes_history") || "[]");
    const opponent = currentOpponentId || (state.multiplayer ? "Unknown Human" : "Vbabe Bot");
    
    const newEntry = {
        winner: state.winner,
        scores: state.scores,
        opponent: opponent,
        timestamp: new Date().toLocaleString()
    };

    history.unshift(newEntry);
    if (history.length > 10) history.pop(); // Keep last 10 matches
    localStorage.setItem("vbabes_history", JSON.stringify(history));
    await renderMatchHistory();
}

let isRenderingHistory = false;
export async function renderMatchHistory() {
    if (isRenderingHistory) return;
    const display = document.getElementById("history-display");
    if (!display) return;

    isRenderingHistory = true;
    try {
    let history = [];
    
    // PILLAR 4: Historical Immersion. Prioritize server-authoritative history reconstructed from blockchain.
    const me = lastLobbyPlayers.find(p => p.id === myClientId);
    if (me && me.match_history && me.match_history.length > 0) {
        // Map server format (MatchHistory struct) to display format
        history = me.match_history.map(m => ({
            winner: m.winner_index, // 0=Win, 1=Loss, 2=Draw
            scores: m.scores,
            opponent: m.opponent_wallet,
            timestamp: new Date(m.timestamp).toLocaleString(),
            tournamentId: m.tournament_id,
            matchId: m.match_id,
            receiptTxId: m.receipt_txid, // Sync with authoritative common_types.go JSON tag
            bounty_reward_micro: m.bounty_reward_micro || 0
        }));
    } else {
        // Fallback to local storage for guest sessions or non-indexed wins
        history = JSON.parse(localStorage.getItem("vbabes_history") || "[]");
    }

    if (history.length === 0) return;
    
    // Batch resolve names for wallets in local history
    const wallets = history.map(e => e.opponent).filter(o => o && o.length > 50);
    await Promise.all(wallets.map(w => resolveEnvoiName(w)));
    
    display.innerHTML = "";
    history.forEach(entry => {
        const div = document.createElement("div");
        div.className = "chat-msg";
        const colors = ["var(--neon-green)", "#ff4b4b", "var(--neon-cyan)"]; // Win, Loss, Draw
        const labels = ["WIN", "LOSS", "DRAW"];
        const color = colors[entry.winner] || "white";
        const label = labels[entry.winner] || "END";

        const opponentDisplay = getCachedEnvoiName(entry.opponent);
        let tourneyTag = '';
        if (entry.tournamentId) {
            const shortId = entry.tournamentId.substring(0, 12);
            tourneyTag = `<span class="text-neon-purple" style="font-size: 0.8em; margin-left: 5px;" title="Tournament ID: ${entry.tournamentId}">[${shortId}${entry.matchId ? `:${entry.matchId}` : ''}]</span>`;
        }
        
        let verificationTag = '';
        if (entry.receiptTxId) {
            verificationTag = `<span class="receipt-verify-badge" title="Blockchain Receipt: ${entry.receiptTxId}" style="color: var(--neon-green); margin-left: 5px; font-weight: bold; cursor: help;">✓</span>`;
        }

        const bountyHtml = entry.bounty_reward_micro > 0 ? ` | <span class="text-gold" style="font-size: 0.85em;">🎯 +${(entry.bounty_reward_micro / 1000000).toFixed(2)} Bounty</span>` : '';

        div.innerHTML = `<span style="color: ${color}; font-weight: bold;">${label}${verificationTag}</span> vs ${opponentDisplay}${tourneyTag}${bountyHtml} <br/> 
                         <small style="opacity: 0.7;">${entry.scores[0]}-${entry.scores[1]} | ${entry.timestamp}</small>`;
        display.appendChild(div);
    });
    } finally {
        isRenderingHistory = false;
    }
}

export function showChallengeNotification(challengerId) {
    currentChallengerId = challengerId;
    const challengeOverlay = document.getElementById("challenge-overlay");
    const challengeText = document.getElementById("challenge-text");

    challengeText.innerText = `${challengerId}`;
    challengeOverlay.classList.remove("hidden");
    // Optionally play a sound or vibrate
}

export function acceptChallenge() {
    if (!socket || !currentChallengerId) return;

    // PILLAR 5: Pre-emptive Engine Initialization.
    // Initialize the WASM 'StartMatch' bridge before notifying the server.
    // This ensures the local game state is in 'Active' phase before the challenger sends moves.
    if (window.StartMatch) window.StartMatch(true);

    const state = window.GetGameState();
    const envelope = {
        type: "challenge",
        to_id: currentChallengerId,
        from_id: myClientId, // Ensure from_id is set for server
        payload: { 
            action: "accept",
            to_id: currentChallengerId,
            deck: state.deck.map(c => c.id),
            avatar: state.avatar_url,
            gloat: state.gloat_message,
            rules: state.rules,
            faceplate: state.equipped_faceplate // PILLAR 4: Exchange cosmetics
        }
    };

    socket.send(JSON.stringify(envelope));
    // PILLAR 5: Visual Confirmation.
    // Provide explicit UI feedback that the challenge has been rejected.
    showToast(`❌ Challenge from ${currentChallengerId} declined.`, "info");

    document.getElementById("challenge-overlay").classList.add("hidden");
    currentChallengerId = null;
}

export function sendMatchSync(targetId) {
    const state = window.GetGameState();
    const envelope = {
        type: "challenge",
        to_id: targetId,
        from_id: myClientId, // Ensure from_id is set for server
        payload: { 
            action: "sync_back", 
            deck: state.deck.map(c => c.id),
            avatar: state.avatar_url,
            gloat: state.gloat_message
        }
    };
    socket.send(JSON.stringify(envelope));
}

export function declineChallenge() {
    if (!socket || !currentChallengerId) return;

    const envelope = {
        type: "challenge",
        from_id: myClientId, // Ensure from_id is set for server
        to_id: currentChallengerId,
        payload: { action: "decline" }
    };

    socket.send(JSON.stringify(envelope));
    document.getElementById("challenge-overlay").classList.add("hidden");
    currentChallengerId = null;
}

export function sendSpectate(targetId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;

    const envelope = {
        type: "spectate",
        from_id: myClientId, // Ensure from_id is set for server
        payload: { target_id: targetId }
    };
    spectatorMatchState = null; // Reset for new spectate session

    // PILLAR 4: Interaction Hardening.
    // Reset local selection states to prevent interaction leaks into the spectator view.
    activeCardId = null;
    pendingQuickCastId = null;

    socket.send(JSON.stringify(envelope));
    showToast(`👁️ Requesting access to stream...`, "info");
}

export function proceedToWarRoom() {
    if (!spectatorMatchState) return;
    initAudioContext();
    
    document.getElementById("match-preview-overlay").classList.add("hidden");
    window.ResetGame();
    
    // PILLAR 4: Replay Resilience & Board Integrity.
    // Transition engine to 'Active' phase BEFORE loading the snapshot state.
    // ForceActive() clears the board; loading the authoritative state must follow 
    // to ensure Resonance/Synergy levels are calculated from valid grid data.
    window.ForceActive();
    window.SetBoardState(spectatorMatchState);
    window.syncUI("all"); // Assuming syncUI is still global or imported
}

export function sendChallenge(targetId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    initAudioContext();

    const state = window.GetGameState();
    const envelope = {
        type: "challenge",
        from_id: myClientId, // Ensure from_id is set for server
        to_id: targetId,
        payload: { 
            action: "invite",
            avatar: state.avatar_url || "",
            gloat: state.gloat_message || "",
            deck: state.deck.map(c => c.id),
            faceplate: state.equipped_faceplate || "" // PILLAR 4: Exchange cosmetics
        }
    };

    socket.send(JSON.stringify(envelope));
    if (window.playChallengeWaitSFX) window.playChallengeWaitSFX();
    showToast(`🛰️ Challenge sent to ${targetId}. Waiting for response...`, "info");
}

export function triggerToggleNetwork() {
    window.toggleNetwork();
    window.syncUI(); // Assuming syncUI is still global or imported
}

/**
 * refreshIdentity allows a player to pay 100 $VBV to update their handle and bio.
 * PILLAR 2: Economic Sink (Section 11).
 */
window.refreshIdentity = () => {
    const state = window.GetGameState();
    if (state.virtual_balance < 100) {
        showToast("❌ Insufficient rewards for identity refresh (100 $VBV required).", "error");
        return;
    }

    const newHandle = prompt("Enter new Handle (Public Signature):", state.id);
    if (!newHandle) return;
    const newBio = prompt("Enter new Bio (Sector Metadata):", "Arena combatant.");
    if (newBio === null) return;

    if (!confirm(`💸 DEPLOY IDENTITY REFRESH?\n\nCost: 100 $VBV\nNew Handle: ${newHandle}\n\nProceed?`)) return;

    showToast("🆔 Dispatching identity refresh to sector network...", "info");
    socket.send(JSON.stringify({
        type: "refresh_identity",
        payload: {
            handle: newHandle,
            bio: newBio
        }
    }));
};

export function selectCard(id) {
    // PILLAR 2: Integer Supremacy. Ensure the ID is numeric to prevent 
    // type-mismatch errors during grid placement validation.
    activeCardId = id !== null ? parseInt(id) : null;
    
    if (window.PlaySelectSound) window.PlaySelectSound();
    
    // PILLAR 5: Reactive Authority. 
    // Trigger a full UI sync to ensure selection glows are updated across all views.
    window.syncUI("all"); 
}

export function clickGrid(index) {
    // PILLAR 5: Optimized Tactical Sync. 
    // CRITICAL: Ensure the WASM engine is ready before attempting IPC.
    if (typeof window.GetGameState !== "function" || typeof window.PlaceCard !== "function") {
        console.error("[BATTLE] WASM Interaction Bridge not initialized.");
        return;
    }

    const state = window.GetGameState("combat");
    if (!state) return;
    
    // PILLAR 4: Replay Resilience.
    // Prevent moves if the engine is catching up or desynced.
    if (state.replay_state && state.replay_state !== "SYNCHRONIZED") {
        showToast("⚠️ Reality Reconstruction in progress. Please wait...", "warning");
        return;
    }

    // Multiplayer Guard: Only allow move if it's actually our turn
    if (state.phase === "Active" && state.turn !== state.local_player_index) {
        showToast("It is not your turn!", "warning");
        return;
    }

    // UX Hardening: Inform player if no card is selected.
    if (activeCardId === null) {
        showToast("Select a card from your hand first", "info");
        return;
    }
    
    const selectedCardId = activeCardId;

    // Execute locally
    // PILLAR 2: Integer Supremacy. Ensure card ID is a valid integer for the WASM engine.
    const success = window.PlaceCard(index, parseInt(selectedCardId));
    if (success) {
        // PILLAR 4: State Resilience.
        // Clear the selection immediately to prevent double-spending the card.
        activeCardId = null; 

        // If in multiplayer, broadcast the move to the opponent
        if (state.phase === "Active" && currentOpponentId) {
            // Find card power for server verification
            const card = (state.deck || []).find(c => c.id === parseInt(selectedCardId));
            const envelope = {
                type: "move",
                to_id: currentOpponentId, // This should be the opponent's client ID
                payload: {
                    grid_index: index,
                    card_id: parseInt(selectedCardId),
                    power: card ? card.power : [0,0,0,0]
                }
            };
            socket.send(JSON.stringify(envelope));
        }

        // PILLAR 4: State Resilience.
        // Clear the selection immediately to prevent double-spending the card.
        activeCardId = null; 
        window.syncUI("combat"); // Assuming syncUI is still global or imported
    } else {
        // PILLAR 3: UX Hardening.
        // If local placement failed (e.g. slot occupied), provide tactical feedback.
        showToast("Slot already occupied. Choose an empty coordinate.", "warning");
    }
}

/**
 * rejoinActiveMatch handles match restoration during a warm boot.
 * PILLAR 4: Replay Resilience.
 */
export function rejoinActiveMatch() {
    // PILLAR 5: Authoritative Decision.
    const state = window.GetGameState("all");
    if (!state || state.phase !== "Active") return;

    console.log(`[MATCH] Rejoining active engagement: ${state.match_id || 'PvP'}`);

    // PILLAR 6: Beacon Integrity Check.
    // Fetch the raw beacon from storage to verify the cryptographic baseline.
    // This ensures that the re-hydrated WASM state accurately matches the authoritative 
    // server outcome at the time of the last local save.
    const cachedBeacon = localStorage.getItem("vbabes_state_beacon");
    if (cachedBeacon) {
        try {
            const beacon = JSON.parse(cachedBeacon);
            const engineHash = state.last_verified_state_hash;
            const authoritativeHash = beacon.profile.last_verified_state_hash;

            if (engineHash && authoritativeHash && engineHash !== authoritativeHash) {
                console.warn("[MATCH] Cryptographic drift detected at save-point. Forcing recovery cycle.");
                if (window.InitiateRecovery) {
                    window.InitiateRecovery();
                    return; // InitiateRecovery triggers requestMatchSync internally
                }
            }
        } catch (e) { console.error("[MATCH] Beacon verification failed.", e); }
    }

    // PILLAR 4: Catch-up Synchronization.
    // Always request missing frames from the server to bridge the gap while offline.
    // Doing this silently (without InitiateRecovery) if hashes match prevents UI flicker.
    import('./network.js').then(m => m.requestMatchSync());

    // PILLAR 4: UI Continuity.
    // Force a full UI sync to transition to the combat arena immediately.
    window.syncUI("all");
}

export function calculateDeckRating(deck) {
    if (deck.length === 0) {
        return "[Z]";
    }

    let maxBin = -1;
    // 1. Find the highest card tier (bin) in the deck
    for (const card of deck) {
        let highestPower = 0;
        for (const p of card.power) {
            if (p > highestPower) {
                highestPower = p;
            }
        }
        let bin = Math.floor((highestPower - 1) / 100);
        // Safety Clamping: Ensure bin stays within 0-25 range (Z-A)
        if (bin < 0) bin = 0;
        if (bin > 25) bin = 25;

        if (bin > maxBin) {
            maxBin = bin;
        }
    }

    if (maxBin === -1) {
        return "[Z]";
    }

    // 2. Map maxBin to Letter
    const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
    const baseLetter = alphabet[25 - maxBin]; // Get the letter for the start of the bin

    // 3. Count how many cards share this highest tier
    let plusCount = 0;
    for (const card of deck) {
        let highestPower = 0;
        for (const p of card.power) {
            if (p > highestPower) {
                highestPower = p;
            }
        }
        let bin = Math.floor((highestPower - 1) / 100);
        // Maintain identical clamping for accurate plusCount comparison
        if (bin < 0) bin = 0;
        if (bin > 25) bin = 25;

        if (bin === maxBin) {
            plusCount++;
        }
    }

    // 4. Construct Suffix
    let suffix = "";
    for (let i = 0; i < plusCount; i++) {
        suffix += "+";
    }

    return `[${baseLetter}${suffix}]`;
}

export async function executeQuickCast(itemId, gridIndex) {
    // PILLAR 5: Bridge Integrity.
    if (typeof window.ApplyArtifactToBoard !== "function") return;
    if (pendingQuickCastId === itemId) return;

    const state = window.GetGameState("all");
    if (!state) return;

    // PILLAR 4: Replay Resilience.
    if (state.replay_state && state.replay_state !== "SYNCHRONIZED") {
        showToast("⚠️ Reality Reconstruction in progress. Please wait...", "warning");
        return;
    }

    pendingQuickCastId = itemId;

    const item = state.inventory.find(c => c.id === itemId);
    if (!item) return;

    const success = window.ApplyArtifactToBoard(gridIndex, item.artifact);

    if (success) {
        const targetName = state.board[gridIndex] ? state.board[gridIndex].name : "Asset";
        showToast(`⚡ Used ${item.name} on ${targetName}!`, "success");
        if (state.multiplayer && currentOpponentId) {
            socket.send(JSON.stringify({
                type: "use_item",
                to_id: currentOpponentId,
                payload: { grid_index: gridIndex, bonus: item.artifact }
            }));
        }
        hidePowerTooltip();
        window.syncUI("combat"); // PILLAR 5: Minimize serialization overhead during match turns
    } else {
        // PILLAR 3: Interaction Hardening.
        // If the application fails (e.g. card removed or grid desync), close the menu 
        // immediately to prevent repetitive error clicks (toast-spam).
        pendingQuickCastId = null;
        showToast("❌ Quick-Cast failed: Invalid sector or target missing.", "warning");
        hidePowerTooltip();
    }
}

export function showPowerTooltip(e, card, index, state) {
    // PILLAR 5: UI Consistency. 
    // Use the exported tooltip element from ui.js rather than checking a window property.
    let targetTooltip = tooltipEl; 
    if (!targetTooltip) {
        targetTooltip = document.createElement("div");
        targetTooltip.className = "power-tooltip";
        document.body.appendChild(targetTooltip);
    }

    const tileMood = state.board_moods ? state.board_moods[index] : "Neutral";
    const moodWeaknesses = { "Volatile": "Serene", "Serene": "Spirited", "Spirited": "Grounded", "Grounded": "Volatile" };
    
    let html = `<div style="color: var(--neon-cyan); font-weight: bold; margin-bottom: 8px; border-bottom: 1px solid var(--neon-cyan); padding-bottom: 5px;">${card.name.toUpperCase()} DATA</div>`;
    
    const sides = ["TOP", "RIGHT", "BOTTOM", "LEFT"];
    
    // Get player stats for the card owner to calculate player-level modifiers
    const ownerPlayerIndex = card.owner;
    const ownerWantedLevel = (ownerPlayerIndex === 0 ? state.p1_wanted_level : state.p2_wanted_level) || 0;
    const ownerCunning = (ownerPlayerIndex === 0 ? state.p1_cunning : state.p2_cunning) || 0;
    const ownerNurturing = (ownerPlayerIndex === 0 ? state.p1_nurturing : state.p2_nurturing) || 0;
    const hasRegBoost = (ownerPlayerIndex === 0 ? state.p1_regional_boost : state.p2_regional_boost) || false;
    const hasCoalitionBoost = (ownerPlayerIndex === 0 ? state.p1_coalition_boost : state.p2_coalition_boost) || false;

    // Calculate player-level modifiers once
    let netWantedPenalty = 0;
    if (ownerWantedLevel > 0) {
        const baseWantedPenalty = ownerWantedLevel * 5;
        const mitigation = ownerCunning * 2;
        netWantedPenalty = -(baseWantedPenalty - Math.min(mitigation, baseWantedPenalty));
    }

    sides.forEach((side, sideIndex) => {
        const base = card.power[sideIndex];
        const artifactBonus = card.artifact || 0;
        
        // PILLAR 1: Coalition & Regional Power Boosts (Tactical UI Sync)
        // Prioritize the 10% Coalition boost for allies. Direct regional members get 5%.
        let coalitionBonus = 0;
        let regBonus = 0;
        if (hasCoalitionBoost) {
            coalitionBonus = Math.floor((base + artifactBonus) * 0.10);
        } else if (hasRegBoost) {
            regBonus = Math.floor((base + artifactBonus) * 0.05);
        }

        let moodModifier = 0;
        if (state.rules?.Elemental_sync && tileMood !== "Neutral" && card.mood && card.mood !== "Neutral") {
            if (card.mood === tileMood) {
                moodModifier = 50; // Match bonus
            } else if (moodWeaknesses[card.mood] === tileMood) {
                moodModifier = -50; // Weakness penalty
            }
        }

        let netFatiguePenalty = 0;
        if (card.fatigue > 50) {
            const baseFatiguePenalty = (card.fatigue - 50);
            const reduction = ownerNurturing;
            if (reduction > baseFatiguePenalty) { reduction = baseFatiguePenalty; }
            netFatiguePenalty = -(baseFatiguePenalty - reduction);
        }

        const loyaltyBonus = card.loyalty >= 100 ? 25 : 0;

        const totalEffectivePower = base + regBonus + coalitionBonus + artifactBonus + moodModifier + netFatiguePenalty + loyaltyBonus + netWantedPenalty;
        const grade = window.GetLevelLabelForDisplay(totalEffectivePower);
        
        // Build the HTML for modifiers
        let modifiersHtml = '';
        if (regBonus !== 0) {
            modifiersHtml += `<span style="color: var(--neon-purple)">+${regBonus}R</span> `;
        }
        if (coalitionBonus !== 0) {
            modifiersHtml += `<span style="color: var(--neon-blue)">+${coalitionBonus}C</span> `;
        }
        if (artifactBonus !== 0) {
            modifiersHtml += `<span style="color: ${artifactBonus > 0 ? 'var(--neon-cyan)' : '#ff4b4b'}">${artifactBonus > 0 ? '+' : ''}${artifactBonus}A</span> `;
        }
        if (moodModifier !== 0) {
            modifiersHtml += `<span style="color: ${moodModifier > 0 ? 'var(--neon-green)' : '#ff4b4b'}">${moodModifier > 0 ? '+' : ''}${moodModifier}M</span> `;
        }
        if (netFatiguePenalty !== 0) {
            modifiersHtml += `<span style="color: #ff4b4b">${netFatiguePenalty}F</span> `;
        }
        if (loyaltyBonus !== 0) {
            modifiersHtml += `<span style="color: var(--neon-cyan)">+${loyaltyBonus}L</span> `;
        }
        if (netWantedPenalty !== 0) {
            modifiersHtml += `<span style="color: #ff4b4b">${netWantedPenalty}W</span> `;
        }

        html += `
            <div class="tooltip-row">
                <span style="opacity: 0.7;">${side}:</span>
                <span style="display: flex; align-items: center; gap: 5px;">
                    <span>${base}</span>
                    ${modifiersHtml ? `<span style="font-size: 0.8em; opacity: 0.8;">(${modifiersHtml.trim()})</span>` : ''}
                    <span>=</span>
                    <b style="color: var(--neon-cyan)">${totalEffectivePower} (${grade})</b>
                </span>
            </div>
        `;
    });

    if (state.rules?.Artifact_bonus && card.owner === myPlayerIndex) {
        html += `
            <div class="tooltip-quickcast">
                <button onclick="event.stopPropagation(); showQuickCastMenu(${index})">
                    ⚡ QUICK-CAST ITEM
                </button>
            </div>
        `;
    }

    if (card.mood && card.mood !== "Neutral") {
        html += `<div style="margin-top: 8px; font-size: 10px; opacity: 0.6; text-align: center;">MOOD: ${card.mood.toUpperCase()} vs TILE: ${tileMood.toUpperCase()}</div>`;
    }

    targetTooltip.innerHTML = html;
    targetTooltip.style.opacity = "1";
    targetTooltip.style.pointerEvents = (state.rules?.Artifact_bonus && card.owner === myPlayerIndex) ? "auto" : "none";
    targetTooltip.onmouseleave = hidePowerTooltip; // PILLAR 5: GC Optimization. Assign function directly.
    movePowerTooltip(e);
}

// Expose to window for inline HTML calls
window.toggleMatchmakingQueue = toggleMatchmakingQueue;
window.sendChatMessage = sendChatMessage;
window.handleChatKey = handleChatKey;
window.saveMatchResult = saveMatchResult;
window.renderMatchHistory = renderMatchHistory;
window.showChallengeNotification = showChallengeNotification;
window.acceptChallenge = acceptChallenge;
window.sendMatchSync = sendMatchSync;
window.reportGloat = reportGloat;
window.declineChallenge = declineChallenge;
window.sendSpectate = sendSpectate;
window.proceedToWarRoom = proceedToWarRoom;
window.sendChallenge = sendChallenge;
window.triggerToggleNetwork = triggerToggleNetwork;
window.selectCard = selectCard;
window.clickGrid = clickGrid;
window.executeQuickCast = executeQuickCast;
window.showPowerTooltip = showPowerTooltip;
window.buildEmptyBoard = buildEmptyBoard;

/**
 * openSpectatorWagerOverlay allows a spectator to place a wager on a match.
 * PILLAR 2: Industrial Loop (Spectator Siphon).
 */
export function openSpectatorWagerOverlay(matchID, p1Wallet, p2Wallet) {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "spectator-wager-overlay";
    overlay.className = "overlay";

    const p1Name = getCachedEnvoiName(p1Wallet);
    const p2Name = getCachedEnvoiName(p2Wallet);

    overlay.innerHTML = `
        <div class="economy-panel glass-panel medium animate-modal w-450 border-neon-cyan">
            <div class="market-header">
                <span class="market-title">SPECTATOR WAGER</span>
                <div class="access-level">MATCH: ${matchID.substring(0, 8)}...</div>
            </div>
            <div class="p-20 text-center">
                <p class="opacity-7 mb-20 font-size-0-85em">
                    Place your $VBV wager on the outcome of this match. A <b>2% House Cut</b> applies to all payouts.
                </p>
                <div class="flex-col gap-10 mb-20">
                    <div class="flex-row justify-center gap-10">
                        <label class="flex-row align-center glass-panel p-10 m-0 flex-1 pointer">
                            <input type="radio" name="bet-on-player" value="${p1Wallet}" class="mr-10">
                            <span class="text-neon-cyan font-bold">${p1Name}</span>
                        </label>
                        <label class="flex-row align-center glass-panel p-10 m-0 flex-1 pointer">
                            <input type="radio" name="bet-on-player" value="${p2Wallet}" class="mr-10">
                            <span class="text-neon-cyan font-bold">${p2Name}</span>
                        </label>
                    </div>
                    <input type="number" id="wager-amount-input" class="glass-input w-full text-center font-size-1-5em" placeholder="0.00 $VBV" step="1" min="1" oninput="window.updatePotentialPayout()">
                </div>
                
                <div class="flex-row justify-between align-center mb-15 p-10 bg-dark-overlay rounded">
                    <span class="font-bold letter-spacing-1">POTENTIAL PAYOUT</span>
                    <b id="potential-payout-display" class="text-neon-green font-size-1-5em" style="text-shadow: 0 0 10px var(--neon-green);">0.00 $VBV</b>
                </div>

                <div class="flex-row gap-10">
                    <button class="outline w-full" onclick="document.getElementById('spectator-wager-overlay').remove()">CANCEL</button>
                    <button class="w-full bg-neon-green text-dark font-bold" onclick="submitSpectatorWager('${matchID}')">PLACE WAGER</button>
                </div>
            </div>
        </div>`;
    document.body.appendChild(overlay);

    // Initial payout calculation
    window.updatePotentialPayout = () => {
        const wagerInput = document.getElementById("wager-amount-input");
        const payoutDisplay = document.getElementById("potential-payout-display");
        const wager = parseFloat(wagerInput.value);
        if (isNaN(wager) || wager <= 0) {
            payoutDisplay.innerText = "0.00 $VBV";
            return;
        }
        // 2% House Cut
        const potentialPayout = wager * (1 - 0.02);
        payoutDisplay.innerText = `${potentialPayout.toFixed(2)} $VBV`;
    };
    window.updatePotentialPayout(); // Call once on load
}

/**
 * submitSpectatorWager sends the wager request to the backend.
 */
export async function submitSpectatorWager(matchID) {
    const wagerInput = document.getElementById("wager-amount-input");
    const betOnPlayerRadio = document.querySelector('input[name="bet-on-player"]:checked');

    if (!userAddress) return showToast("❌ Connect wallet first.", "error");
    if (!betOnPlayerRadio) return showToast("❌ Select a player to bet on.", "error");
    
    const wagerAmount = parseFloat(wagerInput.value);
    if (isNaN(wagerAmount) || wagerAmount <= 0) return showToast("❌ Enter a valid wager amount.", "error");

    const state = window.GetGameState();
    if (state.virtual_balance < wagerAmount) return showToast("❌ Insufficient rewards for wager.", "error");

    showToast(`💰 Placing ${wagerAmount.toFixed(2)} $VBV wager...`, "info");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/match/wager`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                spectator_wallet: userAddress,
                match_id: matchID,
                bet_on_wallet: betOnPlayerRadio.value,
                wager_micro: Math.round(wagerAmount * 1000000) // Convert to micro-units
            })
        });

        if (response.ok) {
            showToast("✅ Wager placed successfully!", "success");
            document.getElementById("spectator-wager-overlay")?.remove();
            if (window.syncUI) window.syncUI("all"); // Refresh UI to reflect balance change
        } else {
            const err = await response.text();
            showToast(`❌ Wager Failed: ${err}`, "error");
        }
    } catch (err) {
        showToast(`❌ Network Error: ${err.message}`, "error");
    }
}
window.rejoinActiveMatch = rejoinActiveMatch;