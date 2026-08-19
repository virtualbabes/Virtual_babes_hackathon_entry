import { CONFIG } from './config.js';
import { socket, myClientId } from './network.js';
import { showToast, hideAllOverlays } from './ui.js';
import { userAddress, walletProvider, signClient } from './wallet.js';
import { getCachedEnvoiName, getNetworkConfig, shortenAddress } from './utils.js';
import { globalClubs } from './admin.js';
import { lastLobbyPlayers } from './game.js';

export let rumorTimers = {};
export let activeRumors = {};
export let staffTrainingInterval = null;
export let ghostProtocolInterval = null;
export let allianceInviteInterval = null;
let sabotageIntervals = {};

const algosdk = window.algosdk;

export function updateActiveRumors(rumorsData) {
    if (!rumorsData) return;
    if (rumorsData.id) {
        activeRumors[rumorsData.id] = rumorsData;
    } else {
        activeRumors = rumorsData;
    }
    renderRumorBoard();
}

export function renderRumorBoard() {
    const container = document.getElementById("rumor-board-container");
    if (!container) return;

    Object.values(rumorTimers).forEach(clearInterval);
    rumorTimers = {};

    const rumors = Object.values(activeRumors);
    if (rumors.length === 0) {
        container.innerHTML = `<div class="opacity-3 italic py-10 font-size-0-8em text-center">No active intel circulating.</div>`;
        return;
    }

    container.innerHTML = rumors.map(r => `
        <div class="rumor-item ${r.type === 'positive' ? 'rumor-positive' : 'rumor-negative'} animate-slide-up">
            <span class="rumor-text">${r.type === 'positive' ? '📈' : '📉'} ${getCachedEnvoiName(r.target_wallet)}: ${r.strength.toFixed(2)}x</span>
            <span class="rumor-timer font-mono" id="rumor-timer-${r.id}">--:--</span>
        </div>
    `).join('');
    
    rumors.forEach(r => {
        const updateTick = () => {
            const el = document.getElementById(`rumor-timer-${r.id}`);
            if (!el) {
                clearInterval(rumorTimers[r.id]);
                return;
            }

            const diff = new Date(r.expires_at) - new Date();
            if (diff <= 0) {
                clearInterval(rumorTimers[r.id]);
                delete activeRumors[r.id];
                delete rumorTimers[r.id];
                renderRumorBoard();
                return;
            }

            const mins = Math.floor(diff / 60000);
            const secs = Math.floor((diff % 60000) / 1000);
            el.textContent = `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
        };

        updateTick();
        rumorTimers[r.id] = setInterval(updateTick, 1000);
    });
}

export function openCourthouse() {
    const state = window.GetGameState();
    const wanted = state.wanted_level || 0;
    if (wanted <= 0) return;

    const fine = wanted * 100;
    const overlay = document.createElement("div");
    overlay.id = "courthouse-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="courthouse-panel glass-panel">
            <h2 class="text-error">ARENA COURTHOUSE</h2>
            <p class="infamy-status">The High Council has flagged you for criminal activities.<br>Infamy Status: <b>LEVEL ${wanted}</b></p>
            
            <div class="glass-panel fine-display-box">
                <div class="fine-label">REHABILITATION FINE</div>
                <b class="fine-amount">${fine} $VBV</b>
                <div class="fine-subtext">(100 $VBV per Wanted point)</div>
            </div>

            <p class="rehabilitation-text">Settling your debt to society will clear your Wanted Level and restore your cards to peak combat performance.</p>

            <div class="courthouse-actions mt-20 flex-row justify-center gap-15">
                <button class="outline btn-lurk" onclick="document.getElementById('courthouse-overlay').remove()">LURK IN SHADOWS</button>
                <button id="courthouse-pay-btn" class="danger" onclick="submitCourthouseFine()">PAY FINE & CLEAR NAME</button>
            </div>
        </div>
    `;
    document.body.appendChild(overlay);
}

export async function submitCourthouseFine() {
    const state = window.GetGameState();
    const wanted = state.wanted_level || 0;
    if (wanted <= 0) return;

    const btn = document.getElementById("courthouse-pay-btn");
    const amountMicro = wanted * 100 * 1000000;
    const network = state.network;
    const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;

    btn.disabled = true;
    btn.innerText = "PROCESSING...";

    try {
        showToast(`⚖️ Signing ${wanted * 100} $VBV Fine...`, "info");
        let txid = "";
        let txObj = null;

        if (network === "VOI") {
            const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]); 
            const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
            const amountArg = new Uint8Array(32);
            const amountBI = BigInt(amountMicro);
            for (let i = 0; i < 8; i++) amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);

            txObj = {
                from: userAddress, type: 'appl', appIndex: parseInt(assetId),
                appArgs: [methodSelector, recipientAddr, amountArg],
                // PILLAR 3: Bound Verification for Courthouse Fines
                note: new TextEncoder().encode(`COURTHOUSE_FINE:${userAddress}:${Date.now()}`)
            };
        } else {
            txObj = {
                from: userAddress, to: CONFIG.VAULT_ADDRESS, type: 'axfer',
                assetIndex: parseInt(assetId), amount: amountMicro,
                // PILLAR 3: Bound Verification for Courthouse Fines
                note: new TextEncoder().encode(`COURTHOUSE_FINE:${userAddress}:${Date.now()}`)
            };
        }

        if (walletProvider === 'nautilus') {
            const signed = await window.algo.signTxn([{ txn: algosdk.encodeObj(txObj), signers: [userAddress] }]);
            const { txId } = await window.algo.sendRawTransaction(signed[0]);
            txid = txId;
        } else if (walletProvider === 'kibisis') {
            const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(txObj)));
            const signed = await window.kibisis.signTxns([{ txn: txnB64 }]);
            const { txId } = await window.kibisis.pushTxns(signed);
            txid = txId;
        } else if (walletProvider === 'walletconnect' && signClient) {
            const sessions = signClient.session.getAll();
            const chainId = network === "VOI" ? CONFIG.VOI_CHAIN_ID : CONFIG.ALGO_CHAIN_ID;
            const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(txObj)));
            const response = await signClient.request({
                topic: sessions[0].topic, chainId: chainId,
                request: { method: "algo_signTxn", params: [[{ txn: txnB64, signers: [userAddress] }]] }
            });
            const signedTxnBytes = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
            const netCfg = getNetworkConfig(network);
            const client = new algosdk.Algodv2("", netCfg.node_url, "");
            const { txId } = await client.sendRawTransaction(signedTxnBytes).do();
            txid = txId;
        }

        if (!txid) throw new Error("Transaction cancelled or failed.");

        const response = await fetch(`${CONFIG.API_BASE}/api/courthouse/reset`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ wallet: userAddress, txid: txid, network: network })
        });

        if (response.ok) {
            const result = await response.json();
            showToast(`⚖️ ${result.message}`, "success");
            document.getElementById("courthouse-overlay")?.remove();
            if (window.SyncOpponentWanted) window.SyncOpponentWanted(0, 0);
            window.syncUI();
        } else {
            const err = await response.text();
            showToast(`❌ Courthouse Error: ${err}`, "error");
        }
    } catch (err) {
        showToast(`Fine Payment Failed: ${err.message}`, "error");
    } finally {
        btn.disabled = false;
        btn.innerText = "PAY FINE & CLEAR NAME";
    }
}

export async function initiateBail(cardId, clubId) {
    if (!userAddress) return showToast("Connect wallet first", "error");
    if (!confirm(`Are you sure you want to pay 200 $VBV to release Card #${cardId}?`)) return;

    try {
        const state = window.GetGameState();
        const network = state.network;
        const bailAmountMicro = 200 * 1000000;
        const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;

        showToast(`⚖️ Signing Bail Payment for Card #${cardId}...`, "info");
        
        let txid = "";
        if (network === "VOI") {
            const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]); 
            const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
            const amountArg = new Uint8Array(32);
            const amountBI = BigInt(bailAmountMicro);
            for (let i = 0; i < 8; i++) amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);

            const txObj = {
                from: userAddress, type: 'appl', appIndex: parseInt(assetId),
                appArgs: [methodSelector, recipientAddr, amountArg],
                // PILLAR 3: Bound Verification for Underworld Bail
                note: new TextEncoder().encode(`BAIL_PAYMENT:${cardId}:${userAddress}:${Date.now()}`)
            };
            
            if (walletProvider === 'nautilus') {
                const signed = await window.algo.signTxn([{ txn: algosdk.encodeObj(txObj), signers: [userAddress] }]);
                const { txId } = await window.algo.sendRawTransaction(signed[0]);
                txid = txId;
            } else if (walletProvider === 'walletconnect') {
                const sessions = signClient.session.getAll();
                const response = await signClient.request({
                    topic: sessions[0].topic, chainId: CONFIG.VOI_CHAIN_ID,
                    request: { method: "algo_signTxn", params: [[{ txn: btoa(String.fromCharCode(...algosdk.encodeObj(txObj))), signers: [userAddress] }]] }
                });
                const signedTxnBytes = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
                const netCfg = getNetworkConfig(network);
                const client = new algosdk.Algodv2("", netCfg.node_url, "");
                const { txId: broadcastId } = await client.sendRawTransaction(signedTxnBytes).do();
                txid = broadcastId;
            }
        }

        if (!txid) throw new Error("Transaction verification failed.");

        socket.send(JSON.stringify({
            type: "bail_card",
            payload: { card_id: parseInt(cardId), club_id: clubId, txid: txid, network: network }
        }));

        document.getElementById("portfolio-view-overlay")?.remove();
    } catch (err) {
        showToast(`Bail Request Failed: ${err.message}`, "error");
    }
}

export function openSecuritySentry() {
    const state = window.GetGameState();
    const club = globalClubs[state.employer_id];
    if (!club) return;

    const overlay = document.createElement("div");
    overlay.id = "security-sentry-overlay";
    overlay.className = "overlay";

    const availableTraps = [
        { id: "tripwire", name: "Laser Tripwire", desc: "+10% Heist Failure" },
        { id: "sentry_turret", name: "Sentry Turret", desc: "+25% Heist Failure" },
        { id: "guard_dog", name: "Bio-Guard Dog", desc: "Forces Jail on Failure" }
    ];

    const activeTrapsList = Object.entries(club.active_buffs || {})
        .filter(([key]) => key.startsWith("TRAP_"));

    overlay.innerHTML = `
        <div class="security-sentry-panel glass-panel">
            <h2>🛡️ SECURITY SENTRY: ${club.name.toUpperCase()}</h2>
            <p class="description">Deploy tactical hardware to protect the Treasury from heisters.</p>
            <div class="defense-section">
                <small class="section-label">ACTIVE DEFENSES (${activeTrapsList.length}/3)</small>
                <div class="flex-col gap-5 mt-5">
                    ${activeTrapsList.length === 0 ? '<div class="no-traps">No active traps detected.</div>' : 
                      activeTrapsList.map(([id, type]) => `
                        <div class="active-trap-item player-item">
                            <span>🛰️ ${type.toUpperCase()}</span>
                            <span class="trap-online-status">ONLINE</span>
                        </div>
                      `).join('')}
                </div>
            </div>
            <div class="available-hardware-section">
                <small class="section-label">AVAILABLE HARDWARE</small>
                <div class="flex-col gap-10 mt-5">
                    ${availableTraps.map(trap => {
                        const count = state.inventory[trap.id] || 0;
                        return `
                            <div class="hardware-item glass-panel flex-row justify-between align-center">
                                <div class="info"><b>${trap.name}</b><div class="desc">${trap.desc}</div></div>
                                <div class="flex-row align-center gap-10">
                                    <span class="count">Owned: ${count}</span>
                                    <button class="outline btn-deploy-trap" ${count === 0 || activeTrapsList.length >= 3 ? 'disabled' : ''} onclick="deployTrap('${trap.id}')">DEPLOY</button>
                                </div>
                            </div>`;
                    }).join('')}
                </div>
            </div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('security-sentry-overlay').remove()">CLOSE TERMINAL</button>
        </div>`;
    document.body.appendChild(overlay);
}

