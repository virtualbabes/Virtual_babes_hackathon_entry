import { CONFIG } from './config.js';
import { socket, myClientId } from './network.js';
import { showToast, hideAllOverlays, showMatchPreview, renderCardHTML, movePowerTooltip, hidePowerTooltip, showQuickCastMenu, handleLocalBanUI } from './ui.js';
import { userAddress, walletProvider, signClient } from './wallet.js';
import { collectiveIntelligence } from '../collective-intelligence.js';
import { getCachedEnvoiName, resolveEnvoiName } from './utils.js';
import { initAudioContext } from './audio.js';

// --- Game State Variables ---
export let activeCardId = null; // Tracks the card you clicked in your hand
export let aiThinking = false; // To track if AI is currently performing a move
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

// --- Game Logic Functions ---

export function buildEmptyBoard() {
    const boardContainer = document.getElementById("board-container");
    boardContainer.innerHTML = "";
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

    const state = window.GetGameState();
    if (state.deck.length < 5) { showToast("Deck must have 5 cards", "error"); return; }

    if (!state.in_matchmaking_queue) {
        socket.send(JSON.stringify({
            type: "join_queue",
            payload: { 
                deck: state.deck.map(c => c.id),
                deck_rating: state.deck_rating
            }
        }));
        const btn = document.getElementById("btn-matchmaking");
        btn.disabled = true; // This will be re-enabled by handleMatchmakingUpdate
        btn.innerText = "Joining Queue...";
    } else {
        socket.send(JSON.stringify({ type: "leave_queue" }));
        const btn = document.getElementById("btn-matchmaking");
        btn.disabled = true;
        btn.innerText = "Leaving Queue...";
    }
}

export function handleMatchmakingUpdate(data) {
    const btn = document.getElementById("btn-matchmaking");
    const status = document.getElementById("queue-status");

    if (data.status === "queued") {
        window.SetInMatchmakingQueue(true); // Update WASM state
        btn.innerText = "Leave Queue";
        btn.style.background = "var(--neon-purple)";
        status.innerHTML = `<span class="status-active">SEARCHING FOR OPPONENT...</span>`;
        showToast("🛰️ Entered global matchmaking pool.", "info");
        btn.disabled = false; // Re-enable after status update
    } else if (data.status === "idle") {
        window.SetInMatchmakingQueue(false); // Update WASM state
        btn.innerText = "Join Matchmaking Pool";
        btn.style.background = "";
        status.innerText = "Ready for automatic pairing?";
        btn.disabled = false; // Re-enable after status update
        showToast("🛰️ Left matchmaking pool.", "info");
    } else if (data.status === "match_found") {
        window.SetInMatchmakingQueue(false); // Update WASM state
        btn.innerText = "Join Matchmaking Pool";
        status.innerText = "Ready for automatic pairing?";
        showToast(`⚔️ MATCH FOUND! Battle vs ${data.opponent.substring(0,8)}...`, "success");
        window.SetPhase("Active"); // Optional: logic to transition visual state
        btn.disabled = false; // Re-enable after status update
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
        
        const targetBanned = p.ban_expires && new Date(p.ban_expires) > Date.now();
        const isDisabled = !isMe && (iAmBanned || targetBanned);
        const adminBadge = p.is_admin ? `<span style="color: var(--neon-cyan); font-weight: bold; font-size: 0.8em; margin-left: 5px;">[ADMIN]</span>` : '';
        const btnTitle = targetBanned ? "Player Banned" : (iAmBanned ? "You are Banned" : "Challenge");

        li.innerHTML = `<span>${p.id} ${isMe ? '(You)' : ''} ${adminBadge}</span>
                        <div style="display: flex; gap: 5px;">
                            ${!isMe ? `<button class="outline" style="padding: 5px 10px; font-size: 10px;" ${isDisabled ? 'disabled' : ''} title="${btnTitle}" onclick="sendChallenge('${p.id}')">Challenge</button>` : ''}
                            ${!isMe ? `<button class="outline" style="padding: 5px 10px; font-size: 10px; border-color: var(--neon-purple); color: var(--neon-purple);" onclick="sendSpectate('${p.id}')">Watch</button>` : ''}
                        </div>`;
        list.appendChild(li);
    });
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
    const msgDiv = document.createElement("div");
    msgDiv.className = "chat-msg";
    
    if (sender === "SERVER") msgDiv.classList.add("system");
    
    msgDiv.innerHTML = `<b>${sender}:</b> ${text}`;
    display.appendChild(msgDiv);
    
    // Auto-scroll to bottom
    display.scrollTop = display.scrollHeight;
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

