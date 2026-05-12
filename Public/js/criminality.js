import { updateActiveRumors, renderRumorBoard } from './criminality.js';
import { CONFIG } from './config.js';
import { socket, myClientId } from './network.js';
import { showToast, hideAllOverlays } from './ui.js';
import { userAddress, walletProvider, signClient } from './wallet.js'; // Removed payoutAddress as it's not used here
import { getCachedEnvoiName, getNetworkConfig } from './utils.js';
import { globalClubs } from './admin.js'; // globalClubs is now in admin.js
import { lastLobbyPlayers, myPlayerIndex, setCurrentOpponentId, setMyPlayerIndex } from './game.js'; // lastLobbyPlayers is now in game.js
import { syncUI } from '../app.js'; // syncUI is still in app.js

export let rumorTimers = {};
export let activeRumors = [];

const algosdk = window.algosdk; // Assuming algosdk is globally available

export function updateActiveRumors(rumorsData) {
    // Clear existing timers
        timerEl.textContent = `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }, 1000);
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
            const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]); // transfer(address,uint256)
            const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
            const amountArg = new Uint8Array(32);
            const amountBI = BigInt(amountMicro);
            for (let i = 0; i < 8; i++) amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);

            txObj = {
                from: userAddress, type: 'appl', appIndex: parseInt(assetId),
                appArgs: [methodSelector, recipientAddr, amountArg],
                note: new TextEncoder().encode(`ARENA_COURTHOUSE_FINE:${wanted}`)
            };
        } else {
            txObj = {
                from: userAddress, to: CONFIG.VAULT_ADDRESS, type: 'axfer',
                assetIndex: parseInt(assetId), amount: amountMicro,
                note: new TextEncoder().encode(`ARENA_COURTHOUSE_FINE:${wanted}`)
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
            if (window.SyncOpponentWanted) window.SyncOpponentWanted(0, 0); // Assuming local player is always P1 for this context
            window.syncUI(); // Access syncUI via window object
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
        // Construction logic mirroring courthouse fine
        if (network === "VOI") {
            const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]); 
            const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
            const amountArg = new Uint8Array(32);
            const amountBI = BigInt(bailAmountMicro);
            for (let i = 0; i < 8; i++) amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);

            const txObj = {
                from: userAddress, type: 'appl', appIndex: parseInt(assetId),
                appArgs: [methodSelector, recipientAddr, amountArg],
                note: new TextEncoder().encode(`BAIL_PAYMENT:${cardId}`)
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
            payload: {
                card_id: parseInt(cardId),
                club_id: clubId,
                txid: txid,
                network: network
            }
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

    // Heuristic: Traps are items with "tripwire", "sentry", or "dog" in ID
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
                                <div class="info">
                                    <b>${trap.name}</b>
                                    <div class="desc">${trap.desc}</div>
                                </div>
                                <div class="flex-row align-center gap-10">
                                    <span class="count">Owned: ${count}</span>
                                    <button class="outline btn-deploy-trap" 
                                            ${count === 0 || activeTrapsList.length >= 3 ? 'disabled' : ''} 
                                            onclick="deployTrap('${trap.id}')">DEPLOY</button>
                                </div>
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>

            <button class="outline mt-20 w-full" onclick="document.getElementById('security-sentry-overlay').remove()">CLOSE TERMINAL</button>
        </div>
    `;
    document.body.appendChild(overlay);
}

export function deployTrap(trapId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    
    showToast(`🛰️ Deploying ${trapId.replace(/_/g, ' ')}...`, "info");
    socket.send(JSON.stringify({
        type: "use_item",
        payload: {
            item_id: trapId
        }
    }));
    document.getElementById("security-sentry-overlay")?.remove();
}