export function deployTrap(trapId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    showToast(`🛰️ Deploying ${trapId.replace(/_/g, ' ')}...`, "info");
    socket.send(JSON.stringify({ type: "use_item", payload: { item_id: trapId } }));
    document.getElementById("security-sentry-overlay")?.remove();
}

let bountyBoardTicker = null;

export async function openBountyBoard() {
    const state = window.GetGameState();
    const myWanted = state.wanted_level || 0;
    const isHunter = myWanted <= 2;
    const hasDisruptor = (state.inventory && state.inventory["cloak_disruptor"] > 0);
    const isIntel = state.job_role === "Intel-Agent";
    const hasDecryptor = (state.inventory && state.inventory["deep_scan_decryptor"] > 0);
    
    // PILLAR 3: Cooldown Tracking
    const disruptorCooldownAt = state.disruptor_cooldown_at ? new Date(state.disruptor_cooldown_at).getTime() : 0;
    const disruptorOnCooldown = disruptorCooldownAt > Date.now();

    const lastScan = state.last_deep_scan_at ? new Date(state.last_deep_scan_at).getTime() : 0;
    const scanCooldownAt = lastScan + 300000;
    const scanOnCooldown = scanCooldownAt > Date.now();

    // Clean up previous interval
    if (bountyBoardTicker) clearInterval(bountyBoardTicker);

    const outlaws = lastLobbyPlayers.filter(p => (p.wanted_level || 0) >= 10);
    
    const overlay = document.createElement("div");
    overlay.id = "bounty-board-overlay";
    overlay.className = "overlay";
    
    let targetsHtml = "";
    if (outlaws.length === 0) {
        targetsHtml = `<div class="opacity-5 py-40">No active bounties in sector.</div>`;
    } else {
        const wallets = outlaws.map(p => p.wallet);
        await Promise.all(wallets.map(w => resolveEnvoiName(w)));

        targetsHtml = outlaws.map(p => {
            const isGhost = p.ghost_protocol_expires_at && new Date(p.ghost_protocol_expires_at) > Date.now();
            let name = getCachedEnvoiName(p.wallet);
            let district = p.last_seen_district || "Unknown Sector";

            if (isGhost) {
                name = "Ghost";
                district = "Undetectable Sector";
            }

            const isMe = p.id === myClientId;
            const employer = globalClubs[p.employer_id]?.name || 'Freelancer';

            return `
                <div class="player-item border-gold">
                    <div class="text-left">
                        <b class="text-gold">${name}</b><br>
                        <small>Wanted: ${p.wanted_level} | Mojo: ${p.mojo}<br>Affiliation: ${employer}</small>
                    </div>
                    <div class="text-right">
                        <b class="text-neon-green">${p.wanted_level * 50} $VBV</b>
                        ${isHunter && !isMe && !isGhost ? `<button class="outline btn-small border-gold mt-5" onclick="window.sendChallenge('${p.id}'); hideAllOverlays();">HUNT</button>` : ''}
                        ${isMe ? `<span class="text-error" style="font-size: 10px;">TARGET: YOU</span>` : ''}
                    </div>
                </div>`;

        }).join('');
    }

    overlay.innerHTML = `
        <div class="glass-panel w-500 border-gold" style="text-align: center;">
            <h2 class="text-gold">🎯 BOUNTY BOARD</h2>
            <p class="font-size-0-8em opacity-7 mb-20">Hunters (Wanted ≤ 2) earn 50 $VBV per Wanted point on victory.</p>
            ${isHunter && hasDisruptor ? `
                <div class="flex-row justify-center gap-10 mb-15">
                    <button id="btn-disrupt-signal" class="outline btn-small border-neon-cyan text-neon-cyan ${isOnCooldown ? 'disabled' : ''}" 
                            ${isOnCooldown ? 'disabled' : ''}
                            onclick="initiateCloakDisruption()">
                        ${isOnCooldown ? `RECALIBRATING (${Math.ceil((cooldownAt - Date.now())/1000)}s)` : 'DISRUPT SIGNAL'}
                    </button>
                    <small class="opacity-5 font-size-0-7em">Requires Cloak Disruptor. Reveals Ghosted targets for 5 min.</small>
                </div>
            ` : ''}
            <div class="flex-col gap-10 max-h-400 overflow-y-auto">${targetsHtml}</div>
            <button class="outline w-full mt-20" onclick="clearInterval(bountyBoardTicker); document.getElementById('bounty-board-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);

    if (isOnCooldown) {
        bountyBoardTicker = setInterval(() => {
            const btn = document.getElementById("btn-disrupt-signal");
            const now = Date.now();
            if (btn && cooldownAt > now) {
                btn.innerText = `RECALIBRATING (${Math.ceil((cooldownAt - now)/1000)}s)`;
            } else if (btn) {
                btn.innerText = "DISRUPT SIGNAL";
                btn.classList.remove("disabled");
                btn.disabled = false;
                clearInterval(bountyBoardTicker);
            }
        }, 1000);
    }
}

export async function openRumorMill() {
    const state = window.GetGameState();
    const playerRewards = (state.rewards && state.rewards[CONFIG.VBV_ASSET_ID]) || 0;
    const rumorCost = 500;
    const others = lastLobbyPlayers.filter(p => p.id !== myClientId);

    if (playerRewards < rumorCost) {
        showToast(`❌ Insufficient $VBV. Spreading a rumor costs ${rumorCost} $VBV.`, "error");
        return;
    }

    const overlay = document.createElement("div");
    overlay.id = "rumor-mill-overlay";
    overlay.className = "overlay";

    overlay.innerHTML = `
        <div class="criminality-panel glass-panel w-600">
            <h2>RUMOR MILL</h2>
            <p class="font-size-0-8em opacity-7">Influence market sentiment for <b class="text-neon-green">${rumorCost} $VBV</b>.</p>
            <div class="flex-col gap-10 mt-20 max-h-400 overflow-y-auto">
                ${others.length === 0 ? '<div class="opacity-3 py-40">No other entities detected in range.</div>' : 
                    others.map(p => `
                    <div class="player-item">
                        <div class="text-left"><b>${p.id}</b><br><small>${p.reputation} REP | ${p.social_rank}</small></div>
                        <div class="flex-row gap-5">
                            <button class="outline btn-rumor-positive" onclick="spreadRumor('${p.wallet}', 'positive', 1.1, 60)">+ POSITIVE</button>
                            <button class="outline btn-rumor-negative" onclick="spreadRumor('${p.wallet}', 'negative', 0.9, 60)">- NEGATIVE</button>
                        </div>
                    </div>`).join('')}
            </div>
            <button class="outline w-full mt-20" onclick="document.getElementById('rumor-mill-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
}

export async function spreadRumor(targetWallet, type, strength, durationMinutes) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return showToast("❌ Not connected to server.", "error");
    if (!userAddress) return showToast("❌ Connect wallet first.", "error");

    const rumorCost = 500;
    const state = window.GetGameState();
    const playerRewards = (state.rewards && state.rewards[CONFIG.VBV_ASSET_ID]) || 0;

    if (playerRewards < rumorCost) {
        showToast(`❌ Insufficient $VBV. Spreading a rumor costs ${rumorCost} $VBV.`, "error");
        return;
    }

    if (!confirm(`Spread a ${type} rumor about ${getCachedEnvoiName(targetWallet)} for ${rumorCost} $VBV?`)) return;

    showToast(`📢 Spreading rumor...`, "info");
    socket.send(JSON.stringify({
        type: "spread_rumor",
        payload: { target_wallet: targetWallet, type, strength, duration_minutes: durationMinutes }
    }));
    document.getElementById("rumor-mill-overlay")?.remove();
}

export function openTrophyView() {
    openSocialPanelOverlay('achievements');
}

