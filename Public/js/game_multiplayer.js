// ============================================================================
// game_multiplayer.js — Real-Time P2P Multiplayer Controller
// ----------------------------------------------------------------------------
// Integrates with existing game.js and network.js WebSocket infrastructure
// Adds real-time move sync, latency tracking, in-match chat, and connection status
// ============================================================================

import { socket as networkSocket } from './network.js';

var API_BASE = '/api';

// Multiplayer state
let mpConnected = false;
let mpMatchId = null;
let mpOpponentId = null;
let mpOpponentName = 'Opponent';
let mpOpponentAvatar = '🤖';
let mpIsHost = false;
let mpLastPing = 0;
let mpLatency = 0;
let mpReconnectAttempts = 0;
let mpChatMessages = [];
let mpOpponentConnected = false;
let mpGameState = 'waiting'; // waiting, active, finished
let mpBoardState = Array(9).fill(null);
let mpSelectedCard = null;
let mpMyTurn = false;
let mpMyDeck = [];
let mpOpponentDeck = [];

// --- Init ---
export function initGameMultiplayer() {
    const el = document.getElementById('wd-game');
    if (!el) return;
    renderMultiplayerLobby(el);
}

// --- Lobby / Matchmaking ---
function renderMultiplayerLobby(el) {
    el.innerHTML = `
        <div class="mp-container">
            <div class="mp-header">
                <span class="mp-icon">⚔️</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Multiplayer Arena</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Real-time P2P card battles</p>
                </div>
                <div class="mp-connection-status" id="mp-connection-status">
                    <span class="mp-status-dot offline"></span>
                    <span class="mp-status-text">Offline</span>
                </div>
            </div>

            <!-- Connection Panel -->
            <div class="mp-section">
                <h4 style="color:#e5e7eb;">Connection</h4>
                <div class="mp-connection-panel">
                    <div class="mp-wallet-status">
                        <span class="mp-wallet-label">Wallet:</span>
                        <span class="mp-wallet-address" id="mp-wallet-address">Not connected</span>
                    </div>
                    <div class="mp-server-status">
                        <span class="mp-server-label">Server:</span>
                        <span class="mp-server-indicator" id="mp-server-indicator">Disconnected</span>
                    </div>
                    <div class="mp-latency">
                        <span class="mp-latency-label">Latency:</span>
                        <span class="mp-latency-value" id="mp-latency-value">-- ms</span>
                    </div>
                </div>
            </div>

            <!-- Matchmaking -->
            <div class="mp-section">
                <h4 style="color:#e5e7eb;">Matchmaking</h4>
                <div class="mp-matchmaking">
                    <button class="vbt-btn vbt-btn-primary" id="mp-find-match-btn">🔍 Find Match</button>
                    <button class="vbt-btn vbt-btn-secondary" id="mp-create-room-btn">🏠 Create Room</button>
                    <button class="vbt-btn vbt-btn-secondary" id="mp-join-room-btn">🚪 Join Room</button>
                </div>
                <div class="mp-matchmaking-status" id="mp-matchmaking-status"></div>
            </div>

            <!-- Active Match -->
            ${mpMatchId ? renderActiveMatch() : ''}

            <!-- Quick Match History -->
            <div class="mp-section">
                <h4 style="color:#e5e7eb;">Recent Matches</h4>
                <div class="mp-history" id="mp-history">
                    ${renderMatchHistory()}
                </div>
            </div>
        </div>
    `;
    attachMultiplayerListeners(el);
    updateConnectionStatus();
}

