// ============================================================================
// game_screen.js — Dynamic render of the board / game screen.
// Previously static markup in index.html (#main-game-container). Now rendered by
// app.js so the entire UI flows through the app.js module graph (no static main
// menu in index.html). Element IDs/classes are preserved EXACTLY so syncUI() and
// engine-driven handlers (handleMaintenanceUI, showMainGameContainer, buildEmptyBoard)
// keep working unchanged.
// ============================================================================
export const GAME_SCREEN_HTML = `    <div class="app-shell main-game-container" id="main-game-container">
        <!-- Top Bar: Faucet, Status, Navigation -->
        <div class="top-bar">
                    <div id="faucet-display" class="faucet-stat" onclick="if(window.openFaucetDashboard) window.openFaucetDashboard()">0.00 $VBV</div>
            <div id="season-countdown-widget" class="hidden season-info flex-col">
                <span class="font-small-caps letter-spacing-1">SEASON <span id="season-num-display">1</span> ENDS</span>
                <span id="season-timer" class="font-bold-rajdhani">--:--:--</span>
            </div>
            
            <!-- Solvency Dashboard: Real-time Coverage Ratio (Task 1101) -->
            <div id="solvency-hud-container" class="hidden ml-10"></div>

            <div id="payout-config-widget" class="glass-panel payout-widget" onclick="openPayoutSettings()">
                <span class="opacity-6 font-size-0-7em">VOI PAYOUT:</span>
                <span id="payout-address-display" class="font-mono text-neon-green">Not Set (Default)</span>
            </div>
            <div id="turn-display" class="round-title m-0 text-base">LOBBY</div>
            <div id="transaction-status" class="hidden font-small opacity-6 ml-10"></div>
            <div id="engine-status" class="font-small opacity-7 ml-10">ENGINE OFFLINE</div>
            <div id="latency-monitor" class="font-small opacity-6">
                LATENCY: <span id="latency-display">-- ms</span>
            </div>
            
            <!-- Sabotage HUD: Visible to Owners during Blackouts (Task 793) -->
            <div id="sabotage-hud-container" class="hidden ml-10"></div>

            <!-- Mutation Stability HUD: Visible to Lab Personnel (Task 848) -->
            <div id="mutation-stability-hud" class="hidden ml-10"></div>

            <!-- Active Bounty Warning: Visible to Outlaws (Task 854) -->
            <div id="active-bounty-warning" class="hidden ml-10"></div>

            <!-- Bounty Hunter HUD: Visible to clean players (Task 855) -->
            <div id="bounty-hunter-hud" class="hidden ml-10"></div>

            <!-- Commission Summary HUD: Visible to Regional Governors (Task 885) -->
            <div id="commission-summary-hud" class="hidden ml-10"></div>

            <!-- District Stabilizer HUD: Displays remaining time for mojo field (Task 936) -->
            <div id="district-stabilizer-hud" class="hidden ml-10"></div>

            <!-- Mojo Decay Status HUD: Displays current decay rate (Task 937) -->
            <div id="mojo-decay-status-hud" class="hidden ml-10"></div>

            <!-- Bounty Tally HUD: Sector prize pool (Task 859) -->
            <div id="bounty-tally-hud" class="hidden ml-10"></div>

            <!-- Mission HUD: Real-time objective tracking (Task 2232) -->
            <div id="mission-hud-container" class="hidden ml-10"></div>

            <button id="music-toggle-btn" class="outline p-5-10 font-size-10px ml-10 mr-10" onclick="toggleMuteMusic()" aria-label="Toggle Background Music">🎵</button>
            <div id="wallet-controls">
                <button id="wallet-btn" onclick="handleWalletAction()">Connect Authority Wallet</button>
            </div>
        </div>

        <!-- MAIN SCREEN: ONE unified glass surface, three seamless regions -->
        <div class="main-screen">

            <!-- LEFT REGION: side-rail (matchmaking + live lobby chat) -->
            <section class="region side-rail">
                <div class="arena-side-head flex-row justify-between align-center">
                    <h3>Arena</h3>
                    <button class="pp-mini-btn" onclick="window.openWorldDashboardToTab('portfolio')">☰ Portfolio</button>
                </div>

                <!-- Matchmaking Controls (always-available actions) -->
                <div id="matchmaking-container" class="matchmaking-box text-center">
                    <div id="queue-status" class="font-small mb-10 opacity-8">Ready for automatic pairing?</div>
                    <button id="btn-matchmaking" onclick="toggleMatchmakingQueue()" class="w-full m-0">Join Matchmaking Pool</button>
                </div>

                <div id="local-ban-cooldown" class="cooldown-container hidden">
                    <div class="cooldown-header">ARENA RESTRICTION ACTIVE</div>
                    <div class="cooldown-progress-bg">
                        <div id="ban-progress-fill" class="cooldown-progress-fill"></div>
                    </div>
                    <div id="ban-countdown-timer" class="font-small mt-10">00:00:00</div>
                </div>

                <!-- Chat mirror: the live lobby chat. Played into #chat-display by
                     renderChatMessage(); this element must persist so chat stays alive.
                     Full lobby (players list, spectate/replay) is in the Profile ▸ Lobby panel. -->
                <div class="chat-container">
                    <div class="chat-title font-small opacity-6 mb-5">LOBBY CHAT</div>
                    <div id="chat-display"></div>
                    <div class="chat-input-area">
                        <input type="text" id="chat-input" class="glass-input" placeholder="Type a message..." title="Lobby Chat" onkeydown="handleChatKey(event)" aria-label="Chat message">
                        <button class="chat-send-btn" onclick="sendChatMessage()">SEND</button>
                    </div>
                </div>
            </section>

            <!-- CENTER REGION: arena-stage (board / spectate / replay / placeholder) -->
            <section class="region arena-stage">
                <div id="tournament-banner" class="hidden tournament-highlight-banner text-center">
                    <div class="font-bold font-xsmall letter-spacing-2">LIVE TOURNAMENT</div>
                    <div id="tournament-status-text" class="font-small mb-5 mt-5">Registration Open!</div>
                    <div class="flex-row justify-center gap-5">
                        <button id="tournament-reg-btn" class="outline p-5-10 font-size-10px" onclick="window.openTournamentBracket(); syncUI();">JOIN EVENT</button>
                        <button class="outline p-5-10 font-size-10px border-purple text-purple" onclick="openTournamentBracket()">VIEW BRACKET</button>
                    </div>
                </div>

                <div id="arena-header" class="arena-header">
                    <h2 id="deck-rating-display" class="deck-rating-label">[Z]</h2>
                    <div id="rewards-dashboard" class="rewards-info-banner"></div>
                </div>

                <!-- Opponent Info -->
                <div id="p2-info" class="flex-row gap-15 mb-10 w-full justify-end pr-32 hidden">
                    <div class="text-right">
                        <div id="p2-name" class="font-bold font-small">Bot</div>
                    </div>
                    <div id="p2-avatar" class="avatar-frame border-error avatar-small"></div>
                </div>

                <!-- 3x3 Grid -->
                <div id="board-container" class="battle-board">
                    <canvas id="particle-canvas" class="particle-canvas"></canvas>
                </div>

                <!-- Player Info -->
                <div id="p1-info" class="flex-row gap-15 mt-20 w-full pl-32 hidden">
                    <div id="p1-avatar" class="avatar-frame avatar-small"></div>
                    <div>
                        <div id="p1-name" class="font-bold font-small">You</div>
                        <div id="cyber-jammer-status" class="hidden font-size-0-7em text-neon-purple mt-5 p-4-8 line-height-1-2 max-w-200">
                            <span class="text-neon-purple">📡 JAMMER ACTIVE</span>
                        </div>
                        <!-- Heist Saboteur Progress -->
                        <div id="heist-saboteur-progress" class="hidden mt-5"></div>

                        <!-- Moderation Notice for Avatars -->
                        <div id="avatar-notice-banner" class="hidden avatar-notice font-size-0-7em mt-5 p-4-8 line-height-1-2 max-w-200"></div>
                    </div>
                </div>

                <div id="ai-thinking-indicator" class="hidden flex-row gap-10 mt-10 text-italic text-neon-cyan">
                    <div class="thinking-spinner thinking-spinner-small"></div>
                    <span>Opponent evaluating move...</span>
                </div>

                <!-- Lobby placeholder: shown ONLY when not in a match/spectate/replay -->
                <div id="lobby-placeholder" class="lobby-placeholder hidden">
                    <div class="lobby-placeholder-icon">🎮</div>
                    <h3>Arena Idle</h3>
                    <p>No active match. The battle board comes online when you join a match, enter the matchmaking pool, or spectate a live game.</p>
                    <div class="lobby-placeholder-actions">
                        <button class="outline border-cyan text-neon-cyan" onclick="window.StartMatch(); syncUI();">▶ Start / Join Match</button>
                        <button class="outline border-purple text-purple" onclick="toggleMatchmakingQueue()">⚡ Matchmaking Pool</button>
                    </div>
                    <p class="lobby-hint">Open the <b>🌍 World Dashboard</b> ▸ <b>Player Hub</b> for the live player list, spectate &amp; replay. Matchmaking is always above. Star a category (☆) to pin it to the Constellation Hub for quick access.</p>
                </div>
            </section>

            <!-- RIGHT REGION: main screen actions (minimal) -->
            <section class="region action-dock">
                <div id="action-dropdown" class="action-dropdown">
                    <button id="action-dropdown-toggle" class="outline border-cyan text-neon-cyan w-full" onclick="window.toggleActionDropdown()">☰ Actions &amp; Categories ▾</button>
                    <div id="action-bar" class="action-dropdown-list flex-col gap-10 mt-10">
                    <!-- World Dashboard — the SINGLE navigation surface: categories -> features -->
                    <button id="world-dashboard-btn" class="outline border-cyan text-neon-cyan" onclick="window.openWorldDashboard()">🌍 World Dashboard</button>
                    <!-- Constellation Hub — quick access to STARRED CATEGORIES -->
                    <button id="constellation-btn" class="outline border-gold text-yellow" onclick="window.openConstellationHub()">✨ Constellation Hub</button>
                    <!-- Quick Match — fast PvP entry -->
                    <button id="quick-match-btn" onclick="toggleMatchmakingQueue()">⚡ Quick Match</button>
                    <!-- Match Arena and Player Profile are deliberately NOT duplicated here.
                         Each surface has exactly ONE entry point, owned by the World
                         Dashboard: Play & Board ▸ Match Arena, Player Hub ▸ Player Profile. -->
                    <div id="history-display" class="font-small scroll-y max-h-150"></div>
                    </div>
                </div>
            </section>

        </div>
    </div>`;

// Render the board/game screen into <body>. Idempotent.
// Starts hidden (nexus-lobby-active) so the constellation hub is the main screen in
// the lobby; syncUI() toggles nexus-lobby-active so the game screen appears during a match.
export function renderGameScreen() {
  if (document.getElementById('main-game-container')) return; // idempotent
  const tpl = document.createElement("div");
  tpl.innerHTML = GAME_SCREEN_HTML.trim();
  const el = tpl.firstElementChild;
  if (!el) return;
  document.body.appendChild(el);
  document.body.classList.add('nexus-lobby-active'); // hidden until a match is Active
}