export async function openSocialPanelOverlay(initialTab = 'alliances') {
    const state = window.GetGameState();

    // PILLAR 1: Cyber-Pulse SFX Trigger.
    // Check for active regional disruption before opening.
    const myClub = globalClubs[state.employer_id];
    if (myClub && myClub.buff_expirations) {
        const hasBlackout = Object.entries(myClub.buff_expirations).some(([key, expiry]) => 
            key.startsWith("DISRUPTION_") && new Date(expiry) > Date.now());
        if (hasBlackout && window.playBlackoutSFX) window.playBlackoutSFX();
    }

    // PILLAR 5: DOM Integrity. 
    // Check for existing overlay to prevent duplicate stacks.
    const existing = document.getElementById("social-hub-overlay");
    if (existing) {
        switchSocialTab(initialTab);
        return;
    }

    const overlay = document.createElement("div");
    overlay.id = "social-hub-overlay";
    overlay.className = "overlay";

    if (bountyBoardTicker) {
        clearInterval(bountyBoardTicker);
        bountyBoardTicker = null;
    }

    overlay.innerHTML = `
        <div class="social-panel glass-panel w-550">
            <div class="social-header">
                <span class="social-title">NEON SOCIAL HUB</span>
                <div class="social-stats">
                    <div class="stat-item"><small>MOJO</small><b>${state.mojo || 0}</b></div>
                    <div class="stat-item"><small>REP</small><b>${state.reputation || 0}</b></div>
                </div>
            </div>
            <div class="flex-row justify-center gap-10 mb-20">
                <button id="social-tab-alliances" class="tab-btn" onclick="switchSocialTab('alliances')">🤝 ALLIANCES</button>
                <button id="social-tab-career" class="tab-btn" onclick="switchSocialTab('career')">💼 CAREER</button>
                <button id="social-tab-justice" class="tab-btn" onclick="switchSocialTab('justice')">⚖️ JUSTICE</button>
                <!-- PILLAR 3: Underworld Contracts Integration -->
                <button id="social-tab-underworld" class="tab-btn" onclick="switchSocialTab('underworld')">💀 UNDERWORLD</button>
                <button id="social-tab-achievements" class="tab-btn" onclick="switchSocialTab('achievements')">🏆 VALOR</button>
            </div>
            <div class="flex-row justify-end mb-10 px-20">
                <small class="text-neon-cyan pointer hover-opacity-1" style="font-size: 0.65em; letter-spacing: 1px;" onclick="window.openSignatureReconciliation()">[FORENSIC BREAKDOWN]</small>
            </div>
            <div id="social-content-hub" class="flex-col gap-15 max-h-400 overflow-y-auto"></div>
            <button class="outline mt-20 w-full" onclick="window.cleanupSocialHub(); document.getElementById('social-hub-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
    switchSocialTab(initialTab);
}

/**
 * renderRegionalSabotageReport generates a situational awareness block for club members
 * notifying them of active disruptions in their sectors.
 * PILLAR 1: Regional Warfare.
 */
function renderRegionalSabotageReport(myClub) {
    if (!myClub || !myClub.buff_expirations) return "";

    const disruptions = Object.entries(myClub.buff_expirations)
        .filter(([key, expiry]) => key.startsWith("DISRUPTION_") && new Date(expiry) > Date.now());

    if (disruptions.length === 0) return "";

    return `
        <div class="sabotage-report glass-panel border-error mb-20 p-15 animate-pulse" style="background: rgba(255, 0, 255, 0.1);">
            <div class="text-error font-bold mb-10" style="letter-spacing: 1px; font-size: 0.8em;">📡 SECTOR ALERT: NETWORK BLACKOUT</div>
            <div class="flex-col gap-10">
                ${disruptions.map(([key, expiry]) => {
                    const district = key.replace("DISRUPTION_", "").replace(/_/g, ' ').toUpperCase();
                    const remaining = Math.max(0, new Date(expiry) - Date.now());
                    const mins = Math.ceil(remaining / 60000);
                    return `
                        <div class="flex-row justify-between align-center font-size-0-85em">
                            <span class="text-white">District: <b class="text-neon-purple">${district}</b></span>
                            <span class="text-warning font-mono">${mins}m remaining</span>
                        </div>`;
                }).join('')}
            </div>
            <div class="mt-10 pt-10 border-top-glass font-size-0-7em italic opacity-7">
                Coalition and Regional defensive coordination is OFFLINE in these sectors.
            </div>
        </div>`;
}

/**
 * renderStaffTrainingStatus generates a notification block for active mutation buffs.
 * PILLAR 6: Specialized Gene-Editing.
 */
function renderStaffTrainingStatus(myClub) {
    if (!myClub || !myClub.buff_expirations) return "";

    const expiry = myClub.buff_expirations["STAFF_TRAINING"];
    if (!expiry || new Date(expiry) <= Date.now()) return "";

    return `
        <div id="staff-training-hud" class="staff-training-status glass-panel m-0 mb-15 p-10 flex-row justify-between align-center" style="background: rgba(0, 242, 254, 0.08); border-left: 3px solid var(--neon-cyan);">
            <div class="text-left">
                <small class="opacity-5 block mb-2 letter-spacing-1">VITALITY LAB UPGRADE</small>
                <b class="text-neon-cyan" style="font-size: 0.9em;">🧬 STAFF TRAINING ACTIVE (+5% SUCCESS)</b>
            </div>
            <div id="staff-training-timer" class="text-right font-mono font-xs text-neon-cyan">
                --:--
            </div>
        </div>`;
}

let allianceInviteInterval = null;

/**
 * startAllianceInviteTimer manages the real-time countdown for partnership proposals.
 * PILLAR 1: Alliance Integration.
 */
function startAllianceInviteTimer(expiresAt) {
    if (allianceInviteInterval) clearInterval(allianceInviteInterval);
    const timerEl = document.getElementById("alliance-invite-timer");
    if (!timerEl) return;

    const expiryTime = new Date(expiresAt).getTime();
    const update = () => {
        const now = Date.now();
        const diff = expiryTime - now;
        if (diff <= 0) {
            document.getElementById("alliance-invite-hud")?.remove();
            clearInterval(allianceInviteInterval);
            return;
        }
        const h = Math.floor(diff / 3600000), m = Math.floor((diff % 3600000) / 60000), s = Math.floor((diff % 60000) / 1000);
        timerEl.innerText = h > 0 ? `${h}h ${m}m ${s}s` : `${m}m ${s}s`;
    };
    update();
    allianceInviteInterval = setInterval(update, 1000);
}

/**
 * renderGhostProtocolStatus generates a notification block for active signal scrambling.
 * PILLAR 3: Criminality & Intelligence.
 */
function renderGhostProtocolStatus(state) {
    const expiry = state.ghost_protocol_expires_at;
    if (!expiry || new Date(expiry) <= Date.now()) return "";

    return `
        <div id="ghost-protocol-hud" class="staff-training-status glass-panel m-0 mb-15 p-10 flex-row justify-between align-center" style="background: rgba(155, 81, 224, 0.08); border-left: 3px solid var(--neon-purple);">
            <div class="text-left">
                <small class="opacity-5 block mb-2 letter-spacing-1">GHOST PROTOCOL ACTIVE</small>
                <b class="text-neon-purple" style="font-size: 0.9em;">👻 SIGNAL SCRAMBLED (BOUNTY IMMUNITY)</b>
            </div>
            <div id="ghost-protocol-timer" class="text-right font-mono font-xs text-neon-purple">
                --:--
            </div>
        </div>`;
}

/**
 * startGhostProtocolTimer manages the real-time countdown for signal scrambling.
 */
function startGhostProtocolTimer(expiresAt) {
    if (ghostProtocolInterval) clearInterval(ghostProtocolInterval);
    const timerEl = document.getElementById("ghost-protocol-timer");
    if (!timerEl) return;

    const expiryTime = new Date(expiresAt).getTime();
    const update = () => {
        const now = Date.now();
        const diff = expiryTime - now;
        if (diff <= 0) {
            document.getElementById("ghost-protocol-hud")?.remove();
            clearInterval(ghostProtocolInterval);
            return;
        }
        const h = Math.floor(diff / 3600000), m = Math.floor((diff % 3600000) / 60000), s = Math.floor((diff % 60000) / 1000);
        timerEl.innerText = h > 0 ? `${h}h ${m}m ${s}s` : `${m}m ${s}s`;
    };
    update();
    ghostProtocolInterval = setInterval(update, 1000);
}

/**
 * startStaffTrainingTimer manages the real-time countdown for active mutation buffs.
 * PILLAR 6: Specialized Gene-Editing.
 */
function startStaffTrainingTimer(expiresAt) {
    if (staffTrainingInterval) clearInterval(staffTrainingInterval);
    const timerEl = document.getElementById("staff-training-timer");
    if (!timerEl) return;

    const expiryTime = new Date(expiresAt).getTime();
    const update = () => {
        const now = Date.now();
        const diff = expiryTime - now;
        if (diff <= 0) {
            document.getElementById("staff-training-hud")?.remove();
            clearInterval(staffTrainingInterval);
            return;
        }
        const h = Math.floor(diff / 3600000), m = Math.floor((diff % 3600000) / 60000), s = Math.floor((diff % 60000) / 1000);
        timerEl.innerText = h > 0 ? `${h}h ${m}m ${s}s` : `${m}m ${s}s`;
    };
    update();
    staffTrainingInterval = setInterval(update, 1000);
}

export async function switchSocialTab(tab) {
    const container = document.getElementById("social-content-hub");
    if (!container) return;

    document.querySelectorAll('.tab-btn').forEach(b => b.classList.toggle('active', b.id === `social-tab-${tab}`));
    const state = window.GetGameState();
    container.innerHTML = `<div class="loading-text">Decrypting datastreams...</div>`;

    if (staffTrainingInterval) {
        clearInterval(staffTrainingInterval);
        staffTrainingInterval = null;
    }

    if (ghostProtocolInterval) {
        clearInterval(ghostProtocolInterval);
        ghostProtocolInterval = null;
    }

    if (allianceInviteInterval) {
        clearInterval(allianceInviteInterval);
        allianceInviteInterval = null;
    }

    // PILLAR 1: Clear Sabotage Intervals
    Object.keys(sabotageIntervals).forEach(k => clearInterval(sabotageIntervals[k]));
    sabotageIntervals = {};

    if (tab === 'alliances') {
        const myClubID = state.employer_id;
        const myClub = globalClubs[myClubID];
        const isOwner = myClub && myClub.owner_wallet && userAddress && myClub.owner_wallet.toLowerCase() === userAddress.toLowerCase();

        const sabotageReport = renderRegionalSabotageReport(myClub);
        const ghostHtml = renderGhostProtocolStatus(state);
        const staffTrainingHtml = renderStaffTrainingStatus(myClub);

        // PILLAR 6: Mutation Performance Metrics for Owners.
        let performanceHtml = "";
        if (myClub && isOwner) {
            const successes = myClub.mutation_successes || 0;
            const failures = myClub.mutation_failures || 0;
            const total = successes + failures;
            const rate = total > 0 ? ((successes / total) * 100).toFixed(1) : "0.0";
            performanceHtml = `
                <div class="facility-stats glass-panel m-0 mb-15 p-10 flex-row justify-between align-center" style="background: rgba(0,242,254,0.05); border-left: 3px solid var(--neon-cyan);">
                    <div class="text-left">
                        <small class="opacity-5 block mb-2 letter-spacing-1">FACILITY PERFORMANCE</small>
                        <b class="text-neon-cyan" style="font-size: 1.1em;">${rate}% MUTATION SUCCESS</b>
                    </div>
                    <div class="text-right font-mono font-xs opacity-7">
                        S: ${successes} / F: ${failures}
                    </div>
                </div>`;
        }

        let allianceHtml = "";

        if (myClub) {
            // PILLAR 1: Alliance Hub UI.
            if (myClub.allied_club_id) {
                const ally = globalClubs[myClub.allied_club_id];
                const combinedTerritories = (myClub.territories ? myClub.territories.length : 0) + (ally && ally.territories ? ally.territories.length : 0);
                const isGovernor = combinedTerritories >= 2;
                allianceHtml = `
                    <div class="alliance-status glass-panel ${isGovernor ? 'border-gold governor-highlight' : 'border-cyan'} mb-20 p-15 accelerated">
                        <div class="${isGovernor ? 'text-gold' : 'text-neon-cyan'} font-bold mb-5" style="letter-spacing: 1px;">🤝 ${isGovernor ? 'REGIONAL GOVERNOR ALLIANCE' : 'ACTIVE ALLIANCE'}</div>
                        <div class="flex-row justify-between align-center">
                            <div class="text-left">
                                <b class="text-neon-purple">${ally ? ally.name.toUpperCase() : 'UNKNOWN COALITION'}</b><br>
                                <small class="opacity-7">Allied Districts: ${combinedTerritories}</small><br>
                                <small class="font-xs italic opacity-5">Assets: ${ally && ally.territories ? ally.territories.join(', ') : 'None'}</small>
                            </div>
                            <button class="outline danger btn-small" onclick="dissolveAlliance('${myClub.id}')">DISSOLVE</button>
                        </div>
                    </div>`;
            } else if (myClub.alliance_invite_id) {
                const requester = globalClubs[myClub.alliance_invite_id];
                allianceHtml = `
                    <div class="alliance-status glass-panel border-warning mb-20 p-15 accelerated">
                        <div class="text-warning font-bold mb-5" style="letter-spacing: 1px;">✉️ ALLIANCE PROPOSAL</div>
                        <div class="flex-row justify-between align-center">
                            <div class="text-left">
                                <b>${requester ? requester.name : 'Unknown Club'}</b><br>
                                <small>Owner: ${getCachedEnvoiName(requester ? requester.owner_wallet : '')}</small>
                            </div>
                            <div class="flex-row gap-5">
                                <button class="outline success btn-small" onclick="acceptAlliance('${myClub.id}')">ACCEPT</button>
                                <button class="outline btn-small" onclick="socket.send(JSON.stringify({type:'alliance_invite', payload:{my_club_id:'${myClub.id}', target_club_id:''}}))">DECLINE</button>
                            </div>
                        </div>
                    </div>`;
            } else if (isOwner) {
                const otherClubs = Object.values(globalClubs).filter(c => c.id !== myClub.id && !c.allied_club_id);
                const isGovernor = (myClub.territories ? myClub.territories.length : 0) >= 2;
                allianceHtml = `
                    <div class="alliance-status glass-panel ${isGovernor ? 'border-gold governor-highlight' : 'border-cyan'} mb-20 p-15 accelerated">
                        <div class="${isGovernor ? 'text-gold' : 'text-neon-cyan'} font-bold mb-5" style="letter-spacing: 1px;">${isGovernor ? '👑 REGIONAL GOVERNOR' : '⚔️ INDEPENDENT STATUS'}</div>
                        <div class="text-left mb-15">
                            <b>${myClub.name.toUpperCase()}</b> is currently operating without external coalitions.
                        </div>
                        <small class="section-label opacity-5">PROPOSE COALITION</small>
                        <div class="flex-col gap-5 mt-10 max-h-150 overflow-y-auto">
                            ${otherClubs.map(c => `
                                <div class="player-item p-10 m-0">
                                    <div class="text-left"><b>${c.name}</b><br><small>Districts: ${c.territories ? c.territories.length : 0}</small></div>
                                    <button class="outline btn-small border-cyan" onclick="sendAllianceInvite('${myClub.id}', '${c.id}')">INVITE</button>
                                </div>`).join('') || '<div class="opacity-3 italic text-center py-10 font-xs">No independent clubs available for alliance.</div>'}
                        </div>
                    </div>`;
            } else {
                // Member view for independent club
                const isGovernor = (myClub.territories ? myClub.territories.length : 0) >= 2;
                allianceHtml = `
                    <div class="alliance-status glass-panel ${isGovernor ? 'border-gold governor-highlight' : 'border-cyan'} mb-20 p-15 accelerated">
                        <div class="${isGovernor ? 'text-gold' : 'text-neon-cyan'} font-bold mb-5" style="letter-spacing: 1px;">${isGovernor ? '🛡️ REGIONAL GOVERNOR' : '🛡️ INDEPENDENT STATUS'}</div>
                        <div class="text-left">
                            <b class="text-neon-purple">${myClub.name.toUpperCase()}</b><br>
                            <small class="opacity-7">Districts: ${myClub.territories ? myClub.territories.length : 0}</small><br>
                            <small class="font-xs italic opacity-5">Your organization is currently independent. Alliance coordination is restricted to the CEO.</small>
                        </div>
                    </div>`;
            }
        }

        const others = lastLobbyPlayers.filter(p => p.id !== myClientId);
        container.innerHTML = sabotageReport + staffTrainingHtml + ghostHtml + performanceHtml + allianceHtml + `
            <small class="section-label opacity-5">PEER NETWORK</small>
            <div class="flex-col gap-5 mt-5">
                ${others.map(p => `
                    <div class="connection-item glass-panel m-0 p-10">
                        <div class="connection-info"><b>${p.id}</b><br><small>${p.social_rank}</small></div>
                        <button class="outline btn-small border-cyan" onclick="window.sendChallenge('${p.id}'); hideAllOverlays();">DUEL</button>
                    </div>`).join('') || '<div class="opacity-3 italic text-center py-10 font-xs">No other entities detected.</div>'}
            </div>`;
    } else if (tab === 'career') {
        const tiers = [
            { name: "Iron", mojo: 0, icon: "🌑" }, { name: "Bronze", mojo: 100, icon: "🥉" },
            { name: "Silver", mojo: 300, icon: "🥈" }, { name: "Gold", mojo: 600, icon: "🥇" },
            { name: "Diamond", mojo: 1000, icon: "💎" }
        ];
        container.innerHTML = `<div class="career-path">${tiers.map(t => {
            const isCurrent = (state.mojo || 0) >= t.mojo;
            return `<div class="career-tier ${isCurrent ? 'current' : 'locked'}">
                <div class="tier-icon">${t.icon}</div>
                <div class="tier-info"><b>${t.name}</b><br><small>Req: ${t.mojo} Mojo</small></div>
            </div>`;
        }).join('')}</div>`;
    } else if (tab === 'achievements') {
        const unlocked = new Set(state.achievements || []);
        const catalog = [
            { id: "FIRST_VICTORY", name: "First Victory" }, { id: "FIRST_HEIST", name: "First Heist" },
            { id: "OUTLAW_SLAYER", name: "Outlaw Slayer" }, { id: "ARENA_LEGEND", name: "Arena Legend" },
            { id: "BOUNTY_HUNTER", name: "Bounty Hunter" },
            { id: "PATRON_OF_THE_ARTS", name: "Patron of the Arts" }
        ];
        container.innerHTML = `<div class="achievements-grid">${catalog.map(t => {
            const isUnlocked = unlocked.has(t.id);
            let progressHtml = "";
            if (t.id === "BOUNTY_HUNTER" && !isUnlocked) {
                const count = state.captured_outlaws ? Object.keys(state.captured_outlaws).length : 0;
                progressHtml = `<div class="badge-progress" style="font-size: 0.7em; opacity: 0.6; margin-top: 4px; color: var(--neon-cyan);">${count}/3 unique captures</div>`;
            } else if (t.id === "PATRON_OF_THE_ARTS" && !isUnlocked) {
                const donated = state.total_donated || 0;
                const percent = Math.min(100, (donated / (5000 * 1000000)) * 100);
                progressHtml = `<div class="badge-progress" style="font-size: 0.7em; opacity: 0.6; margin-top: 4px; color: var(--gold);">${percent.toFixed(0)}% of 5,000 VBV</div>`;
            }
            return `
            <div class="trophy-badge ${isUnlocked ? 'unlocked' : 'locked'}">
                <div class="badge-icon">${isUnlocked ? '🏆' : '🔒'}</div>
                <div class="badge-name">${t.name}</div>
                ${progressHtml}
            </div>`;
        }).join('')}</div>`;
    }
    else if (tab === 'justice') {
        renderJusticeDashboard(container);
    }
    else if (tab === 'underworld') {
        fetchUnderworldContractsAndRender(container);
    }
}

/**
 * openLaunderingTerminal provides an interface for Launderers to reduce Wanted Level.
 * PILLAR 3: Career Path Actions.
 */
export function openLaunderingTerminal() {
    const overlay = document.createElement("div");
    overlay.id = "laundering-terminal-overlay";
    overlay.className = "overlay";

    overlay.innerHTML = `
        <div class="criminality-panel glass-panel w-400 border-neon-purple">
            <h2 class="text-neon-purple">🧼 LAUNDERING TERMINAL</h2>
            <p class="opacity-7 font-size-0-85em">Process hot capital to scrub your digital signature.<br>Fee: <b class="text-neon-green">1,000 $VBV</b> | Reduction: <b class="text-neon-cyan">-3 Wanted Level</b></p>
            <div class="flex-col gap-10 mt-20">
                <button class="w-full bg-neon-purple text-white font-bold" onclick="window.submitLaunderRequest()">EXECUTE SCRUB</button>
                <button class="outline w-full mt-10" onclick="document.getElementById('laundering-terminal-overlay').remove()">ABORT</button>
            </div>
        </div>`;
    document.body.appendChild(overlay);
}
window.openLaunderingTerminal = openLaunderingTerminal;

window.submitLaunderRequest = () => {
    const state = window.GetGameState();
    if (state.virtual_balance < 1000) return showToast("❌ Insufficient rewards for laundering.", "error");
    if (state.job_role !== "Launderer") return showToast("❌ Career Path: Launderer Required.", "error");

    showToast("🧼 Initiating capital scrubbing protocol...", "info");
    socket.send(JSON.stringify({
        type: "launder_capital",
        payload: { fee_micro: 1000 * 1000000 }
    }));
    document.getElementById("laundering-terminal-overlay")?.remove();
};

/**
 * renderJusticeDashboard populates the Justice Tier Bounty Center with real-time sector intel.
 * PILLAR 3: Criminality & Intelligence.
 */
function renderJusticeDashboard(container) {
    const state = window.GetGameState();

    // PILLAR 3: Justice Path Authorization.
    // Identify roles aligned with the Justice faction according to the Expansion Plan trajectories.
    const justiceRoles = ["Intel-Agent", "Bounty Hunter", "Armed-Offender-Squad (AOS)", "Justice Recruiter", "Justice Commissioner", "Judge", "Warden", "Forensic Analyst", "Tax Auditor", "Sector Peacekeeper"];
    const isJusticeAligned = justiceRoles.includes(state.job_role);

    const isMission001Active = state.active_justice_mission_id === "MISSION-001";
    const isMission007Active = state.active_justice_mission_id === "MISSION-007";
    const isMission008Active = state.active_justice_mission_id === "MISSION-008";
    const isMission012Active = state.active_justice_mission_id === "MISSION-012";
    const isMission013Active = state.active_justice_mission_id === "MISSION-013";
    const isMission014Active = state.active_justice_mission_id === "MISSION-014";
    const isMission016Active = state.active_justice_mission_id === "MISSION-016";
    const isMission021Active = state.active_justice_mission_id === "MISSION-021";
    const isMission020Active = state.active_justice_mission_id === "MISSION-020";
    const isMission019Active = state.active_justice_mission_id === "MISSION-019";
    const isMission018Active = state.active_justice_mission_id === "MISSION-018";
    const isMission015Active = state.active_justice_mission_id === "MISSION-015";
    const isMission022Active = state.active_justice_mission_id === "MISSION-022";
    const isMission023Active = state.active_justice_mission_id === "MISSION-023";
    const isMission025Active = state.active_justice_mission_id === "MISSION-025";
    const isMission026Active = state.active_justice_mission_id === "MISSION-026";
    const isMission027Active = state.active_justice_mission_id === "MISSION-027";

    const arenaCenterClub = Object.values(globalClubs).find(c => (c.territories || []).includes("arena_center"));
    const arenaCenterOwner = arenaCenterClub ? arenaCenterClub.owner_wallet?.toLowerCase() : null;

    const licenseExpiry = state.bounty_hunter_license_expires_at ? new Date(state.bounty_hunter_license_expires_at) : null;
    const hasActiveLicense = licenseExpiry && licenseExpiry > Date.now();

    const outlaws = lastLobbyPlayers.filter(p => (p.wanted_level || 0) >= 10);
    
    container.innerHTML = `
        <div class="glass-panel p-20 m-0 border-neon-cyan">
            <div id="license-status-banner" class="glass-panel p-10 m-0 mb-15 flex-row justify-between align-center ${hasActiveLicense ? 'border-neon-cyan' : 'expired'}" style="background: rgba(0, 242, 254, 0.05);">
                <div class="text-left">
                    <small class="opacity-5 block mb-2 letter-spacing-1">ENFORCEMENT LICENSE</small>
                    <b class="${hasActiveLicense ? 'text-neon-green' : 'text-error'} font-size-0-9em">${hasActiveLicense ? 'STATUS: VALID' : 'STATUS: EXPIRED / NONE'}</b>
                </div>
                ${hasActiveLicense ? 
                    `<small class="text-neon-cyan font-mono">${licenseExpiry.toLocaleDateString()}</small>` : 
                    `<button class="outline success btn-small" onclick="window.purchaseBountyHunterLicense()">BUY (50 $VBV)</button>`
                }
            </div>

            <h3 class="text-neon-cyan mb-10">JUSTICE TIER BOUNTY CENTER</h3>
            <p class="opacity-7 font-size-0-85em mb-15">
                Sector-wide tracking active. Monitoring high-infamy signatures for enforcement eligibility.
            </p>
            
            <div class="flex-col gap-10">
                <div class="section-label opacity-5 font-size-0-7em letter-spacing-1">REAL-TIME TRACKING GRID</div>
                <div id="justice-tracking-grid" class="flex-col gap-5 max-h-200 overflow-y-auto p-10 bg-dark-overlay rounded">
                    ${outlaws.length === 0 ? 
                        '<div class="opacity-3 italic text-center py-20 font-size-0-85em">No high-infamy targets detected in sector.</div>' :
                        outlaws.map(p => {
                            const isGhost = p.ghost_protocol_expires_at && new Date(p.ghost_protocol_expires_at) > Date.now();
                            const isHost = Object.keys(p.kidnapped_cards || {}).length > 0;
                            const isTargetFor001 = isMission001Active && (p.wanted_level || 0) >= 15 && !isGhost;
                            const isTargetFor007 = isMission007Active && (p.wanted_level || 0) >= 15 && !isGhost;

                            // MISSION-008 & MISSION-012: Regional Governor's status.
                            const isGovernor = p.employer_id && globalClubs[p.employer_id]?.owner_wallet?.toLowerCase() === p.wallet?.toLowerCase() && 
                                               ((globalClubs[p.employer_id]?.territories?.length || 0) + 
                                                (globalClubs[p.employer_id]?.allied_club_id ? (globalClubs[globalClubs[p.employer_id].allied_club_id]?.territories?.length || 0) : 0)) >= 2;
                            const isTargetFor008 = isMission008Active && isGovernor && !isGhost;
                            const isTargetFor012 = isMission012Active && isGovernor && !isGhost;
                            
                            const uniqueVictims = new Set(Object.values(p.kidnapped_cards || {}));
                            const isTargetFor013 = isMission013Active && (p.wanted_level || 0) >= 25 && uniqueVictims.size >= 2 && !isGhost;
                            const isTargetFor014 = isMission014Active && isGovernor && (p.wanted_level || 0) >= 15 && Object.keys(p.kidnapped_cards || {}).length >= 2 && !isGhost;
                            const isTargetFor015 = isMission015Active && (p.wanted_level || 0) >= 30 && !isGhost && p.match_history?.some(h => h.underworld_contract_id === "CONTRACT-018" && h.is_underworld_contract_success);
                            const isTargetFor016 = isMission016Active && p.wallet && arenaCenterOwner === p.wallet.toLowerCase() && (p.wanted_level || 0) >= 35 && !isGhost;
                            const isTargetFor017 = isMission017Active && (p.wanted_level || 0) >= 40 && !isGhost && p.match_history?.some(h => h.underworld_contract_id === "CONTRACT-019" && h.is_underworld_contract_success);                            
                            const isTargetFor021 = isMission021Active && (p.wanted_level || 0) >= 60 && Object.keys(p.kidnapped_cards || {}).length >= 3 && !isGhost;
                            
                            let isTargetFor020 = false;
                            if (isMission020Active && (p.wanted_level || 0) >= 50 && !isGhost && p.match_history) {
                                const executed018 = p.match_history.some(h => h.underworld_contract_id === "CONTRACT-018" && h.is_underworld_contract_success);
                                const executed019 = p.match_history.some(h => h.underworld_contract_id === "CONTRACT-019" && h.is_underworld_contract_success);
                                if (executed018 && executed019) {
                                    isTargetFor020 = true;
                                }
                            }

                            let isTargetFor026 = false;
                            if (isMission026Active && (p.wanted_level || 0) >= 15 && !isGhost) {
                                const playerClub = Object.values(globalClubs).find(c => c.owner_wallet?.toLowerCase() === p.wallet?.toLowerCase());
                                if (playerClub) {
                                    const hasKidnapperStaff = Object.values(playerClub.staff || {}).some(role => role === "Kidnapper");
                                    if (hasKidnapperStaff) {
                                        isTargetFor026 = true;
                                    }
                                }
                            }

                            let isTargetFor027 = false;
                            if (isMission027Active && state.job_role === "Intel-Agent" && (p.wanted_level || 0) >= 40 && !isGhost && p.match_history) {
                                const heistedArenaCenter = p.match_history.some(h => (h.underworld_contract_id === "CONTRACT-015" || h.underworld_contract_id === "CONTRACT-023") && h.is_underworld_contract_success);
                                if (heistedArenaCenter) {
                                    isTargetFor027 = true;
                                }
                            }

                            let isTargetFor022 = false;
                            if (isMission022Active && (p.wanted_level || 0) >= 70 && !isGhost && p.match_history) {
                                const executed022 = p.match_history.some(h => h.underworld_contract_id === "CONTRACT-022" && h.is_underworld_contract_success);
                                if (executed022) {
                                    isTargetFor022 = true;
                                }
                            }

                            let isTargetFor023 = false;
                            if (isMission023Active && (p.wanted_level || 0) >= 80 && !isGhost && p.match_history) {
                                const executed021 = p.match_history.some(h => h.underworld_contract_id === "CONTRACT-021" && h.is_underworld_contract_success);
                                const executed022 = p.match_history.some(h => h.underworld_contract_id === "CONTRACT-022" && h.is_underworld_contract_success);
                                if (executed021 && executed022) {
                                    isTargetFor023 = true;
                                }
                            }

                            let isTargetFor025 = false;
                            if (isMission025Active && state.job_role === "Tax Auditor" && !isGhost) {
                                const ownedClub = Object.values(globalClubs).find(c => c.owner_wallet?.toLowerCase() === p.wallet?.toLowerCase());
                                if (ownedClub && ownedClub.treasury >= 10000) {
                                    isTargetFor025 = true;
                                }
                            }

                            let isTargetFor018 = false;
                            if (isMission018Active && (p.wanted_level || 0) >= 20 && !isGhost) {
                                const playerClub = Object.values(globalClubs).find(c => c.owner_wallet?.toLowerCase() === p.wallet?.toLowerCase());
                                if (playerClub) {
                                    const hasCriminalStaff = Object.values(playerClub.staff || {}).some(role => role === "Criminal");
                                    if (hasCriminalStaff) {
                                        isTargetFor018 = true;
                                    }
                                }
                            }
                            
                            let isTargetFor019 = false;
                            if (isMission019Active && (p.wanted_level || 0) >= 25 && !isGhost) {
                                const playerClub = Object.values(globalClubs).find(c => c.owner_wallet?.toLowerCase() === p.wallet?.toLowerCase());
                                if (playerClub) {
                                    const hasLaundererStaff = Object.values(playerClub.staff || {}).some(role => role === "Launderer");
                                    if (hasLaundererStaff) {
                                        isTargetFor019 = true;
                                    }
                                }
                            }

                            const isPriorityTarget = isTargetFor001 || isTargetFor007 || isTargetFor008 || isTargetFor012 || isTargetFor013 || isTargetFor014 || isTargetFor015 || isTargetFor016 || isTargetFor017 || isTargetFor018 || isTargetFor019 || isTargetFor020 || isTargetFor021 || isTargetFor022 || isTargetFor023 || isTargetFor025 || isTargetFor026 || isTargetFor027;

                            let targetBadge = '';
                            if (isTargetFor001) targetBadge = '<div class="text-neon-cyan font-bold font-size-0-7em mb-2">PRIORITY TARGET</div>';
                            else if (isTargetFor007) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2">OUTLAW ASSET</div>';
                            else if (isTargetFor008) targetBadge = '<div class="text-gold font-bold font-size-0-7em mb-2">GOVERNOR ASSET</div>';
                            else if (isTargetFor012) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ CORRUPTION SIG</div>';
                            else if (isTargetFor013) targetBadge = '<div class="text-warning font-bold font-size-0-7em mb-2 pulse">⚠️ KINGPIN SIG</div>';
                            else if (isTargetFor014) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ GOVERNOR SIG</div>';
                            else if (isTargetFor015 || isTargetFor016) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ HEGEMONY SIG</div>';
                            else if (isTargetFor017) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ CHAOS SIG</div>';
                            else if (isTargetFor018 || isTargetFor021) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ SYNDICATE SIG</div>';
                            else if (isTargetFor019) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ LAUNDRY SIG</div>';
                            else if (isTargetFor020) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ APEX SIG</div>';
                            else if (isTargetFor022) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ SOVEREIGN SIG</div>';
                            else if (isTargetFor023) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ ULTIMATE SIG</div>';
                            else if (isTargetFor026) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ KIDNAPPER SIG</div>';
                            else if (isTargetFor027) targetBadge = '<div class="text-error font-bold font-size-0-7em mb-2 pulse">⚠️ BREACH SIG</div>';
                            else if (isTargetFor025) targetBadge = '<div class="text-neon-cyan font-bold font-size-0-7em mb-2 pulse">⚠️ AUDIT SIG</div>';

                            return `
                                <div class="flex-row justify-between align-center p-8 border-bottom-glass ${isPriorityTarget ? 'priority-target-row' : ''}">
                                    <div class="flex-col">
                                        <b class="${isGhost ? 'text-neon-purple' : 'text-error'}">${isGhost ? 'SIG: [SCRAMBLED]' : getCachedEnvoiName(p.wallet)}</b>
                                        <small class="opacity-5">Last Seen: ${p.last_seen_district || 'Sector Unknown'}</small><br>
                                        ${!isGhost ? (isJusticeAligned ? 
                                            `<div class="flex-row gap-5 mt-5">
                                                <button class="outline btn-small border-neon-cyan text-neon-cyan btn-flag-signature" onclick="flagPlayerForReview('${p.wallet}')">FLAG</button>
                                                ${state.job_role === 'Tax Auditor' && (p.wanted_level || 0) > 30 ? 
                                                    `<button class="outline btn-small border-error text-error" onclick="window.freezeDividends('${p.wallet}')">FREEZE</button>` : ''}
                                                ${isHost && state.job_role === 'Armed-Offender-Squad' ? 
                                                    `<button class="outline btn-small border-neon-purple text-neon-purple" onclick="window.initiateAOSRaid('${p.wallet}')">RAID HOST</button>` : ''}
                                            </div>` : 
                                            `<button class="outline btn-small border-grey text-grey mt-5 opacity-4" style="cursor: not-allowed; font-size: 8px;" disabled title="Justice Career Path Required">FLAG SIGNATURE</button>`
                                        ) : ''}
                                    </div>
                                    <div class="text-right">
                                        ${targetBadge}
                                        <span class="text-error font-mono font-bold">L-INF: ${p.wanted_level}</span>
                                        ${isGhost ? '<div class="font-size-0-6em text-neon-purple font-bold">GHOST ACTIVE</div>' : ''}
                                    </div>
                                </div>
                            `;
                        }).join('')
                    }
                </div>

                <div class="section-label opacity-5 font-size-0-7em letter-spacing-1 mt-10">ACTIVE JUSTICE MISSIONS</div>
                <div id="justice-missions-list" class="flex-col gap-10">
                    <div class="glass-panel p-15 m-0 border-warning italic opacity-5 text-center font-size-0-8em">
                        Decrypting mission dossier...
                    </div>
                </div>
            </div>
        </div>`;

    fetchJusticeMissionsAndRender();
}

window.purchaseBountyHunterLicense = () => {
    const state = window.GetGameState();
    if (state.virtual_balance < 50) return showToast("❌ Insufficient rewards for license.", "error");
    
    showToast("⚖️ Maintaining enforcement status...", "info");
    socket.send(JSON.stringify({ type: "purchase_bounty_license", payload: {} }));
};

window.initiateAOSRaid = (targetWallet) => {
    const state = window.GetGameState();
    if (state.virtual_balance < 1500) return showToast("❌ Raid Failed: 1,500 $VBV Tactical Fee required.", "error");
    
    if (!confirm(`Deploy Armed Offender Squad to strike ${getCachedEnvoiName(targetWallet)}?\n\nTeammates from your club join automatically. Success chance based on team size.`)) return;

    showToast("🎯 Deploying AOS tactical team...", "warning");
    socket.send(JSON.stringify({ type: "aos_raid", payload: { target_wallet: targetWallet } }));
    window.cleanupSocialHub();
    document.getElementById("social-hub-overlay")?.remove();
};

/**
 * purchaseRaidInsurance allows a Hostage Host to secure their vault.
 * PILLAR 1: VBV Sinks.
 */
window.purchaseRaidInsurance = () => {
    const state = window.GetGameState();
    if (state.job_role !== "Hostage Host") return showToast("❌ Career Path: Hostage Host required.", "error");
    if (state.virtual_balance < 3000) return showToast("❌ Insufficient rewards for premium (3,000 $VBV).", "error");

    if (!confirm("🛡️ PURCHASE RAID INSURANCE?\n\nCost: 3,000 $VBV\nDuration: 24 Hours\nCoverage: Blocks 1 successful AOS Raid strike.\n\nProceed with activation?")) return;

    showToast("🛡️ Activating safehouse insurance protocols...", "info");
    socket.send(JSON.stringify({ type: "purchase_raid_insurance", payload: {} }));
};

/**
 * freezeDividends allows a Tax Auditor to disrupt an outlaw's capital flow.
 * PILLAR 3: Justice path.
 */
window.freezeDividends = (targetWallet) => {
    const state = window.GetGameState();
    if (state.job_role !== "Tax Auditor") return showToast("❌ Career Path: Tax Auditor required.", "error");
    
    const targetName = getCachedEnvoiName(targetWallet);
    if (!confirm(`⚖️ INITIATE DIVIDEND FREEZE?\n\nTarget: ${targetName}\n\nDivert future organizational yield to the Faucet?`)) return;

    showToast(`⚖️ Dispatched Regulatory Alert: Freezing ${targetName}...`, "warning");
    socket.send(JSON.stringify({ type: "freeze_dividends", payload: { target_wallet: targetWallet } }));
};

/**
 * purchaseBountyBond allows a player to escrow a security deposit for high-tier missions.
 * PILLAR 1: Industrial Loop (Capital Commitment).
 */
window.purchaseBountyBond = () => {
    const state = window.GetGameState();
    if (state.virtual_balance < 1000) return showToast("❌ Insufficient rewards for security bond (1,000 $VBV required).", "error");
    if ((state.bounty_hunter_bond_micro || 0) > 0) return showToast("⚠️ Security bond already deposited.", "warning");

    if (!confirm("⚖️ DEPOSIT SECURITY BOND?\n\nCost: 1,000 $VBV (Escrowed)\nBenefit: Unlocks high-tier Justice missions.\n\nDeposit is refundable upon clean retirement (Wanted Level ≤ 2). Proceed?")) return;

    showToast("⚖️ Securing bounty hunter bond...", "info");
    socket.send(JSON.stringify({ type: "purchase_bounty_bond", payload: {} }));
};

/**
 * refundBountyBond allows a clean player to retrieve their security deposit.
 * PILLAR 1: Industrial Loop (Capital Retrieval).
 */
window.refundBountyBond = () => {
    const state = window.GetGameState();
    if ((state.bounty_hunter_bond_micro || 0) === 0) return showToast("❌ No active bond found for retrieval.", "error");

    // PILLAR 3: Tactical Warning. Infamy denies the refund protocol.
    if (state.wanted_level > 2) {
        return showToast("❌ <b>REFUND DENIED:</b> Your signature is currently flagged for infamy. Visit the Courthouse to rehabilitate first.", "error");
    }

    if (!confirm("⚖️ REFUND SECURITY BOND?\n\nAction: Returns 1,000 $VBV to your liquid balance.\nConsequence: High-tier Justice status will be revoked.\n\nProceed with retrieval?")) return;

    showToast("⚖️ Requesting bond retrieval...", "info");
    socket.send(JSON.stringify({ type: "refund_bounty_bond", payload: {} }));
};

/**
 * openSignatureReconciliation provides a forensic breakdown of the player's Reputation Standing.
 * PILLAR 3: Identity & Modular Authority.
 */
window.openSignatureReconciliation = () => {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "sig-reconciliation-overlay";
    overlay.className = "overlay";

    // DERIVE REPUTATION COMPONENTS (Synced with economy_service.go weights)
    const baseStanding = 100;
    const victoryBonus = (state.wins || 0) * 10;
    const wantedPenalty = (state.wanted_level || 0) * -10;
    const jailPenalty = Object.keys(state.jailed_cards || {}).length * -50;
    const mojoBonus = Math.floor((state.mojo || 0) / 5);
    
    const achWeights = {
        "GOVERNOR": 150, "ARENA_LEGEND": 200, "FIRST_VICTORY": 50,
        "ART_COLLECTOR": 100, "REHABILITATED": 75, "HEIST_SABOTEUR": 75,
        "CORPORATE_ESPIONAGE": 100
    };
    let achTotal = 0;
    const achList = (state.achievements || []).map(id => {
        const val = achWeights[id] || 0;
        achTotal += val;
        return val > 0 ? `<div class="flex-row justify-between opacity-7 font-size-0-8em"><span>${id.replace(/_/g,' ')}</span><b class="text-neon-cyan">+${val}</b></div>` : '';
    }).join('');

    const myClub = globalClubs[state.employer_id];
    const isSabotaged = myClub?.buff_expirations?.["SABOTAGE"] && new Date(myClub.buff_expirations["SABOTAGE"]) > Date.now();

    overlay.innerHTML = `
        <div class="economy-panel glass-panel medium animate-modal w-450 border-neon-cyan">
            <div class="market-header">
                <span class="market-title">⚖️ SIGNATURE RECONCILIATION</span>
                <div class="access-level">FORENSIC STANDING ANALYSIS</div>
            </div>
            <div class="p-20">
                <div class="flex-col gap-10 mb-20 border-bottom-glass pb-15">
                    <div class="flex-row justify-between"><span>BASE STANDING</span><b>${baseStanding}</b></div>
                    <div class="flex-row justify-between"><span>VICTORY BONUS (${state.wins} WINS)</span><b class="text-neon-green">+${victoryBonus}</b></div>
                    <div class="flex-row justify-between"><span>MOJO INFLUENCE</span><b class="text-neon-purple">+${mojoBonus}</b></div>
                    ${achList}
                    <div class="flex-row justify-between mt-5"><span>INFAMY PENALTY (L-${state.wanted_level})</span><b class="text-error">${wantedPenalty}</b></div>
                    <div class="flex-row justify-between"><span>JAIL CUSTODY PENALTY</span><b class="text-error">${jailPenalty}</b></div>
                </div>

                <div class="flex-row justify-between align-center mb-15 p-10 bg-dark-overlay rounded">
                    <span class="font-bold letter-spacing-1">TOTAL REPUTATION</span>
                    <b class="text-neon-cyan font-size-1-5em" style="text-shadow: 0 0 10px var(--neon-cyan);">${state.reputation || 0}</b>
                </div>

                <div class="flex-col gap-5 mb-20">
                    <div class="flex-row justify-between font-size-0-85em">
                        <span>REGIONAL MULTIPLIER</span>
                        <b class="${state.job_role === 'Governor' ? 'text-gold' : 'opacity-5'}">${state.job_role === 'Governor' ? '1.25x' : '1.00x'}</b>
                    </div>
                    <div class="flex-row justify-between font-size-0-85em">
                        <span>SABOTAGE DISRUPTION</span>
                        <b class="${isSabotaged ? 'text-error' : 'opacity-5'}">${isSabotaged ? '0.80x' : '1.00x'}</b>
                    </div>
                </div>

                ${isSabotaged ? `<div class="text-error font-size-0-8em italic text-center mb-15">⚠️ WARNING: Sabotage is suppressing active multipliers.</div>` : ''}
                <button class="outline w-full" onclick="document.getElementById('sig-reconciliation-overlay').remove()">CLOSE ANALYSIS</button>
            </div>
        </div>`;
    document.body.appendChild(overlay);
};
window.openSignatureReconciliation = window.openSignatureReconciliation;

/**
 * fetchJusticeMissionsAndRender retrieves available objectives for law-enforcement players.
 * PILLAR 3: Justice Path.
 */
export async function fetchJusticeMissionsAndRender() {
    const container = document.getElementById("justice-missions-list");
    if (!container) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/justice/missions`);
        if (!response.ok) throw new Error(await response.text());
        const missions = await response.json();

        const state = window.GetGameState();
        const activeMissionId = state.active_justice_mission_id;

        if (missions.length === 0) {
            container.innerHTML = `<div class="opacity-3 py-20 italic text-center font-size-0-85em">No priority enforcement missions active.</div>`;
            return;
        }

        container.innerHTML = missions.map(m => {
            const isActive = activeMissionId === m.id;
            const isCorruptionTarget = m.id === "MISSION-012";
            const isAuditTarget = m.id === "MISSION-025";
            const isKidnapperTarget = m.id === "MISSION-026";
            const isBreachTarget = m.id === "MISSION-027";
            const isKingpinTarget = m.id === "MISSION-013";
            const isHighStakesAudit = m.id === "MISSION-014";
            const isHegemonyTarget = m.id === "MISSION-015" || m.id === "MISSION-016";
            const isChaosTarget = m.id === "MISSION-017";
            const isWorkforceTarget = m.id === "MISSION-018";
            const isFinancialTarget = m.id === "MISSION-019";
            const isApexTarget = m.id === "MISSION-020" || m.id === "MISSION-024";
            const isSyndicateTarget = m.id === "MISSION-021";
            const isSovereignTarget = m.id === "MISSION-022";
            const isUltimateTarget = m.id === "MISSION-023";
            const badgeHtml = isCorruptionTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ CORRUPTION TARGET</div>` : 
                             isKingpinTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ KINGPIN TARGET</div>` :
                             isHighStakesAudit ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ HIGH-STAKES AUDIT</div>` :
                             isHegemonyTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ HEGEMONY TARGET</div>` : 
                             isChaosTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ CHAOS TARGET</div>` :
                             isWorkforceTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ WORKFORCE TARGET</div>` :
                             isFinancialTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ FINANCIAL TARGET</div>` :
                             isApexTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ APEX TARGET</div>` :
                             isSyndicateTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ SYNDICATE TARGET</div>` :
                             isSovereignTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ SOVEREIGN TARGET</div>` :
                             isUltimateTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ ULTIMATE TARGET</div>` :
                             isAuditTarget ? `<div class="text-neon-cyan font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ AUDIT TARGET</div>` :
                             isKidnapperTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ KIDNAPPER TARGET</div>` :
                             isBreachTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ BREACH TARGET</div>` : "";

            return `
            <div class="glass-panel p-10 m-0 border-neon-cyan flex-col gap-5">
                ${badgeHtml}
                <div class="flex-row justify-between align-center">
                    <b class="text-neon-cyan">${m.title} ${isActive ? '<small class="ml-5 text-neon-cyan pulse">[ACTIVE]</small>' : ''}</b>
                    <span class="text-neon-green font-bold">${(m.reward_micro / 1000000).toFixed(2)} $VBV</span>
                </div>
                <small class="opacity-7 font-size-0-85em">${m.description}</small>
                ${isActive ? 
                    `<div class="flex-row gap-5">
                        ${m.id === "MISSION-005" || m.id === "MISSION-007" || m.id === "MISSION-008" ? `<button class="outline border-neon-cyan text-neon-cyan btn-small mt-5 flex-1" onclick="performForensicAudit('${m.id}')">PERFORM AUDIT</button>` : ''}
                        <button class="outline border-error text-error btn-small mt-5 flex-1" onclick="abortJusticeMission('${m.id}')">ABORT MISSION</button>
                    </div>` :
                    `<button class="outline ${activeMissionId ? 'border-grey text-grey' : 'border-neon-cyan text-neon-cyan'} btn-small mt-5" ${activeMissionId ? 'disabled' : ''} onclick="acceptJusticeMission('${m.id}')">
                        ${activeMissionId ? 'LOCKED' : 'ACCEPT MISSION'}
                    </button>`
                }
            </div>
        `}).join('');
    } catch (err) {
        console.error("[JUSTICE] Mission fetch failed:", err);
        container.innerHTML = `<div class="text-error py-20 text-center font-size-0-85em">Failed to load enforcement dossier.</div>`;
    }
}
window.fetchJusticeMissionsAndRender = fetchJusticeMissionsAndRender;

