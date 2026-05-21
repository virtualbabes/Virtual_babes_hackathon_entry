import { CONFIG } from './config.js';
import { socket } from './network.js';
import { userAddress } from './wallet.js'; // userAddress is now in wallet.js
import { showToast } from './ui.js';
import { availableNetworks } from './admin.js';

export let assetCache = {}; // Asset ID -> Symbol
export let envoiCache = {}; // Wallet Address -> Envoi Name

/**
 * Returns a cached asset symbol or a generic fallback.
 */
export function getAssetSymbol(id) {
    if (!id) return "Token";
    const idStr = id.toString();
    if (idStr === CONFIG.VBV_ASSET_ID?.toString()) return "$VBV";
    if (idStr === CONFIG.AVOI_ASSET_ID?.toString()) return "$AVoi";
    return assetCache[idStr] || "Token";
}

/**
 * Asynchronously resolves an asset symbol from the backend.
 */
export async function resolveAssetSymbol(id) {
    const idStr = id.toString();
    if (assetCache[idStr]) return assetCache[idStr];
    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/asset-symbol?id=${idStr}`);
        const data = await response.json();
        if (data.symbol) {
            assetCache[idStr] = data.symbol;
            return data.symbol;
        }
    } catch (err) { console.warn(`[UTILS] Symbol resolution failed for ${idStr}`); }
    return "Token";
}

/**
 * shortenAddress provides a standardized truncation for blockchain identifiers.
 */
export function shortenAddress(address) {
    if (!address || address.length < 12) return address;
    return address.substring(0, 6) + "..." + address.substring(address.length - 4);
}

/**
 * Returns a cached Envoi name or a truncated address.
 */
export function getCachedEnvoiName(address) {
    if (!address || address === "TBD" || address === "BYE") return address;
    if (address === "DRAW") return "DRAW";
    if (userAddress && address.toLowerCase() === userAddress.toLowerCase()) return "You";
    return envoiCache[address.toLowerCase()] || shortenAddress(address);
}

/**
 * Asynchronously resolves a .voi or .algo name for a wallet address.
 */
export async function resolveEnvoiName(address) {
    if (!address || address.length < 50 || envoiCache[address.toLowerCase()]) return;

    // PILLAR 5: Memory Leak Protection.
    // If the cache exceeds 500 entries (high-traffic session), prune it 
    // to keep the frontend footprint lean.
    if (Object.keys(envoiCache).length > 500) {
        console.log("[UTILS] Pruning Envoi name cache...");
        envoiCache = {};
    }

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/envoi-name?address=${address}`);
        const data = await response.json();
        if (data.name) envoiCache[address.toLowerCase()] = data.name;
    } catch (err) { console.warn(`[UTILS] Envoi resolution failed for ${address}`); }
}

/**
 * Helper to retrieve network-specific configuration data.
 */
export function getNetworkConfig(name) {
    return availableNetworks[name];
}

/**
 * reportGloat allows players to flag offensive taunt messages for administrative review.
 * PILLAR 3: Moderation Uplink.
 */
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