export async function renderMatchHistory() {
    const history = JSON.parse(localStorage.getItem("vbabes_history") || "[]");
    const display = document.getElementById("history-display");
    if (!display || history.length === 0) return;
    
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

        div.innerHTML = `<span style="color: ${color}; font-weight: bold;">${label}</span> vs ${opponentDisplay} <br/> 
                         <small style="opacity: 0.7;">${entry.scores[0]}-${entry.scores[1]} | ${entry.timestamp}</small>`;
        display.appendChild(div);
    });
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
    const state = window.GetGameState();
    const envelope = {
        type: "challenge",
        to_id: currentChallengerId,
        from_id: myClientId, // Ensure from_id is set for server
        payload: { 
            action: "accept",
            to_id: currentChallengerId,
            deck: state.deck.map(c => c.id),
            avatar: state.p1_avatar,
            gloat: state.p1_gloat,
            rules: state.rules
        }
    };

    socket.send(JSON.stringify(envelope));
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
            avatar: state.p1_avatar,
            gloat: state.p1_gloat
        }
    };
    socket.send(JSON.stringify(envelope));
}

export function reportGloat(opponentClientId, gloatText) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        showToast("Cannot report: Not connected to server.", "error");
        return;
    }
    if (!confirm("Are you sure you want to report this gloat message as offensive?")) {
        return;
    }

    const envelope = {
        type: "report_gloat",
        payload: {
            opponent_client_id: opponentClientId,
            gloat_text: gloatText
        }
    };
    socket.send(JSON.stringify(envelope));
    showToast("Gloat message reported. Thank you for helping keep the arena clean!", "success");
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

    socket.send(JSON.stringify(envelope));
    showToast(`👁️ Requesting access to stream...`, "info");
}

export function proceedToWarRoom() {
    if (!spectatorMatchState) return;
    initAudioContext();
    
    document.getElementById("match-preview-overlay").classList.add("hidden");
    window.ResetGame();
    window.SetBoardState(spectatorMatchState);
    window.ForceActive();
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
            avatar: state.p1_avatar || "",
            gloat: state.p1_gloat || "",
            deck: state.deck.map(c => c.id)
        }
    };

    socket.send(JSON.stringify(envelope));
    alert(`Challenge sent to ${targetId}`);
}

export function triggerToggleNetwork() {
    window.toggleNetwork();
    window.syncUI(); // Assuming syncUI is still global or imported
}

export function selectCard(id) {
    activeCardId = id;
    if (window.PlaySelectSound) window.PlaySelectSound();
    window.syncUI("inventory"); // Re-render to show the selected card glowing
}

export function clickGrid(index) {
    const state = window.GetGameState();
    
    // Multiplayer Guard: Only allow move if it's actually our turn
    if (state.phase === "Active" && state.turn !== state.local_player_index) {
        console.warn("It is not your turn!");
        return;
    }

    if (activeCardId === null) {
        return;
    }
    
    const selectedCardId = activeCardId;

    // Execute locally
    const success = window.PlaceCard(index, activeCardId);
    if (success) {
        // If in multiplayer, broadcast the move to the opponent
        if (state.phase === "Active" && currentOpponentId) {
            // Find card power for server verification
            const card = state.deck.find(c => c.id === selectedCardId);
            const envelope = {
                type: "move",
                to_id: currentOpponentId, // This should be the opponent's client ID
                payload: {
                    grid_index: index,
                    card_id: selectedCardId,
                    power: card ? card.power : [0,0,0,0]
                }
            };
            socket.send(JSON.stringify(envelope));
        }
        activeCardId = null; 
        window.syncUI("combat"); // Assuming syncUI is still global or imported
    }
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
    const state = window.GetGameState();
    const item = state.inventory.find(c => c.id === itemId);
    if (!item) return;

    const success = window.ApplyArtifactToBoard(gridIndex, item.artifact);

    if (success) {
        showToast(`⚡ Used ${item.name} on ${state.board[gridIndex].name}!`, "success");
        if (state.multiplayer && currentOpponentId) {
            socket.send(JSON.stringify({
                type: "use_item",
                to_id: currentOpponentId,
                payload: { grid_index: gridIndex, bonus: item.artifact }
            }));
        }
        hidePowerTooltip();
        window.syncUI(); // Assuming syncUI is still global or imported
    }
}

export function showPowerTooltip(e, card, index, state) {
    if (!window.tooltipEl) { // Using window.tooltipEl as it's defined in ui.js
        window.tooltipEl = document.createElement("div");
        window.tooltipEl.className = "power-tooltip";
        document.body.appendChild(window.tooltipEl);
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

        const totalEffectivePower = base + artifactBonus + moodModifier + netFatiguePenalty + loyaltyBonus + netWantedPenalty;
        const grade = window.GetLevelLabelForDisplay(totalEffectivePower);
        
        // Build the HTML for modifiers
        let modifiersHtml = '';
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

    window.tooltipEl.innerHTML = html;
    window.tooltipEl.style.opacity = "1";
    window.tooltipEl.style.pointerEvents = (state.rules?.Artifact_bonus && card.owner === myPlayerIndex) ? "auto" : "none";
    window.tooltipEl.onmouseleave = () => hidePowerTooltip();
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