/**
 * acceptJusticeMission dispatches a request to the backend to initiate a pro-social mission.
 * PILLAR 3: Justice Path.
 */
export function acceptJusticeMission(missionId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        showToast("❌ Connection lost. Re-linking to sector network...", "error");
        return;
    }
    showToast(`⚖️ <b>AUTHORIZING:</b> Mission ${missionId} dossier is being synchronized...`, "info");
    socket.send(JSON.stringify({ type: "accept_justice_mission", payload: { mission_id: missionId } }));
}
window.acceptJusticeMission = acceptJusticeMission;

/**
 * abortJusticeMission requests the backend to terminate the active law-enforcement mission.
 * PILLAR 3: Justice Path.
 */
export function abortJusticeMission(missionId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        showToast("❌ Connection lost. Re-linking to sector network...", "error");
        return;
    }
    if (!confirm("⚠️ ABORT MISSION: This will incur a Reputation penalty. Proceed?")) return;
    
    showToast(`⚖️ <b>ABORTING:</b> Terminating mission ${missionId}...`, "warning");
    socket.send(JSON.stringify({ type: "abort_justice_mission", payload: { mission_id: missionId } }));
}
window.abortJusticeMission = abortJusticeMission;

/**
 * performForensicAudit dispatches a request to the backend to use a Forensic Audit Kit on a target card.
 * PILLAR 3: Justice Path.
 */
