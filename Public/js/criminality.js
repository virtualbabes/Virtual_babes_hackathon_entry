import { CONFIG } from './config.js';
import { socket, myClientId } from './network.js';
import { showToast, hideAllOverlays } from './ui.js';
import { userAddress, walletProvider, signClient } from './wallet.js';
import { getCachedEnvoiName, getNetworkConfig } from './utils.js';
import { globalClubs } from './admin.js';
import { lastLobbyPlayers } from './game.js';

export let rumorTimers = {};
export let activeRumors = {};

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

export async function openBountyBoard() {
    const state = window.GetGameState();
    const myWanted = state.wanted_level || 0;
    const isHunter = myWanted <= 2;
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
            const name = getCachedEnvoiName(p.wallet);
            const isMe = p.id === myClientId;
            const employer = globalClubs[p.employer_id]?.name || 'Freelancer';
            const mojo = p.mojo || 0;

            return `
                <div class="player-item border-gold">
                    <div class="text-left"><b class="text-gold">${name}</b><br><small>Wanted: ${p.wanted_level} | Mojo: ${mojo}<br>Affiliation: ${employer}</small></div>
                    <div class="text-right">
                        <b class="text-neon-green">${p.wanted_level * 50} $VBV</b>
                        ${isHunter && !isMe ? `<button class="outline btn-small border-gold mt-5" onclick="window.sendChallenge('${p.id}'); hideAllOverlays();">HUNT</button>` : ''}
                        ${isMe ? `<span class="text-error" style="font-size: 10px;">TARGET: YOU</span>` : ''}
                    </div>
                </div>`;
        }).join('');
    }

    overlay.innerHTML = `
        <div class="glass-panel w-500 border-gold" style="text-align: center;">
            <h2 class="text-gold">🎯 BOUNTY BOARD</h2>
            <p class="font-size-0-8em opacity-7 mb-20">Hunters (Wanted ≤ 2) earn 50 $VBV per Wanted point on victory.</p>
            <div class="flex-col gap-10 max-h-400 overflow-y-auto">${targetsHtml}</div>
            <button class="outline w-full mt-20" onclick="document.getElementById('bounty-board-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
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
    const overlay = document.createElement("div");
    overlay.id = "social-hub-overlay";
    overlay.className = "overlay";

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
                <button id="social-tab-achievements" class="tab-btn" onclick="switchSocialTab('achievements')">🏆 VALOR</button>
            </div>
            <div id="social-content-hub" class="flex-col gap-15 max-h-400 overflow-y-auto"></div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('social-hub-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
    switchSocialTab(initialTab);
}

export async function switchSocialTab(tab) {
    const container = document.getElementById("social-content-hub");
    if (!container) return;

    document.querySelectorAll('.tab-btn').forEach(b => b.classList.toggle('active', b.id === `social-tab-${tab}`));
    const state = window.GetGameState();
    container.innerHTML = `<div class="loading-text">Decrypting datastreams...</div>`;

    if (tab === 'alliances') {
        const others = lastLobbyPlayers.filter(p => p.id !== myClientId);
        container.innerHTML = others.map(p => `
            <div class="connection-item glass-panel m-0">
                <div class="connection-info"><b>${p.id}</b><br><small>${p.social_rank}</small></div>
                <button class="outline btn-small border-cyan" onclick="window.sendChallenge('${p.id}'); hideAllOverlays();">DUEL</button>
            </div>`).join('') || "No other entities detected.";
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
            { id: "OUTLAW_SLAYER", name: "Outlaw Slayer" }, { id: "ARENA_LEGEND", name: "Arena Legend" }
        ];
        container.innerHTML = `<div class="achievements-grid">${catalog.map(t => `
            <div class="trophy-badge ${unlocked.has(t.id) ? 'unlocked' : 'locked'}">
                <div class="badge-icon">${unlocked.has(t.id) ? '🏆' : '🔒'}</div>
                <div class="badge-name">${t.name}</div>
            </div>`).join('')}</div>`;
    }
}

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
    text.innerHTML = `ESTIMATED SUCCESS: <b class="text-neon-green">${(successChance * 100).toFixed(0)}%</b>`;
    btn.onclick = () => executeHeistStrike(clubId);
}

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
window.releaseHostage = releaseHostage;
