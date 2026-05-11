import { CONFIG } from './config.js';
import { socket } from './network.js';
import { showToast, hideAllOverlays } from './ui.js';
import { userAddress, walletProvider, signClient, linkedWallets } from './wallet.js';
import { getCachedEnvoiName, getNetworkConfig, resolveEnvoiName } from './utils.js';
import { globalClubs } from './admin.js'; // globalClubs is now in admin.js
import { lastLobbyPlayers } from './game.js'; // lastLobbyPlayers is now in game.js
import { syncUI } from '../app.js'; // syncUI is still in app.js

const algosdk = window.algosdk; // Assuming algosdk is globally available

    }
}

export function tradeShares(entityId, action, amount) { // Exported for use in app.js
    if (!socket || socket.readyState !== WebSocket.OPEN) return;

    socket.send(JSON.stringify({
    }));

    document.getElementById("portfolio-view-overlay")?.remove();
}

export async function openBlackMarket() {
    const state = window.GetGameState();