export function performForensicAudit(missionId) {
    const cardId = prompt("Enter the ID of the card to audit:");
    if (!cardId) return;
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        showToast("❌ Connection lost. Re-linking to sector network...", "error");
        return;
    }
    showToast(`⚖️ <b>AUDITING:</b> Initiating forensic scan on Card #${cardId}...`, "info");
    socket.send(JSON.stringify({ type: "use_item", payload: { item_id: "forensic_audit_kit", target_card_id: parseInt(cardId) } }));
}

/**
 * flagPlayerForReview triggers the Justice Terminal flagging protocol for a target outlaw.
 * PILLAR 3: Criminality & Intelligence.
 */
export function flagPlayerForReview(targetWallet) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        showToast("❌ Connection lost. Re-linking to sector network...", "error");
        return;
    }
    if (!confirm(`Flag high-infamy outlaw ${getCachedEnvoiName(targetWallet)} for priority administrative review?`)) return;
    showToast(`⚖️ <b>TERMINAL:</b> Flagging signature ${shortenAddress(targetWallet)}...`, "info");
    socket.send(JSON.stringify({ type: "justice_flag_player", payload: { target_wallet: targetWallet } }));
}
window.flagPlayerForReview = flagPlayerForReview;

/**
 * fetchUnderworldContractsAndRender fetches available underworld contracts from the backend
 * and renders them into the social hub.
 * PILLAR 3: Underworld Contracts Integration.
 */
