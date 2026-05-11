import { CONFIG } from './config.js';
import { socket } from './network.js';
import { userAddress } from './wallet.js'; // userAddress is now in wallet.js

export let assetCache = {}; // Asset ID -> Symbol
export let envoiCache = {}; // Wallet Address -> Envoi Name