function renderActiveMatch() {
    return `
        <div class="mp-section mp-active-match">
            <h4 style="color:#e5e7eb;">Active Match</h4>
            <div class="mp-match-header">
                <div class="mp-player mp-player-you">
                    <span class="mp-player-avatar">👤</span>
                    <span class="mp-player-name">You</span>
                    <span class="mp-player-score" id="mp-player-score">0</span>
                </div>
                <div class="mp-vs">VS</div>
                <div class="mp-player mp-player-opponent">
                    <span class="mp-player-avatar">${mpOpponentAvatar}</span>
                    <span class="mp-player-name">${mpOpponentName}</span>
                    <span class="mp-player-score" id="mp-opponent-score">0</span>
                </div>
            </div>

            <!-- Game Board -->
            <div class="mp-board-wrapper">
                <div class="mp-board" id="mp-board">
                    ${renderBoardTiles()}
                </div>
            </div>

            <!-- Player Hand -->
            <div class="mp-hand-section">
                <h5>Your Hand</h5>
                <div class="mp-hand" id="mp-hand">
                    ${renderPlayerHand()}
                </div>
            </div>

            <!-- Turn Indicator -->
            <div class="mp-turn-indicator" id="mp-turn-indicator">
                ${mpMyTurn ? 'Your Turn' : 'Opponent\'s Turn'}
            </div>

            <!-- Match Actions -->
            <div class="mp-match-actions">
                <button class="vbt-btn vbt-btn-secondary" id="mp-chat-btn">💬 Chat</button>
                <button class="vbt-btn vbt-btn-secondary" id="mp-reaction-btn">😀 React</button>
                <button class="vbt-btn vbt-btn-danger" id="mp-forfeit-btn">🏳️ Forfeit</button>
            </div>

            <!-- Chat Panel -->
            <div class="mp-chat-panel" id="mp-chat-panel">
                <div class="mp-chat-messages" id="mp-chat-messages">
                    ${renderChatMessages()}
                </div>
                <div class="mp-chat-input">
                    <input type="text" id="mp-chat-input" placeholder="Type a message..." />
                    <button class="vbt-btn vbt-btn-primary" id="mp-chat-send">Send</button>
                </div>
            </div>
        </div>
    `;
}

function renderBoardTiles() {
    return Array(9).fill(0).map((_, i) => {
        const tile = mpBoardState[i];
        return `
            <div class="mp-board-tile ${tile ? 'occupied' : 'empty'}" data-index="${i}">
                ${tile ? `<span class="mp-tile-card">${tile.icon || '🃏'}</span>` : ''}
            </div>
        `;
    }).join('');
}

function renderPlayerHand() {
    if (mpMyDeck.length === 0) {
        return '<p style="color:#90a4ae;">No cards in hand</p>';
    }
    return mpMyDeck.map((card, i) => `
        <div class="mp-hand-card ${mpSelectedCard === i ? 'selected' : ''}" data-index="${i}">
            <span class="mp-card-icon">${card.icon || '🃏'}</span>
            <span class="mp-card-power">${card.power || 0}</span>
        </div>
    `).join('');
}

function renderChatMessages() {
    if (mpChatMessages.length === 0) {
        return '<p style="color:#90a4ae;">No messages yet</p>';
    }
    return mpChatMessages.map(msg => `
        <div class="mp-chat-msg ${msg.isMe ? 'me' : 'opponent'}">
            <span class="mp-chat-sender">${msg.sender}:</span>
            <span class="mp-chat-text">${msg.text}</span>
        </div>
    `).join('');
}

function renderMatchHistory() {
    // Fetch from matchHistory via game.js
    const history = [];
    if (typeof window.getMatchHistory === 'function') {
        const matches = window.getMatchHistory();
        matches.slice(0, 5).forEach(m => {
            history.push(`
                <div class="mp-history-item">
                    <span class="mp-history-opponent">vs ${m.opponent?.substring(0, 8) || 'Unknown'}</span>
                    <span class="mp-history-result ${m.won ? 'win' : 'loss'}">${m.won ? 'W' : 'L'}</span>
                    <span class="mp-history-score">${m.scores?.[0] || 0}-${m.scores?.[1] || 0}</span>
                </div>
            `);
        });
    }
    return history.length > 0 ? history.join('') : '<p style="color:#90a4ae;">No recent matches</p>';
}