// Dynamic difficulty scaling for underworld contracts — $VBV-gated by player career tier
let currentDifficultyMultiplier = 1.0;

/**
 * computeContractDifficultyRating returns a human-readable difficulty label based on scaled multiplier.
 */
function getDifficultyLabel(multiplier) {
    if (multiplier <= 0.75) return "EASY";
    if (multiplier <= 1.25) return "MODERATE";
    if (multiplier <= 2.0) return "HARD";
    return "ELITE";
}

function getDifficultyColor(multiplier) {
    if (multiplier <= 0.75) return "text-neon-green";
    if (multiplier <= 1.25) return "text-warning";
    if (multiplier <= 2.0) return "text-error";
    return "text-gold pulse";
}

async function fetchUnderworldContractsAndRender(container) {
    container.innerHTML = `
        <div class="glass-panel p-20 m-0 border-error">
            <h3 class="text-error mb-10">UNDERWORLD CONTRACTS</h3>
            <p class="opacity-7 font-size-0-85em mb-15">
                Accept illicit contracts, manage black market operations, and expand criminal influence.
            </p>
            <div id="underworld-contracts-list" class="flex-col gap-10 max-h-300 overflow-y-auto">
                <div class="glass-input w-full text-center opacity-5">Scanning for available contracts...</div>
            </div>
            <button class="w-full outline border-error text-error mt-15">VIEW UNDERWORLD MISSIONS</button>
        </div>`;

    const contractsListEl = document.getElementById("underworld-contracts-list");
    if (!contractsListEl) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/underworld/contracts`);
        if (!response.ok) throw new Error(await response.text());
        const contracts = await response.json();

        const state = window.GetGameState();
        const activeContractId = state.active_underworld_contract_id;

        // Compute dynamic difficulty multiplier from player's $VBV-gated career tier + Cunning level
        const vbvMultiplier = state.vbv_gating_multiplier || 1.0;
        const cunningBonus = Math.min(0.5, (state.cunning || 0) / 200);
        currentDifficultyMultiplier = Math.max(0.3, vbvMultiplier * (1 - cunningBonus));

        if (contracts.length === 0) {
            contractsListEl.innerHTML = `<div class="opacity-3 py-20 italic text-center">No active underworld contracts available.</div>`;
            return;
        }

        contractsListEl.innerHTML = contracts.map(contract => {
            const isActive = activeContractId === contract.id;
            const isGovTarget = contract.id === "CONTRACT-010";
            const isTitanTarget = contract.id === "CONTRACT-011";
            const isStabilizerTarget = contract.id === "CONTRACT-012";
            const isPremierTarget = contract.id === "CONTRACT-013";
            const isSovereignTarget = contract.id === "CONTRACT-014" || contract.id === "CONTRACT-022";
            const isHegemonyTarget = contract.id === "CONTRACT-015" || contract.id === "CONTRACT-016"; // Hegemony covers both heist and sabotage
            const isEliteLiberationTarget = contract.id === "CONTRACT-017";
            const isFortressTarget = contract.id === "CONTRACT-018";
            const isChaosTarget = contract.id === "CONTRACT-019";
            const isApexTarget = contract.id === "CONTRACT-020" || contract.id === "CONTRACT-023";
            const isSyndicateTarget = contract.id === "CONTRACT-021";
            const isFenceTarget = contract.id === "CONTRACT-024";
            const isKidnapperTarget = contract.id === "CONTRACT-025";
            const isSmugglerTarget = contract.id === "CONTRACT-027";
            const badgeHtml = isGovTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1">⚠️ GOVERNOR TARGET</div>` : 
                             isTitanTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1">⚠️ TITAN TARGET</div>` : 
                             isStabilizerTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1">⚠️ STABILIZER TARGET</div>` : 
                             isPremierTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ PREMIER TARGET</div>` : 
                             isSovereignTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ SOVEREIGN TARGET</div>` : 
                             isHegemonyTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ HEGEMONY TARGET</div>` :
                             isEliteLiberationTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ ELITE LIBERATION</div>` :
                             isFortressTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ FORTRESS TARGET</div>` :
                             isChaosTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ CHAOS TARGET</div>` :
                             isApexTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ APEX TARGET</div>` :
                             isSyndicateTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ SYNDICATE TARGET</div>` :
                             isFenceTarget ? `<div class="text-warning font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ FENCING TARGET</div>` :
                             isKidnapperTarget ? `<div class="text-error font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ KIDNAPPER SIG</div>` :
                             isSmugglerTarget ? `<div class="text-warning font-bold font-size-0-7em mb-2 letter-spacing-1 pulse">⚠️ SMUGGLER SIG</div>` : "";

            const baseReward = contract.reward_micro / 1000000;
            const scaledReward = baseReward * currentRewardRatio;

            // Compute difficulty rating for this contract based on player scaling
            const diffLabel = getDifficultyLabel(currentDifficultyMultiplier);
            const diffColor = getDifficultyColor(currentDifficultyMultiplier);

            return `
            <div class="glass-panel p-10 m-0 ${isActive ? 'border-neon-green' : 'border-warning'} flex-col gap-5">
                ${badgeHtml}
                <!-- PILLAR 3: Dynamic Difficulty Indicator -->
                <div class="flex-row justify-between align-center mb-5">
                    <span class="font-size-0-7em letter-spacing-1 ${diffColor}" style="text-shadow: 0 0 6px currentColor;">⚠️ DIFFICULTY: ${diffLabel}</span>
                    ${currentRewardRatio < 1.0 ? `<small class="opacity-5 font-size-0-6em">Scaling ×${(1/currentDifficultyMultiplier).toFixed(2)}</small>` : ''}
                </div>
                <div class="flex-row justify-between align-center">
                    <b class="${isActive ? 'text-neon-green' : 'text-warning'}">${contract.title} ${isActive ? '<small class="ml-5 text-neon-green pulse">[ACTIVE]</small>' : ''}</b>
                    <div class="flex-col align-end">
                        <span class="text-neon-green font-bold">${scaledReward.toFixed(2)} $VBV</span>
                        ${currentRewardRatio < 1.0 ? `<small class="opacity-5 font-size-0-6em">Target: ${baseReward.toFixed(0)}</small>` : ''}
                    </div>
                </div>
                <small class="opacity-7">${contract.description}</small>
                ${isActive ? 
                    `<button class="outline border-error text-error btn-small mt-5" onclick="abortUnderworldContract('${contract.id}')">ABORT CONTRACT</button>` :
                    `<button class="outline ${activeContractId ? 'border-grey text-grey' : 'border-warning text-warning'} btn-small mt-5" ${activeContractId ? 'disabled' : ''} onclick="acceptUnderworldContract('${contract.id}')">
                        ${activeContractId ? 'LOCKED' : 'ACCEPT CONTRACT'}
                    </button>`
                }
            </div>
        `}).join('');

    } catch (err) {
        console.error("[UNDERWORLD] Failed to fetch contracts:", err);
        contractsListEl.innerHTML = `<div class="text-error py-20 text-center">Failed to load contracts.</div>`;
    }
}