export async function openBountyBoard() {
    const state = window.GetGameState();
    const myWanted = state.wanted_level || 0;
    const isHunter = myWanted <= 2;
    
    const overlay = document.createElement("div");
    overlay.id = "bounty-board-overlay";
    overlay.className = "overlay";
    
    const outlaws = lastLobbyPlayers.filter(p => (p.wanted_level || 0) >= 10);
    
    let targetsHtml = "";
    if (outlaws.length === 0) {
        targetsHtml = `<div style="padding: 40px; opacity: 0.5;">No active bounties in this sector.</div>`;
    } else {
        // Pre-resolve envoi names
        const wallets = outlaws.map(p => p.wallet);
        await Promise.all(wallets.map(w => getCachedEnvoiName(w)));

        outlaws.forEach(p => {
            const name = getCachedEnvoiName(p.wallet);
            const bounty = p.wanted_level * 50;
            const isMe = p.id === myClientId;
            
            targetsHtml += `
                <div class="player-item" style="padding: 15px; border-color: #ffd700;">
                    <div style="text-align: left;">
                        <b style="color: #ffd700;">${name}</b>
                        <div style="font-size: 0.75em; opacity: 0.6;">WANTED LEVEL: ${p.wanted_level}</div>
                    </div>
                    <div style="text-align: right;">
                        <div style="color: var(--neon-green); font-weight: bold;">${bounty} $VBV</div>
                        ${isHunter && !isMe ? `<button class="outline" style="font-size: 10px; padding: 6px 12px; border-color: #ffd700; color: #ffd700;" onclick="document.getElementById('bounty-board-overlay').remove(); window.sendChallenge('${p.id}')">HUNT TARGET</button>` : ''}
                        ${isMe ? `<span style="font-size: 10px; color: #ff4b4b;">YOU ARE THE TARGET</span>` : ''}
                    </div>
                </div>`;
        });
    }

    overlay.innerHTML = `
        <div class="glass-panel" style="width: 500px; text-align: center; border-color: #ffd700;">
            <h2 style="color: #ffd700; letter-spacing: 3px;">🎯 BOUNTY BOARD</h2>
            <p style="font-size: 0.8em; opacity: 0.7; margin-bottom: 20px;">High-infamy outlaws currently in the lobby. Hunters (Wanted ≤ 2) earn 50 $VBV per Wanted point on victory.</p>
            <div class="flex-col gap-10" style="max-height: 400px; overflow-y: auto; padding-right: 5px;">${targetsHtml}</div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('bounty-board-overlay').remove()">CLOSE BOARD</button>
        </div>`;
    document.body.appendChild(overlay);
}

export async function openRumorMill() {
    const state = window.GetGameState();
    const playerRewards = state.rewards[CONFIG.VBV_ASSET_ID] || 0;
    const rumorCost = 500; // Matches server-side cost

    if (playerRewards < rumorCost) {
        showToast(`❌ Insufficient $VBV. Spreading a rumor costs ${rumorCost} $VBV.`, "error");
        return;
    }

    const overlay = document.createElement("div");
    overlay.id = "rumor-mill-overlay";
    overlay.className = "overlay";

    let targetsHtml = '';
    if (lastLobbyPlayers.length === 0) {
        targetsHtml = `<div class="empty-notice">No other players in the lobby to spread rumors about.</div>`;
    } else {
        // Filter out self
        const otherPlayers = lastLobbyPlayers.filter(p => p.id !== myClientId);
        if (otherPlayers.length === 0) {
            targetsHtml = `<div class="empty-notice">No other players in the lobby to spread rumors about.</div>`;
        } else {
            // Pre-resolve envoi names for all targets
            const targetWallets = new Set(otherPlayers.map(p => p.wallet));
            await Promise.all(Array.from(targetWallets).map(w => getCachedEnvoiName(w)));

            targetsHtml = otherPlayers.map(p => {
                const targetName = getCachedEnvoiName(p.wallet);
                return `
                    <div class="rumor-target-item player-item">
                        <div class="rumor-target-info">
                            <b class="rumor-target-name">${targetName}</b>
                            <div class="rumor-target-stats">${p.reputation} REP | ${p.wins} WINS</div>
                        </div>
                        <div class="flex-row gap-5">
                            <button class="outline btn-rumor-positive" onclick="spreadRumor('${p.wallet}', 'positive', 1.1, 60)">+ POSITIVE</button>
                            <button class="outline btn-rumor-negative" onclick="spreadRumor('${p.wallet}', 'negative', 0.9, 60)">- NEGATIVE</button>
                        </div>
                    </div>
                `;
            }).join('');
        }
    }

    overlay.innerHTML = `
        <div class="rumor-mill-panel glass-panel">
            <h2>RUMOR MILL</h2>
            <p class="description">Influence market sentiment. Cost: <b class="text-neon-green">${rumorCost} $VBV</b></p>
            <div class="targets-scroll-list flex-col gap-10">
                ${targetsHtml}
            </div>
            <button class="outline mt-20" onclick="document.getElementById('rumor-mill-overlay').remove()">CLOSE</button>
        </div>
    `;

    document.body.appendChild(overlay);
}

