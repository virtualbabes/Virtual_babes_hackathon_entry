import { CONFIG } from './config.js';
import { socket } from './network.js';
import { showToast, renderCardHTML } from './ui.js';
import { userAddress, linkedWallets } from './wallet.js';
import { getNetworkConfig } from './utils.js';
import { calculateDeckRating } from './game.js';

// --- Deck Manager State ---
export let userNFTs = [];
export let currentAvatarUrl = "";
export let cropState = { x: 0, y: 0, zoom: 1 };
export let isCropInitialized = false;

export const setUserNFTs = (nfts) => { userNFTs = nfts; };
export const setCurrentAvatarUrl = (url) => { currentAvatarUrl = url; };
export const setCropState = (state) => { cropState = state; };
export const setIsCropInitialized = (initialized) => { isCropInitialized = initialized; };

export function openDeckManager() {
    document.getElementById("deck-manager-overlay").classList.remove("hidden");
    // PILLAR 5: Explicit Scope Sync. Fetch full inventory since it's pruned from 'all'.
    window.syncUI("inventory");
}

export function closeDeckManager() {
    document.getElementById("deck-manager-overlay").classList.add("hidden");

    // TACTICAL SYNC: Report the highest possible deck rating to the Hall of Fame
    const rating = calculateDeckRating(window.GetGameState().deck);
    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({
            type: "update_rating",
            payload: { best_rating: rating }
        }));
    }
    window.syncUI("all");
}

/**
 * renderDeckManager populates the card pool and active deck UI.
 * PILLAR 5: Scoped Data Handling. Now accepts state to prevent redundant WASM calls.
 */
export function renderDeckManager(state) {
    // Fallback: fetch inventory scope if state not provided or lacks inventory pool
    if (!state || !state.inventory) state = window.GetGameState("inventory");

    const invGrid = document.getElementById("inventory-grid");
    const deckZone = document.getElementById("deck-drop-zone");
    const selector = document.getElementById("deck-selector-bar");
    const atkEl = document.getElementById("total-atk");
    const defEl = document.getElementById("total-def");

    if (!invGrid || !deckZone || !selector) return;

    invGrid.innerHTML = "";
    deckZone.innerHTML = "";
    selector.innerHTML = "";

    let totalAtk = 0;
    let totalDef = 0;

    // 1. Render Inventory
    state.inventory.forEach(card => {
        const cardEl = document.createElement("div");
        // PILLAR 5: Visual Authority. 
        // The 'selected-item' class is now handled internally by renderCardHTML.
        cardEl.className = "card-mini";
        cardEl.draggable = true;
        cardEl.innerHTML = renderCardHTML(card);
        cardEl.ondragstart = (e) => e.dataTransfer.setData("cardID", card.id);
        
        cardEl.onclick = () => {
            window.selectCard(card.id);
        };

        invGrid.appendChild(cardEl);
    });

    // 2. Render Active Deck
    state.deck.forEach((card, idx) => {
        const cardEl = document.createElement("div");
        cardEl.className = "card-mini";
        cardEl.style.width = "100%";
        cardEl.style.height = "60px";
        cardEl.innerHTML = `<span style="font-size: 10px;">${card.name}</span><button onclick="window.RemoveFromDeck(${idx}); window.syncUI('inventory');" style="float: right; padding: 2px 5px; font-size: 9px;">X</button>`;
        
        // Calculate Stats: Attack (Top + Right), Defense (Bottom + Left)
        totalAtk += (card.power[0] + card.power[1]);
        totalDef += (card.power[2] + card.power[3]);
        
        deckZone.appendChild(cardEl);
    });

    if (atkEl) atkEl.innerText = totalAtk;
    if (defEl) defEl.innerText = totalDef;

    // 3. Render Deck Selectors (Unlocks)
    const thresholds = [0, 250, 600, 1000];
    for(let i=0; i<4; i++) {
        const btn = document.createElement("button");
        const isLocked = state.reputation < thresholds[i];
        btn.className = `deck-slot-btn ${i === state.active_deck ? 'active' : ''} ${isLocked ? 'locked' : ''}`;
        btn.innerText = isLocked ? `🔒 ${thresholds[i]} REP` : `Deck ${i+1}`;
        // PILLAR 5: Scoped Update. Refresh view after selection.
        btn.onclick = () => { if(!isLocked) { window.SelectDeck(i); window.syncUI("inventory"); } };
        selector.appendChild(btn);
    }
}

export function renderAvatarGrid(nfts) {
    const grid = document.getElementById("avatar-grid");
    if (!grid) return;
    grid.innerHTML = "";
    
    const state = window.GetGameState();

    nfts.forEach(nft => {
        // PILLAR 3: Multi-Standard Image Resolution.
        // Handle both ServerCard objects (nft.image) and raw indexer metadata strings.
        let url = nft.image || "";
        if (!url && nft.metadata) {
            try { url = JSON.parse(nft.metadata).image || ""; } catch(e) {}
        }
        if (!url) return;
        
        // Filter out banned avatars
        const isBanned = state.banned_avatars && state.banned_avatars[url];
        if (isBanned) return;

        const item = document.createElement("div");
        item.className = "avatar-item";
        item.style.backgroundImage = `url(${url})`;
        item.onclick = () => selectAvatar(url);
        grid.appendChild(item);
    });
}