function attachMultiplayerListeners(el) {
    // Find Match
    const findMatchBtn = el.querySelector('#mp-find-match-btn');
    if (findMatchBtn) findMatchBtn.addEventListener('click', () => {
        findMatch();
    });

    // Create Room
    const createRoomBtn = el.querySelector('#mp-create-room-btn');
    if (createRoomBtn) createRoomBtn.addEventListener('click', () => {
        createRoom();
    });

    // Join Room
    const joinRoomBtn = el.querySelector('#mp-join-room-btn');
    if (joinRoomBtn) joinRoomBtn.addEventListener('click', () => {
        joinRoom();
    });

    // Chat send
    const chatSend = el.querySelector('#mp-chat-send');
    if (chatSend) chatSend.addEventListener('click', () => {
        sendChatMessage();
    });

    const chatInput = el.querySelector('#mp-chat-input');
    if (chatInput) chatInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') sendChatMessage();
    });

    // Forfeit
    const forfeitBtn = el.querySelector('#mp-forfeit-btn');
    if (forfeitBtn) forfeitBtn.addEventListener('click', () => {
        forfeitMatch();
    });

    // Board tile clicks
    el.querySelectorAll('.mp-board-tile').forEach(tile => {
        tile.addEventListener('click', () => {
            const index = parseInt(tile.dataset.index);
            placeCard(index);
        });
    });

    // Hand card clicks
    el.querySelectorAll('.mp-hand-card').forEach(card => {
        card.addEventListener('click', () => {
            const index = parseInt(card.dataset.index);
            selectCard(index);
        });
    });
}

// --- WebSocket Integration ---
function getSocket() {
    if (networkSocket) return networkSocket;
    if (typeof window !== 'undefined' && window.socket) return window.socket;
    return null;
}

function findMatch() {
    const socket = getSocket();
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        if (window.showToast) window.showToast('⚠️ Not connected to server', 'error');
        return;
    }

    const state = window.GetGameState ? window.GetGameState('all') : null;
    if (!state || (state.deck?.length ?? 0) < 5) {
        if (window.showToast) window.showToast('⚠️ Deck must have 5 cards', 'error');
        return;
    }

    socket.send(JSON.stringify({
        type: 'join_queue',
        payload: {
            deck: state.deck.map(c => c.id),
            deck_rating: state.deck_rating
        }
    }));

    updateMatchmakingStatus('Searching for opponent...');
    if (window.showToast) window.showToast('🔍 Searching for match...', 'info');
}

function createRoom() {
    const socket = getSocket();
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        if (window.showToast) window.showToast('⚠️ Not connected to server', 'error');
        return;
    }

    const roomId = Math.random().toString(36).substring(2, 8).toUpperCase();
    socket.send(JSON.stringify({
        type: 'create_room',
        payload: { room_id: roomId }
    }));

    updateMatchmakingStatus(`Room created: ${roomId} — waiting for opponent...`);
    if (window.showToast) window.showToast(`🏠 Room ${roomId} created!`, 'success');
}

function joinRoom() {
    const roomId = prompt('Enter room code:');
    if (!roomId) return;

    const socket = getSocket();
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        if (window.showToast) window.showToast('⚠️ Not connected to server', 'error');
        return;
    }

    socket.send(JSON.stringify({
        type: 'join_room',
        payload: { room_id: roomId.toUpperCase() }
    }));

    updateMatchmakingStatus(`Joining room ${roomId}...`);
    if (window.showToast) window.showToast(`🚪 Joining room ${roomId}...`, 'info');
}