/**
 * acceptUnderworldContract dispatches a request to the backend to initiate a criminal mission.
 * PILLAR 3: Underworld Contracts.
 */
export function acceptUnderworldContract(contractId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        showToast("❌ Connection lost. Re-linking to sector network...", "error");
        return;
    }
    showToast(`💀 <b>NEGOTIATING:</b> Contract ${contractId} is being finalized...`, "info");
    socket.send(JSON.stringify({ type: "accept_underworld_contract", payload: { contract_id: contractId } }));
}
window.acceptUnderworldContract = acceptUnderworldContract;

/**
 * abortUnderworldContract requests the backend to terminate the active criminal mission.
 * PILLAR 3: Underworld Contracts.
 */
export function abortUnderworldContract(contractId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        showToast("❌ Connection lost. Re-linking to sector network...", "error");
        return;
    }
    if (!confirm("⚠️ ABORT CONTRACT: This will incur a Reputation penalty. Proceed?")) return;
    
    showToast(`💀 <b>ABORTING:</b> Terminating contract ${contractId}...`, "warning");
    socket.send(JSON.stringify({ type: "abort_underworld_contract", payload: { contract_id: contractId } }));
}
window.abortUnderworldContract = abortUnderworldContract;

export function openHeistPlanningOverlay() {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "heist-overlay";
    overlay.className = "overlay";
    const clubs = Object.values(globalClubs).filter(c => c.id !== state.employer_id);
    
    overlay.innerHTML = `
        <div class="criminality-panel heist-terminal glass-panel animate-modal w-800">
            <div class="criminality-header">
                <span class="criminality-title">HEIST PLANNING TERMINAL</span>
                <div class="criminality-stats">
                    <div class="stat-item"><small>WANTED</small><b class="text-error">${state.wanted_level || 0}</b></div>
                    <div class="stat-item"><small>CUNNING</small><b class="text-neon-cyan">${state.cunning || 0}</b></div>
                </div>
            </div>
            <div class="p-20">
                <div class="criminality-targets mb-20">
                    <div class="targets-list" style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px; max-height: 300px; overflow-y: auto;">
                        ${clubs.map(c => `<div class="target-item glass-panel m-0 p-15" onclick="updateHeistRiskAssessment('${c.id}')">
                            <b class="text-neon-purple">${c.name.toUpperCase()}</b><br>
                            <small class="text-neon-green">${c.treasury.toFixed(2)} $VBV</small>
                        </div>`).join('') || "No external clubs detected."}
                    </div>
                </div>
                <div id="heist-risk-section" class="criminality-risk invisible mt-10 p-20 glass-panel">
                    <div class="risk-bar"><div id="heist-risk-fill" class="risk-fill" style="width: 0%;"></div></div>
                    <div id="heist-chance-text" class="mt-10 font-bold"></div>
                    <div class="flex-row gap-15 mt-20">
                        <button class="outline w-full secondary" onclick="document.getElementById('heist-overlay').remove()">ABORT</button>
                        <button id="heist-execute-btn" class="w-full danger">EXECUTE STRIKE</button>
                    </div>
                </div>
            </div>
        </div>`;
    document.body.appendChild(overlay);
}

export function updateHeistRiskAssessment(clubId) {
    const state = window.GetGameState("combat"); // Use scoped fetch for performance
    const club = globalClubs[clubId];
    const fill = document.getElementById("heist-risk-fill");
    const text = document.getElementById("heist-chance-text");
    const btn = document.getElementById("heist-execute-btn");
    if (!club || !fill) return;

    document.getElementById("heist-risk-section").classList.remove("invisible");
    let securityStaff = 0;
    if (club.staff) Object.values(club.staff).forEach(role => { if(role === "Security") securityStaff++; });
    const securityLevel = (club.club_mojo / 10) + (securityStaff * 15);
    const trapModifiers = { "tripwire": -0.10, "sentry_turret": -0.25, "guard_dog": -0.05 };
    let trapPenalty = 0;
    if (club.active_buffs) Object.values(club.active_buffs).forEach(id => trapPenalty += (trapModifiers[id] || 0));

    const successChance = Math.min(0.95, Math.max(0.05, 0.50 + (state.cunning - securityLevel) / 100 + trapPenalty));
    fill.style.width = `${(1 - successChance) * 100}%`;

    // PILLAR 1: Surcharge Visibility.
    const targetOwner = lastLobbyPlayers.find(p => p.wallet?.toLowerCase() === club.owner_wallet?.toLowerCase());
    const reps = targetOwner?.reparations_received_count || 0;
    const surchargePercent = Math.floor(reps / 5) * 10;

    text.innerHTML = `
        ESTIMATED SUCCESS: <b class="text-neon-green">${(successChance * 100).toFixed(0)}%</b>
        ${surchargePercent > 0 ? `<div class="text-warning mt-5 font-size-0-8em" style="letter-spacing: 1px;">🛡️ HARDENED TARGET: +${surchargePercent}% Sabotage Surcharge</div>` : ''}`;
    
    btn.onclick = () => executeHeistStrike(clubId);
}

