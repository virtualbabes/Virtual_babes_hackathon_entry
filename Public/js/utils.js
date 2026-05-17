import { CONFIG } from './config.js';
import { socket } from './network.js';
import { userAddress } from './wallet.js'; // userAddress is now in wallet.js
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
 * Returns a cached Envoi name or a truncated address.
 */
export function getCachedEnvoiName(address) {
    if (!address || address === "TBD" || address === "BYE") return address;
    if (address === "DRAW") return "DRAW";
    if (userAddress && address.toLowerCase() === userAddress.toLowerCase()) return "You";
    return envoiCache[address.toLowerCase()] || (address.substring(0, 6) + "..." + address.substring(address.length - 4));
}

/**
 * Asynchronously resolves a .voi or .algo name for a wallet address.
 */
export async function resolveEnvoiName(address) {
    if (!address || address.length < 50 || envoiCache[address.toLowerCase()]) return;
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