function placeCard(tileIndex) {
    if (!mpMyTurn) {
        if (window.showToast) window.showToast('⚠️ Not your turn!', 'warning');
        return;
    }

    if (mpBoardState[tileIndex] !== null) {
        if (window.showToast) window.showToast('⚠️ Tile already occupied!', 'warning');
        return;
    }

    if (mpSelectedCard === null) {
        if (window.showToast) window.showToast('⚠️ Select a card first!', 'warning');
        return;
    }

    const socket = getSocket();
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        if (window.showToast) window.showToast('⚠️ Not connected!', 'error');
        return;
    }

    const card = mpMyDeck[mpSelectedCard];
    socket.send(JSON.stringify({
        type: 'mp_move',
        payload: {
            match_id: mpMatchId,
            card_id: card.id,
            tile_index: tileIndex
        }
    }));

    // Optimistic update
    mpBoardState[tileIndex] = card;
    mpMyDeck.splice(mpSelectedCard, 1);
    mpSelectedCard = null;
    mpMyTurn = false;

    updateTurnIndicator();
    renderBoard();
}

function selectCard(index) {
    mpSelectedCard = mpSelectedCard === index ? null : index;
    renderHand();
}

function sendChatMessage() {
    const input = document.getElementById('mp-chat-input');
    if (!input) return;
    const text = input.value.trim();
    if (!text) return;

    const socket = getSocket();
    if (!socket || socket.readyState !== WebSocket.OPEN) return;

    socket.send(JSON.stringify({
        type: 'mp_chat',
        payload: {
            match_id: mpMatchId,
            text: text
        }
    }));

    mpChatMessages.push({
        sender: 'You',
        text: text,
        isMe: true
    });

    input.value = '';
    renderChat();
}

function forfeitMatch() {
    if (!confirm('Are you sure you want to forfeit?')) return;

    const socket = getSocket();
    if (!socket || socket.readyState !== WebSocket.OPEN) return;

    socket.send(JSON.stringify({
        type: 'mp_forfeit',
        payload: { match_id: mpMatchId }
    }));

    if (window.showToast) window.showToast('🏳️ You forfeited the match', 'info');
    resetMatch();
}

// --- State Updates ---
function updateConnectionStatus() {
    const statusEl = document.getElementById('mp-connection-status');
    if (!statusEl) return;

    const dot = statusEl.querySelector('.mp-status-dot');
    const text = statusEl.querySelector('.mp-status-text');

    if (mpConnected) {
        dot.className = 'mp-status-dot online';
        text.textContent = 'Online';
    } else {
        dot.className = 'mp-status-dot offline';
        text.textContent = 'Offline';
    }
}

function updateMatchmakingStatus(text) {
    const el = document.getElementById('mp-matchmaking-status');
    if (el) el.textContent = text;
}

function updateTurnIndicator() {
    const el = document.getElementById('mp-turn-indicator');
    if (el) {
        el.textContent = mpMyTurn ? 'Your Turn' : "Opponent's Turn";
        el.className = `mp-turn-indicator ${mpMyTurn ? 'my-turn' : 'opponent-turn'}`;
    }
}

function updateLatency() {
    const el = document.getElementById('mp-latency-value');
    if (el) el.textContent = `${mpLatency} ms`;
}

// --- Render Helpers ---
function renderBoard() {
    const board = document.getElementById('mp-board');
    if (board) board.innerHTML = renderBoardTiles();
}

function renderHand() {
    const hand = document.getElementById('mp-hand');
    if (hand) hand.innerHTML = renderPlayerHand();
}

function renderChat() {
    const chat = document.getElementById('mp-chat-messages');
    if (chat) chat.innerHTML = renderChatMessages();
}

// --- WebSocket Message Handlers ---
export function handleMultiplayerMessage(msg) {
    switch (msg.type) {
        case 'match_found':
            onMatchFound(msg.payload);
            break;
        case 'mp_move':
            onOpponentMove(msg.payload);
            break;
        case 'mp_state_sync':
            onStateSync(msg.payload);
            break;
        case 'mp_chat':
            onChatMessage(msg.payload);
            break;
        case 'mp_forfeit':
            onOpponentForfeit(msg.payload);
            break;
        case 'mp_opponent_disconnected':
            onOpponentDisconnected(msg.payload);
            break;
        case 'mp_opponent_reconnected':
            onOpponentReconnected(msg.payload);
            break;
        case 'mp_game_over':
            onGameOver(msg.payload);
            break;
        case 'pong':
            onPong(msg.payload);
            break;
    }
}