/**
 * sendHeistRequest provides a clean functional hook for heist initiation.
 * PILLAR 3: Functional Wiring.
 */
export function sendHeistRequest(targetClubId) {
    showToast("🔪 Operatives deployed...", "warning");
    socket.send(JSON.stringify({ type: "heist", payload: { target_club_id: targetClubId } }));
}
window.sendHeistRequest = sendHeistRequest;

export function executeHeistStrike(clubId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    showToast("🔪 Operatives deployed...", "warning");
    socket.send(JSON.stringify({ type: "heist", payload: { target_club_id: clubId } }));
    document.getElementById("heist-overlay")?.remove();
}

export function handleHeistResult(payload) {
    const isSuccess = payload.status === "success";
    showToast(`<b>HEIST ${isSuccess ? 'SUCCESS' : 'FAILED'}</b><br>${isSuccess ? 'Looted!' : 'Escaped!'}`, isSuccess ? "success" : "error");
    if (isSuccess && payload.kidnap_eligible) setTimeout(() => openKidnapSelectionOverlay(payload.target_club_id), 1500);
}

export function openKidnapSelectionOverlay(targetClubId) {
    const club = globalClubs[targetClubId];
    if (!club) return;
    const overlay = document.createElement("div");
    overlay.id = "kidnap-selection-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="criminality-panel glass-panel w-400">
            <h2 class="text-warning">KIDNAP GAMBIT</h2>
            <p>Cornered high-value asset of <b class="text-neon-purple">${club.name}</b>.</p>
            <button class="w-full danger mt-20" onclick="executeKidnap('${targetClubId}')">EXECUTE KIDNAPPING</button>
            <button class="outline w-full mt-10" onclick="document.getElementById('kidnap-selection-overlay').remove()">RELEASE</button>
        </div>`;
    document.body.appendChild(overlay);
}

export function executeKidnap(targetClubId) {
    socket.send(JSON.stringify({ type: "kidnap_request", payload: { target_club_id: targetClubId } }));
    document.getElementById("kidnap-selection-overlay")?.remove();
}

export function showKidnapOverlay(payload) {
    const overlay = document.getElementById("kidnap-overlay");
    const content = document.getElementById("kidnap-content");
    if (!overlay || !content) return;

    content.innerHTML = `
        <p>Card <strong>#${payload.card_id}</strong> kidnapped!</p>
        <p>Ransom: <b class="text-neon-cyan">${(payload.ransom / 1000000).toFixed(2)} $VBV</b></p>
        <button class="w-full danger" onclick="payRansom(${payload.card_id}, '${payload.perp_wallet}', ${payload.ransom})">PAY RANSOM</button>
    `;
    overlay.classList.remove("hidden");
    startRecoveryTimer(payload.expires_at);
}

export async function payRansom(cardId, perpWallet, ransomAmount) {
    socket.send(JSON.stringify({ type: "pay_ransom", payload: { card_id: cardId, perp_wallet: perpWallet, ransom_amount: ransomAmount } }));
    hideAllOverlays();
}

export function releaseHostage(cardId) {
    if (!confirm(`Release Card #${cardId}?`)) return;
    socket.send(JSON.stringify({ type: "release_hostage", payload: { card_id: cardId } }));
}

export function startRecoveryTimer(expiresAt) {
    const timerEl = document.getElementById("recovery-timer");
    if (!timerEl) return;

    const interval = setInterval(() => {
        const diff = new Date(expiresAt) - new Date();
        if (diff <= 0) { clearInterval(interval); timerEl.innerText = "00:00:00"; return; }
        const h = Math.floor(diff / 3600000);
        const m = Math.floor((diff % 3600000) / 60000);
        const s = Math.floor((diff % 60000) / 1000);
        timerEl.innerText = `${h.toString().padStart(2,'0')}:${m.toString().padStart(2,'0')}:${s.toString().padStart(2,'0')}`;
    }, 1000);
}

// PILLAR 1: Alliance Hub Handlers
export function sendAllianceInvite(myClubId, targetClubId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    showToast("📡 Dispatching alliance proposal...", "info");
    socket.send(JSON.stringify({ type: "alliance_invite", payload: { my_club_id: myClubId, target_club_id: targetClubId } }));
    switchSocialTab('alliances');
}

export function acceptAlliance(myClubId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    showToast("🤝 Finalizing regional alliance...", "success");
    socket.send(JSON.stringify({ type: "alliance_accept", payload: { my_club_id: myClubId } }));
    switchSocialTab('alliances');
}

export function dissolveAlliance(myClubId) {
    if (!confirm("⚠️ Terminate regional alliance? All shared territory bonuses will be lost.")) return;
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    showToast("📡 Dissolving alliance...", "warning");
    socket.send(JSON.stringify({ type: "alliance_dissolve", payload: { my_club_id: myClubId } }));
    switchSocialTab('alliances');
}

// initiateRegionalSabotage allows a player to trigger a regional network blackout.
export function initiateRegionalSabotage(targetClubId, targetTerritoryId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return showToast("❌ Not connected to server.", "error");
    if (!userAddress) return showToast("❌ Connect wallet first.", "error");

    if (!confirm(`⚠️ Initiate Regional Sabotage on ${targetTerritoryId} (Club: ${globalClubs[targetClubId]?.name})? This is a high-cost, high-impact action.`)) return;

    showToast(`📡 Initiating regional blackout in ${targetTerritoryId}...`, "warning");
    socket.send(JSON.stringify({
        type: "regional_sabotage",
        payload: {
            target_club_id: targetClubId,
            target_territory_id: targetTerritoryId
        }
    }));
    hideAllOverlays(); // Close any open overlays after action
}

/**
 * initiateCloakDisruption allows a Bounty Hunter to use a Cloak Disruptor item
 * to temporarily reveal a Ghosted target's location.
 * PILLAR 3: Criminality & Intelligence.
 */
export function initiateCloakDisruption() {
    const state = window.GetGameState();
    const myWanted = state.wanted_level || 0;
    const hasDisruptor = (state.inventory && state.inventory["cloak_disruptor"] > 0);

    // PILLAR 3: Cooldown enforcement
    const cooldownAt = state.disruptor_cooldown_at ? new Date(state.disruptor_cooldown_at).getTime() : 0;
    if (cooldownAt > Date.now()) {
        const remaining = Math.ceil((cooldownAt - Date.now()) / 1000);
        showToast(`❌ Disruptor Recalibrating: Ready in ${remaining}s.`, "error");
        return;
    }

    if (myWanted > 2 || !hasDisruptor) {
        showToast("❌ Access Denied: Requires Bounty Hunter status (Wanted ≤ 2) and a Cloak Disruptor.", "error");
        return;
    }

    const targetWallet = prompt("Enter the wallet address of the Ghosted target to disrupt:");
    if (!targetWallet) return;

    showToast(`📡 Attempting to disrupt signal for ${shortenAddress(targetWallet)}...`, "info");
    socket.send(JSON.stringify({
        type: "use_item",
        payload: { item_id: "cloak_disruptor", target_wallet: targetWallet }
    }));
    document.getElementById('bounty-board-overlay')?.remove();
}

/**
 * reportPlayer triggers the automated reporting protocol for a specific target.
 * PILLAR 3: Criminality & Intelligence.
 */
export async function reportPlayer(targetWallet) {
    const reason = prompt("Enter reason for report (e.g. Harassment, Exploiting, Cheating):");
    if (!reason) return;
    const details = prompt("Enter additional details (Match ID, specific behavior):") || "";

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/report-player`, {
            method: "POST",
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                reporter_wallet: userAddress,
                target_wallet: targetWallet,
                reason: reason,
                details: details
            })
        });

        if (response.ok) {
            showToast("🚩 Report submitted to Arena Security.", "success");
        } else {
            const err = await response.text();
            showToast(`❌ Failed to submit report: ${err}`, "error");
        }
    } catch (err) {
        console.error("[MODERATION ERROR]", err);
        showToast("❌ Connection error during report submission.", "error");
    }
}

/**
 * flagPlayerForReview triggers the Justice Terminal flagging protocol for a target outlaw.
 * PILLAR 3: Criminality & Intelligence.
 */
export function flagPlayerForReview(targetWallet) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        showToast("❌ Connection lost. Re-linking to sector network...", "error");
        return;
    }
    if (!confirm(`Flag high-infamy outlaw ${getCachedEnvoiName(targetWallet)} for priority administrative review?`)) return;
    showToast(`⚖️ <b>TERMINAL:</b> Flagging signature ${shortenAddress(targetWallet)}...`, "info");
    socket.send(JSON.stringify({ type: "justice_flag_player", payload: { target_wallet: targetWallet } }));
}

/**
 * window.cleanupSocialHub purges all active intervals to prevent memory leaks.
 * PILLAR 5: Separation of Concerns.
 */
window.cleanupSocialHub = () => {
    if (staffTrainingInterval) clearInterval(staffTrainingInterval);
    if (ghostProtocolInterval) clearInterval(ghostProtocolInterval);
    if (allianceInviteInterval) clearInterval(allianceInviteInterval);
    Object.keys(sabotageIntervals).forEach(k => clearInterval(sabotageIntervals[k]));
    sabotageIntervals = {};
};

window.sendAllianceInvite = sendAllianceInvite;
window.acceptAlliance = acceptAlliance;
window.dissolveAlliance = dissolveAlliance;
window.openCourthouse = openCourthouse;
window.submitCourthouseFine = submitCourthouseFine;
window.initiateBail = initiateBail;
window.openSecuritySentry = openSecuritySentry;
window.deployTrap = deployTrap;
window.openBountyBoard = openBountyBoard;
window.openRumorMill = openRumorMill;
window.spreadRumor = spreadRumor;
window.openTrophyView = openTrophyView;
window.openSocialPanelOverlay = openSocialPanelOverlay;
window.switchSocialTab = switchSocialTab;
window.openHeistPlanningOverlay = openHeistPlanningOverlay;
window.updateHeistRiskAssessment = updateHeistRiskAssessment;
window.executeHeistStrike = executeHeistStrike;
window.openKidnapSelectionOverlay = openKidnapSelectionOverlay;
window.executeKidnap = executeKidnap;
window.showKidnapOverlay = showKidnapOverlay;
window.payRansom = payRansom;
window.initiateCloakDisruption = initiateCloakDisruption;
window.reportPlayer = reportPlayer;
window.releaseHostage = releaseHostage;
window.flagPlayerForReview = flagPlayerForReview;
window.initiateDeepScan = initiateDeepScan;
// PILLAR 3: Underworld Contract WS event handlers — dispatched by network.js on server push
export function handleContractAssigned(payload) {
    showToast(`💀 <b>CONTRACT ASSIGNED:</b><br>${payload.contract_title || payload.id} — Reward: ${(payload.reward_micro / 1000000).toFixed(2)} $VBV`, "warning", 6000);
    // Refresh the contracts list if social hub is visible
    const container = document.getElementById("underworld-contracts-list");
    if (container) fetchUnderworldContractsAndRender(container.parentElement?.parentElement || container);
}

export function handleContractCompleted(payload) {
    showToast(`✅ <b>CONTRACT COMPLETED:</b><br>${payload.contract_title || payload.id} — Earned ${(payload.reward_micro / 1000000).toFixed(2)} $VBV + ${payload.xp_awarded || 0} CareerXP`, "success", 6000);
    // Refresh the contracts list if social hub is visible
    const container = document.getElementById("underworld-contracts-list");
    if (container) fetchUnderworldContractsAndRender(container.parentElement?.parentElement || container);
}

window.freezeDividends = window.freezeDividends;