export async function spreadRumor(targetWallet, type, strength, durationMinutes) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return showToast("❌ Not connected to server.", "error");
    if (!userAddress) return showToast("❌ Connect wallet first.", "error");

    const rumorCost = 500; // Matches server-side cost
    const state = window.GetGameState();
    const playerRewards = state.rewards[CONFIG.VBV_ASSET_ID] || 0;

    if (playerRewards < rumorCost) {
        showToast(`❌ Insufficient $VBV. Spreading a rumor costs ${rumorCost} $VBV.`, "error");
        return;
    }

    if (!confirm(`Are you sure you want to spread a ${type} rumor about ${getCachedEnvoiName(targetWallet)} for ${rumorCost} $VBV?`)) return;

    try {
        showToast(`📢 Spreading ${type} rumor about ${getCachedEnvoiName(targetWallet)}...`, "info");
        
        socket.send(JSON.stringify({
            type: "spread_rumor",
            payload: {
                target_wallet: targetWallet,
                type: type,
                strength: strength,
                duration_minutes: durationMinutes
            }
        }));

        document.getElementById("rumor-mill-overlay")?.remove();
    } catch (err) {
        showToast(`❌ Failed to spread rumor: ${err.message}`, "error");
    }
}

export function openTrophyView() {
    openSocialPanelOverlay('achievements');
}

/**
 * Opens the integrated Social Hub featuring Alliances, Career paths, and Achievements.
 * Utilizes orphaned _social.scss styles for immersive hierarchy.
 */
export async function openSocialPanelOverlay(initialTab = 'alliances') {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "social-hub-overlay";
    overlay.className = "overlay";

    overlay.innerHTML = `
        <div class="social-panel glass-panel">
            <div class="social-header">
                <span class="social-title">NEON SOCIAL HUB</span>
                <div class="social-stats">
                    <div class="stat-item">
                        <div class="stat-label">MOJO</div>
                        <div class="stat-value">${state.mojo || 0}</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-label">REP</div>
                        <div class="stat-value">${state.reputation || 0}</div>
                    </div>
                </div>
            </div>

            <div class="flex-row justify-center gap-10 mb-20">
                <button id="social-tab-alliances" class="tab-btn ${initialTab === 'alliances' ? 'active' : ''}" onclick="switchSocialTab('alliances')">🤝 ALLIANCES</button>
                <button id="social-tab-career" class="tab-btn ${initialTab === 'career' ? 'active' : ''}" onclick="switchSocialTab('career')">💼 CAREER</button>
                <button id="social-tab-achievements" class="tab-btn ${initialTab === 'achievements' ? 'active' : ''}" onclick="switchSocialTab('achievements')">🏆 VALOR</button>
            </div>

            <div id="social-content-hub" class="content-hub-scroll flex-col gap-15">
                <!-- Content injected by switchSocialTab -->
            </div>

            <button class="outline mt-20 w-full" onclick="document.getElementById('social-hub-overlay').remove()">DISCONNECT HUB</button>
        </div>
    `;

    document.body.appendChild(overlay);
    switchSocialTab(initialTab);
}