export function applyAvatarFilters() {
    const search = document.getElementById("avatar-search").value.toLowerCase();
    const sort = document.getElementById("avatar-sort").value;
    
    let filtered = userNFTs.filter(nft => {
        // Support filtering for both ServerCard objects and raw metadata
        let name = nft.name || "";
        if (!name && nft.metadata) {
            try { name = JSON.parse(nft.metadata).name || ""; } catch(e) {}
        }
        return name.toLowerCase().includes(search);
    });
    
    if (sort === "oldest") {
        filtered.sort((a, b) => a.mintRound - b.mintRound);
    } else if (sort === "newest") {
        filtered.sort((a, b) => b.mintRound - a.mintRound);
    }
    
    renderAvatarGrid(filtered);
}

export function selectAvatar(url) {
    const preview = document.getElementById("avatar-preview-section");
    const img = document.getElementById("crop-image");
    if (!preview || !img) return;
    
    currentAvatarUrl = url;
    img.src = url;
    
    // Pre-populate gloat from cache
    const cachedGloat = localStorage.getItem("vbabes_gloat_msg") || "";
    const gloatInput = document.getElementById("gloat-message-input");
    if (gloatInput) gloatInput.value = cachedGloat;

    preview.classList.remove("hidden");
}

export function setupCropEvents() {
    const frame = document.getElementById("crop-frame");
    const img = document.getElementById("crop-image");
    const slider = document.getElementById("zoom-slider");
    const zoomVal = document.getElementById("zoom-val");
    const confirmBtn = document.getElementById("confirm-avatar-btn");
    
    if (!frame || !img || !slider || !confirmBtn) return;
    if (isCropInitialized) return;
    isCropInitialized = true;

    let isDragging = false;
    let startX, startY;

    const updateTransform = () => {
        img.style.transform = `translate(${cropState.x}px, ${cropState.y}px) scale(${cropState.zoom})`;
    };

    img.onload = () => {
        const frameSize = 220;
        const w = img.naturalWidth;
        const h = img.naturalHeight;

        const scaleW = frameSize / w;
        const scaleH = frameSize / h;
        const baseScale = Math.max(scaleW, scaleH);

        cropState.zoom = baseScale;
        cropState.x = (frameSize - (w * baseScale)) / 2;
        cropState.y = (frameSize - (h * baseScale)) / 2;

        slider.min = baseScale.toFixed(2);
        slider.max = (baseScale * 4).toFixed(2);
        slider.value = baseScale;
        if (zoomVal) zoomVal.innerText = "1.0x";
        
        updateTransform();
    };

    slider.oninput = () => {
        cropState.zoom = parseFloat(slider.value);
        const relZoom = cropState.zoom / parseFloat(slider.min);
        if (zoomVal) zoomVal.innerText = relZoom.toFixed(1) + "x";
        updateTransform();
    };

    frame.onmousedown = (e) => {
        if (e.button !== 0) return;
        isDragging = true;
        startX = e.clientX - cropState.x;
        startY = e.clientY - cropState.y;
        frame.style.cursor = "grabbing";
    };

    window.addEventListener('mousemove', (e) => {
        if (!isDragging) return;
        cropState.x = e.clientX - startX;
        cropState.y = e.clientY - startY;
        updateTransform();
    });

    window.addEventListener('mouseup', () => {
        isDragging = false;
        if (frame) frame.style.cursor = "grab";
    });

    confirmBtn.onclick = () => {
        if (window.SetAvatar && currentAvatarUrl) {
            const gloat = document.getElementById("gloat-message-input").value.trim();
            localStorage.setItem("vbabes_gloat_msg", gloat);

            const state = window.GetGameState();
            window.SetAvatar(currentAvatarUrl, gloat, "", state.favorite_card_id || 0);

            if (socket && socket.readyState === WebSocket.OPEN) {
                socket.send(JSON.stringify({
                    type: "register_avatar",
                    payload: { 
                        url: currentAvatarUrl,
                        gloat: gloat
                    }
                }));
            }
            showToast("Avatar verified. Entering Arena.", "success");
        }
    };
}

/**
 * Synchronizes the player's card inventory by querying indexers for the primary and all linked wallets.
 * PILLAR 3: Multi-Standard Discovery. Aggregates assets from ARC-72, ARC-19, and ARC-69 sources.
 */
export async function refreshInventory() {
    if (!userAddress) return;
    
    console.log("[DECK] Synchronizing blockchain assets...");
    
    const grid = document.getElementById("avatar-grid");
    if (grid) {
        grid.innerHTML = `<div class="grid-span-all opacity-5 py-40 italic animate-shimmer">Scanning blockchain for assets...</div>`;
    }

    try {
        // Query the backend dispatcher to find all owned cards across primary and linked wallets
        const response = await fetch(`${CONFIG.API_BASE}/api/card-details?wallet=${userAddress}`);
        if (!response.ok) throw new Error(await response.text());
        
        const cards = await response.json();
        userNFTs = cards;

        // Update WASM engine for each discovered card
        if (window.ImportARC72Card) {
            cards.forEach(card => {
                window.ImportARC72Card(card.id, "Voi Mainnet"); // Standards resolved by backend MetadataDispatcher
            });
        }

        if (grid) {
            grid.innerHTML = ""; // Clear shimmer before rendering
        }

        renderAvatarGrid(cards);
        console.log(`[DECK] Sync Complete. Discovered ${cards.length} tactical assets.`);
    } catch (err) {
        console.error("[DECK] Sync Failed:", err);
    }
}
