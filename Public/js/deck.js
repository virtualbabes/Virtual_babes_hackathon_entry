import { CONFIG } from './config.js';
import { socket } from './network.js';
import { showToast, renderCardHTML } from './ui.js';
import { userAddress, linkedWallets } from './wallet.js';
import { getNetworkConfig } from './utils.js';
import { calculateDeckRating, activeCardId } from './game.js';

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
    renderDeckManager();
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

export function renderDeckManager() {
    const state = window.GetGameState();
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
        // Use live binding for activeCardId from game.js
        const isSelected = activeCardId === card.id;
        cardEl.className = `card-mini ${isSelected ? 'selected-item' : ''}`;
        cardEl.draggable = true;
        cardEl.innerHTML = renderCardHTML(card);
        cardEl.ondragstart = (e) => e.dataTransfer.setData("cardID", card.id);
        
        cardEl.onclick = () => {
            window.selectCard(card.id);
            renderDeckManager();
        };

        invGrid.appendChild(cardEl);
    });

    // 2. Render Active Deck
    state.deck.forEach((card, idx) => {
        const cardEl = document.createElement("div");
        cardEl.className = "card-mini";
        cardEl.style.width = "100%";
        cardEl.style.height = "60px";
        cardEl.innerHTML = `<span style="font-size: 10px;">${card.name}</span><button onclick="window.RemoveFromDeck(${idx}); renderDeckManager();" style="float: right; padding: 2px 5px; font-size: 9px;">X</button>`;
        
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
        btn.onclick = () => { if(!isLocked) { window.SelectDeck(i); renderDeckManager(); } };
        selector.appendChild(btn);
    }
}

export function renderAvatarGrid(nfts) {
    const grid = document.getElementById("avatar-grid");
    if (!grid) return;
    grid.innerHTML = "";
    
    const state = window.GetGameState();

    nfts.forEach(nft => {
        let meta = {};
        try { meta = JSON.parse(nft.metadata || "{}"); } catch(e) {}
        const url = meta.image || "";
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
        let meta = {};
        try { meta = JSON.parse(nft.metadata || "{}"); } catch(e) {}
        return (meta.name || "").toLowerCase().includes(search);
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