export async function switchSocialTab(tab) {
    const container = document.getElementById("social-content-hub");
    if (!container) return;

    // Update Tab Styles
    document.querySelectorAll('#social-hub-overlay .tab-btn').forEach(b => b.classList.remove('active'));
    const tabBtn = document.getElementById(`social-tab-${tab}`);
    if (tabBtn) tabBtn.classList.add('active');

    const state = window.GetGameState();
    container.innerHTML = `<div class="loading-text">Decrypting social datastreams...</div>`;

    if (tab === 'alliances') {
        const otherPlayers = lastLobbyPlayers.filter(p => p.id !== myClientId);
        if (otherPlayers.length > 0) {
            await Promise.all(otherPlayers.map(p => getCachedEnvoiName(p.wallet)));
        }

        // Filter for existing alliances (simulated from portfolio/employment state)
        const myClub = Object.values(globalClubs).find(c => c.id === state.employer_id);
        const allianceWallets = myClub ? Object.keys(myClub.members || {}) : [];

        const renderConnection = (p, isAlly) => `
            <div class="connection-item glass-panel m-0 ${isAlly ? 'border-cyan' : ''}">
                <div class="connection-avatar">
                    <img src="${p.avatar_url || 'Assets/Images/portraits/placeholder.webp'}" alt="Entity">
                </div>
                <div class="connection-info text-left">
                    <div class="connection-name font-bold ${isAlly ? 'text-neon-cyan' : ''}">${getCachedEnvoiName(p.wallet)}</div>
                    <div class="connection-role font-size-0-75em opacity-6">${p.social_rank} | ${p.job_role || 'Freelancer'}</div>
                        <div class="connection-status online">ACTIVE LINK</div>
                </div>
                <div class="connection-actions">
                        <button class="action-btn message" onclick="document.getElementById('social-hub-overlay').remove(); window.sendChallenge('${p.id}')" title="Challenge duel"></button>
                    ${!isAlly ? `<button class="action-btn invite" onclick="proposeAlliance('${p.id}')" title="Propose Alliance"></button>` : ''}
                    <button class="action-btn block" onclick="showToast('Entity communication restricted.', 'info')" title="Block stream"></button>
                </div>
            </div>`;

        const allies = otherPlayers.filter(p => allianceWallets.includes(p.wallet?.toLowerCase()));
        const others = otherPlayers.filter(p => !allianceWallets.includes(p.wallet?.toLowerCase()));

        container.innerHTML = `
            <div class="social-network flex-col gap-20">
                <div class="alliance-management">
                    <div class="network-header mb-10">
                        <span class="network-title text-neon-cyan font-bold letter-spacing-1">CONFIRMED ALLIANCES</span>
                        <div class="network-stats opacity-5 font-size-0-8em">STRENGTH: ${allies.length}</div>
                    </div>
                    <div class="connections-list alliances-grid">
                        ${allies.length === 0 ? '<div class="opacity-3 p-20 italic font-size-0-9em border-glass">No active alliance contracts found.</div>' : 
                            allies.map(p => renderConnection(p, true)).join('')}
                    </div>
                </div>

                <div class="sector-discovery">
                    <div class="network-header mb-10">
                        <span class="network-title text-neon-purple font-bold letter-spacing-1">SECTOR ENTITIES</span>
                        <div class="network-stats opacity-5 font-size-0-8em">DETECTED: ${others.length}</div>
                    </div>
                    <div class="connections-list discovery-grid">
                        ${others.length === 0 ? '<div class="opacity-3 p-20 italic font-size-0-9em">Scanning... no other entities in proximity.</div>' : 
                            others.map(p => renderConnection(p, false)).join('')}
                    </div>
                </div>
            </div>`;
    } else if (tab === 'career') {
        const tiers = [
            { name: "Iron", mojo: 0, desc: "A nobody in the neon gutter.", icon: "🌑" },
            { name: "Bronze", mojo: 100, desc: "A regular face at the local shops.", icon: "🥉" },
            { name: "Silver", mojo: 300, desc: "Gaining recognition in the sector.", icon: "🥈" },
            { name: "Gold", mojo: 600, desc: "An icon of the regional circuit.", icon: "🥇" },
            { name: "Diamond", mojo: 1000, desc: "Arena legend. The elite respect you.", icon: "💎" }
        ];

        container.innerHTML = `
            <div class="career-system">
                <div class="career-header">
                    <span class="career-title">PATH: <span class="job-role job-role--${(state.job_role || 'Freelancer').toLowerCase()}">${state.job_role || 'Freelancer'}</span></span>
                    <div class="career-level">MOJO ${state.mojo || 0}</div>
                </div>
                <div class="career-path">
                    ${tiers.map(t => {
                        const isCurrent = (state.mojo || 0) >= t.mojo && (state.mojo || 0) < (tiers[tiers.indexOf(t)+1]?.mojo || 9999);
                        const isCompleted = (state.mojo || 0) >= (tiers[tiers.indexOf(t)+1]?.mojo || 9999);
                        const isLocked = (state.mojo || 0) < t.mojo;
                        return `
                            <div class="career-tier ${isCurrent ? 'current' : ''} ${isCompleted ? 'completed' : ''} ${isLocked ? 'locked' : ''}">
                                <div class="tier-content">
                                    <div class="tier-icon">${t.icon}</div>
                                    <div class="tier-info">
                                        <div class="tier-name">${t.name}</div>
                                        <div class="tier-description">${t.desc}</div>
                                        <div class="tier-requirements">
                                            <span class="requirement ${!isLocked ? 'completed' : ''}">REQ: ${t.mojo} MOJO</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>
        `;
    } else if (tab === 'achievements') {
        const unlocked = new Set(state.achievements || []);
        const trophyCatalog = [
            { id: "FIRST_VICTORY", name: "First Victory", description: "Win your first match.", tier: 1 },
            { id: "TOURNAMENT_CHAMPION", name: "Tournament Champion", description: "Win a tournament.", tier: 2 },
            { id: "FIRST_HEIST", name: "First Heist", description: "Complete a successful Club heist.", tier: 1 },
            { id: "OUTLAW_SLAYER", name: "Outlaw Slayer", description: "Defeat a high-infamy opponent.", tier: 2 },
            { id: "ARENA_LEGEND", name: "Arena Legend", description: "Achieve legendary status in the arena.", tier: 3 },
            { id: "REHABILITATED", name: "Rehabilitated", description: "Pay off your courthouse fine and reset wanted level.", tier: 2 },
            { id: "GOVERNOR", name: "Governor", description: "Control 2+ territories as a club leader.", tier: 3 }
        ];

        container.innerHTML = `
            <div class="achievement-system">
                <div class="achievements-header">
                    <span class="achievements-title">HALL OF VALOR</span>
                    <div class="achievements-progress">UNLOCKED: <span class="progress-text">${unlocked.size}/${trophyCatalog.length}</span></div>
                </div>
                <div class="achievements-grid">
                    ${trophyCatalog.map(trophy => {
                        const hasUnlocked = unlocked.has(trophy.id);
                        return `
                            <div class="trophy-badge tier-${trophy.tier} ${hasUnlocked ? 'unlocked' : 'locked'}">
                                <div class="badge-icon ${hasUnlocked ? 'unlocked' : ''}">${hasUnlocked ? '🏆' : ''}</div>
                                <div class="badge-name">${trophy.name}</div>
                                <div class="badge-description">${trophy.description}</div>
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>
        `;
    }
}

export function openHeistPlanningOverlay() {
	const state = window.GetGameState();
	const overlay = document.createElement("div");
	overlay.id = "heist-overlay";
	overlay.className = "overlay";

	// Filter for external clubs only
	const clubs = Object.values(globalClubs).filter(c => c.id !== state.employer_id);
	
	overlay.innerHTML = `
		<div class="criminality-panel heist-terminal glass-panel animate-modal">
			<div class="criminality-header">
				<span class="criminality-title">HEIST PLANNING TERMINAL</span>
				<div class="criminality-stats">
					<div class="stat-item">
						<div class="stat-label stat-label-red">WANTED</div>
						<div class="stat-value stat-value-red">${state.wanted_level || 0}</div>
					</div>
					<div class="stat-item">
						<div class="stat-label stat-label-cyan">CUNNING</div>
						<div class="stat-value stat-value-cyan">${state.cunning || 0}</div>
					</div>
				</div>
			</div>

			<div class="p-20">
				<div class="criminality-targets mb-20">
					<div class="targets-header">
						<div class="targets-title">DETECTED SECTOR ENTITIES</div>
					</div>
					<div class="targets-list" style="grid-template-columns: repeat(2, 1fr); gap: 12px; max-height: 300px;">
						${clubs.length === 0 ? '<div class="grid-span-all opacity-3 italic py-40">No external club treasuries detected in local range.</div>' : 
							clubs.map(club => `
								<div class="target-item glass-panel m-0 p-15 hover-lift" onclick="updateHeistRiskAssessment('${club.id}')">
									<div class="target-info">
										<div class="target-name font-bold text-neon-purple" style="font-size: 1.1em;">${club.name.toUpperCase()}</div>
										<div class="target-details mt-5">
											<span class="detail-item wealth">${club.treasury.toFixed(2)} $VBV</span>
											<span class="detail-item level">MOJO: ${club.club_mojo}</span>
										</div>
									</div>
									<div class="target-select-btn mt-10">ANALYZE DEFENSES</div>
								</div>
							`).join('')}
					</div>
				</div>

				<div id="heist-risk-section" class="criminality-risk invisible mt-10 p-20 glass-panel animate-shimmer">
					<div class="risk-header mb-15">
						<span class="risk-icon">📡</span>
						<span class="risk-title">TACTICAL PROBABILITY ANALYSIS</span>
					</div>
					
					<div class="risk-meter">
						<div class="risk-labels">
							<span class="risk-low">SURGICAL</span>
							<span class="risk-high">CRITICAL RISK</span>
						</div>
						<div class="risk-bar">
							<div id="heist-risk-fill" class="risk-fill" style="width: 0%;"></div>
						</div>
					</div>
					
					<div class="flex-row justify-between align-center mt-15">
						<div id="heist-chance-text" class="progress-status"></div>
						<div id="heist-security-details" class="security-details font-mono"></div>
					</div>
					
					<div class="flex-row justify-center gap-15 mt-25">
						<button class="outline w-full secondary" onclick="document.getElementById('heist-overlay').remove()">ABORT OPS</button>
						<button id="heist-execute-btn" class="w-full danger btn-execute-strike">EXECUTE STRIKE</button>
					</div>
				</div>
			</div>
			
			<div class="text-center pb-20 opacity-4 font-size-0-7em letter-spacing-1">
				SECURITY ENFORCED BY THE INDUSTRIAL LOOP PROTOCOL
			</div>
		</div>
	`;

	document.body.appendChild(overlay);
}

export function updateHeistRiskAssessment(clubId) {
	const state = window.GetGameState();
	const club = globalClubs[clubId];
	const section = document.getElementById("heist-risk-section");
	const fill = document.getElementById("heist-risk-fill");
	const text = document.getElementById("heist-chance-text");
	const secText = document.getElementById("heist-security-details");
	const btn = document.getElementById("heist-execute-btn");

	if (!club || !section) return;

	// Visual activation
	section.classList.remove("invisible");
	document.querySelectorAll('.target-item').forEach(item => item.classList.remove('selected'));
	event.currentTarget.classList.add('selected');
	
	// Tactical Math: Base 50% + (Effective Cunning - Security Level)
	let securityStaff = 0;
	if (club.staff) Object.values(club.staff).forEach(role => { if(role === "Security") securityStaff++; });
	
	const securityLevel = (club.club_mojo / 10) + (securityStaff * 15);

	// Registry-aligned Trap Modifiers
	const trapModifiers = {
		"tripwire": -0.10,
		"sentry_turret": -0.25,
		"guard_dog": -0.05
	};
	let trapPenalty = 0;
	if (club.active_buffs) {
		Object.values(club.active_buffs).forEach(itemId => {
			if (trapModifiers[itemId]) trapPenalty += trapModifiers[itemId];
		});
	}

	const successChance = Math.min(0.95, Math.max(0.05, 0.50 + (state.cunning - securityLevel) / 100 + trapPenalty));
	const riskPercent = (1 - successChance) * 100;

	// UI Feedback
	fill.style.width = `${riskPercent}%`;
	text.innerHTML = `ESTIMATED SUCCESS: <b class="text-neon-green font-size-1-2em">${(successChance * 100).toFixed(0)}%</b>`;
	secText.innerHTML = `TARGET SEC_LEVEL: ${securityLevel.toFixed(1)} [STAFF: ${securityStaff}]`;
	
	btn.disabled = false;
	btn.onclick = () => executeHeistStrike(clubId);
}

/**
 * Dispatches the heist request to the server.
 */
export function executeHeistStrike(clubId) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	
	showToast("🔪 Deploying field operatives...", "warning");
	socket.send(JSON.stringify({
		type: "heist",
		payload: { target_club_id: clubId }
	}));
	document.getElementById("heist-overlay")?.remove();
}

export function handleHeistResult(payload) {
	const title = payload.status === "success" ? "HEIST SUCCESS" : "HEIST FAILED";
	const type = payload.status === "success" ? "success" : "error";
	const msg = payload.status === "success" ? `Successfully looted the treasury! Infamy increased.` : `The alarm was triggered! You barely escaped.`;
	showToast(`<b>${title}</b><br>${msg}`, type, 8000);

	// Trigger Kidnap Gambit if eligible
	if (payload.status === "success" && payload.kidnap_eligible) {
		setTimeout(() => openKidnapSelectionOverlay(payload.target_club_id), 1500);
	}
}

/**
 * Opens the Kidnap Selection interface following a successful heist.
 * Utilizes hostage-card and criminality styles for a high-stakes feel.
 */
export function openKidnapSelectionOverlay(targetClubId) {
	const club = globalClubs[targetClubId];
	if (!club) return;

	const overlay = document.createElement("div");
	overlay.id = "kidnap-selection-overlay";
	overlay.className = "overlay";
	
	overlay.innerHTML = `
		<div class="kidnap-selection-panel glass-panel animate-slide-up">
			<div class="criminality-header">
				<span class="criminality-title">KIDNAP GAMBIT</span>
			</div>
			
			<div class="p-20 text-center">
				<p>The heist was so clean you've cornered a high-value asset of <b class="text-neon-purple">${club.name}</b>.</p>
				
				<div class="hostage-card p-15 mt-20 mb-20 glass-panel">
					<div class="label">TARGET IDENTIFIED</div>
					<b class="owner-wallet">CLUB OWNER: ${club.owner_wallet.substring(0,12)}...</b>
					<div class="quote mt-10 italic">"A hostage ensures they won't retaliate... or provides a secondary payday."</div>
				</div>

				<div class="flex-col gap-10">
					<button class="outline w-full" onclick="document.getElementById('kidnap-selection-overlay').remove()">RELEASE & VANISH</button>
					<button class="w-full btn-execute-kidnap" onclick="executeKidnap('${targetClubId}')">EXECUTE KIDNAPPING</button>
				</div>
			</div>
		</div>
	`;
	document.body.appendChild(overlay);
}

export function executeKidnap(targetClubId) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	
	showToast("💀 Seizing the hostage...", "warning");
	socket.send(JSON.stringify({
		type: "kidnap_request",
		payload: { target_club_id: targetClubId }
	}));
	document.getElementById("kidnap-selection-overlay")?.remove();
}

export function showKidnapOverlay(payload) {
    const overlay = document.getElementById("kidnap-overlay");
    const content = document.getElementById("kidnap-content");
    if (!overlay || !content) return;

    const ransomValue = payload.ransom || payload.ransom_amount || 0;
    const perpWallet = payload.perp_wallet || "Unknown";

    content.innerHTML = `
        <p>Your card <strong>${payload.card_name}</strong> has been kidnapped!</p>
        <p>Ransom: <span class="ransom-amount">${(ransomValue / 1000000).toFixed(2)} $VBV</span></p>
        <p class="kidnap-victim-info">Kidnapper: ${perpWallet}</p>
        <button class="pay-ransom-btn" onclick="payRansom(${payload.card_id}, '${perpWallet}', ${ransomValue})">Pay Ransom</button>
        <p class="insurance-timer">Insurance recovery in: <span id="recovery-timer">48:00:00</span></p>
    `;
    overlay.classList.remove("hidden");

    // Start countdown timer
    startRecoveryTimer(payload.expires_at);
}

export function payRansom(cardId, perpWallet, ransomAmount) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    if (!perpWallet) {
        showToast("Unable to pay ransom: missing kidnapper wallet.", "error");
        return;
    }

    if (!ransomAmount || ransomAmount <= 0) {
        const amountInput = prompt("Enter the ransom amount in VBV to pay for this hostage card:", "0");
        if (!amountInput) return;
        const amountNumber = Number(amountInput);
        if (isNaN(amountNumber) || amountNumber <= 0) {
            showToast("Invalid ransom amount entered.", "error");
            return;
        }
        ransomAmount = Math.round(amountNumber * 1000000);
    }

    socket.send(JSON.stringify({
        type: "pay_ransom",
        payload: { card_id: cardId, perp_wallet: perpWallet, ransom_amount: ransomAmount }
    }));
}

export function releaseHostage(cardId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    if (!confirm(`Release Card #${cardId} back to its victim?`)) return;

    socket.send(JSON.stringify({
        type: "release_hostage",
        payload: { card_id: cardId }
    }));
}

export function startRecoveryTimer(expiresAt) {
    const timerEl = document.getElementById("recovery-timer");
    if (!timerEl) return;

    const interval = setInterval(() => {
        const now = Date.now();
        const remaining = expiresAt - now;
        if (remaining <= 0) {
            clearInterval(interval);
            timerEl.textContent = "00:00:00";
            return;
        }
        const hours = Math.floor(remaining / (1000 * 60 * 60));
        const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((remaining % (1000 * 60)) / 1000);
        timerEl.textContent = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }, 1000);
}