function onMatchFound(payload) {
    mpMatchId = payload.match_id;
    mpOpponentId = payload.opponent_id;
    mpOpponentName = payload.opponent_name || 'Opponent';
    mpOpponentAvatar = payload.opponent_avatar || '🤖';
    mpIsHost = payload.is_host || false;
    mpMyTurn = payload.first_turn || false;
    mpGameState = 'active';
    mpBoardState = Array(9).fill(null);
    mpMyDeck = payload.deck || [];
    mpSelectedCard = null;
    mpChatMessages = [];

    if (window.showToast) window.showToast(`⚔️ Match found vs ${mpOpponentName}!`, 'success');

    const el = document.getElementById('wd-game');
    if (el) renderMultiplayerLobby(el);
}

function onOpponentMove(payload) {
    const { card, tile_index } = payload;
    mpBoardState[tile_index] = card;
    mpMyTurn = true;
    updateTurnIndicator();
    renderBoard();

    if (window.showToast) window.showToast(`🎴 ${mpOpponentName} played a card`, 'info');
}

function onStateSync(payload) {
    mpBoardState = payload.board || Array(9).fill(null);
    mpMyTurn = payload.your_turn || false;
    renderBoard();
    updateTurnIndicator();
}

function onChatMessage(payload) {
    mpChatMessages.push({
        sender: payload.sender || mpOpponentName,
        text: payload.text,
        isMe: false
    });
    renderChat();
}

function onOpponentForfeit(payload) {
    if (window.showToast) window.showToast(`🏳️ ${mpOpponentName} forfeited! You win!`, 'success');
    resetMatch();
}

function onOpponentDisconnected(payload) {
    mpOpponentConnected = false;
    if (window.showToast) window.showToast(`⚠️ ${mpOpponentName} disconnected`, 'warning');
}

function onOpponentReconnected(payload) {
    mpOpponentConnected = true;
    if (window.showToast) window.showToast(`✅ ${mpOpponentName} reconnected`, 'success');
}

function onGameOver(payload) {
    mpGameState = 'finished';
    const won = payload.winner === mpOpponentId;
    if (window.showToast) {
        window.showToast(won ? '🎉 You won!' : '😤 You lost!', won ? 'success' : 'error');
    }
    resetMatch();
}

function onPong(payload) {
    if (payload.timestamp) {
        mpLatency = Date.now() - payload.timestamp;
        updateLatency();
    }
}

function resetMatch() {
    mpMatchId = null;
    mpOpponentId = null;
    mpOpponentName = 'Opponent';
    mpOpponentAvatar = '🤖';
    mpIsHost = false;
    mpMyTurn = false;
    mpGameState = 'waiting';
    mpBoardState = Array(9).fill(null);
    mpMyDeck = [];
    mpSelectedCard = null;
    mpChatMessages = [];

    const el = document.getElementById('wd-game');
    if (el) renderMultiplayerLobby(el);
}

// --- Ping/Pong for Latency ---
export function startPingInterval() {
    setInterval(() => {
        const socket = getSocket();
        if (!socket || socket.readyState !== WebSocket.OPEN) return;

        mpLastPing = Date.now();
        socket.send(JSON.stringify({
            type: 'ping',
            payload: { timestamp: mpLastPing }
        }));
    }, 5000);
}

// --- Globals ---
window.initGameMultiplayer = initGameMultiplayer;
window.handleMultiplayerMessage = handleMultiplayerMessage;
window.startPingInterval = startPingInterval;
