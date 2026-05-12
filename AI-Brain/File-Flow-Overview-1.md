 ## high-level overview of how the .go, .js, .scss, .css, .html, and Dockerfile files work together to create the Virtualbabes Arena application.

High-Level Synergy: Virtualbabes Arena Architecture
The Virtualbabes Arena is designed as a blockchain-integrated platform where real-time multiplayer gaming meets decentralized economics. This requires a robust interplay between server-side logic, client-side game engine, and a dynamic user interface, all deployed efficiently.

Backend (Go - .go files in Root DIR/A1. Server):

Purpose: This is the authoritative core of the application. It manages all server-side logic, real-time communication, and interaction with the blockchain.
Key Files: server.go acts as the central hub, handling WebSocket connections (Public/js/network.js connects here), HTTP APIs, rate-limiting, and client concurrency. Other .go files (like lobby_manager.go, battle_service.go, economy_service.go, tournament_manager.go, oracle_service.go, etc.) decompose the backend into domain-specific services.
Functionality:
Game State Management: The server maintains real-time in-memory game state for matches, lobbies, and tournaments.
Blockchain Interaction: oracle_service.go and others use indexers to read authenticated data and receipts directly from blockchain networks (Voi, Algorand). This is crucial for verifying transactions (e.g., tournament buy-ins), fetching NFT metadata, and reconstructing critical game state (leaderboards, match history) without relying on a traditional database.
Real-time Communication: Uses WebSockets to broadcast updates (e.g., lobby changes, chat messages, match events) to connected clients.
Security: Implements the "Switchboard Pattern" for secure faucet payouts (server-side signing) and client-side nonce proofs for verification.
Business Logic: Orchestrates matchmaking, tournament progression, economic services (faucet, loans, auctions), criminality, and social layers.
Game Engine (Go WASM - Root DIR/A2. Game-interaction/main.go compiled to Public/main.wasm):

Purpose: This Go code, compiled to WebAssembly (main.wasm), runs directly in the user's browser. It provides the core game rules, AI logic, and deterministic calculations.
Key Files: main.go (Go source) compiles to Public/main.wasm and Public/wasm_exec.js.
Functionality:
Deterministic Gameplay: Encapsulates the core game logic (Triple Triad-inspired rules like "Same," "Plus," "Combo"), AI move calculations, and deck building heuristics.
Client-Side Computation: Allows complex game logic to run efficiently in the browser, ensuring tamper-proof calculations and reducing server load for immediate feedback.
JS Bridge: Exposes functions (like window.GetGameState, window.PlaceCard, window.SetAvatar, window.syncUI) that the frontend JavaScript (app.js) can call to interact with the game engine.
Frontend (JavaScript, SCSS, CSS, HTML - Public directory):

Purpose: Provides the interactive user interface, handles client-side logic, communicates with the backend, and renders the game.
Key Files:
Public/index.html: The single-page application entry point. It loads the compiled main.wasm (via wasm_exec.js), the main app.js, and the compiled styles.css. It defines the base structure of the UI.
Public/app.js: The central client-side orchestrator. It initializes the WASM engine, establishes WebSocket connections, integrates WalletConnect (Public/js/wallet.js), manages UI state, and calls functions from the WASM engine and other modular JavaScript files. It contains the primary syncUI function that updates the entire user interface based on the GetGameState() from WASM.
Public/js/*.js (e.g., game.js, ui.js, network.js, wallet.js, deck.js, economy.js, criminality.js, admin.js, leaderboard.js, utils.js, audio.js, particles.js): These are modular JavaScript files that break down specific functionalities. They interact with app.js, the WASM engine, and the backend WebSockets to provide features like game logic, UI rendering, network communication, wallet interactions, deck management, economic transactions, criminal actions, admin controls, leaderboards, utility functions, audio, and visual effects.
Public/collective-intelligence.js: This specific JS file generates NPC taunts based on player playstyle, which is then rendered by app.js using renderChatMessage.
Public/src/scss/*.scss: These SCSS files (e.g., _neon-glass.scss, _dashboard.scss, _variables.scss, _criminality.scss, _territory.scss) define the "neon-glass" aesthetic. They are pre-processed into Public/styles.css.
Public/styles.css: The compiled stylesheet applied to index.html, dynamically styling elements rendered by JavaScript.
Deployment (Dockerfile - Root DIR/B7. Docker-file/Dockerfile):

Purpose: To package the entire application (Go backend, WASM game engine, frontend assets) into a portable, self-contained unit for consistent deployment.
Functionality: The Dockerfile specifies how to build the application image. This typically involves:
Building the Go backend (server.go and related services).
Compiling main.go to main.wasm.
Copying the compiled WASM files and all frontend assets (HTML, JS, CSS, images, audio) into the Docker image.
Defining the entry point to run the Go backend server, which then serves the static frontend assets.
How they Synergize:
Real-time Loop: The Go backend (server) broadcasts game state updates over WebSockets. Public/js/network.js receives these. Public/app.js's handleServerMessage processes them, potentially calling functions in the Go WASM engine (main.wasm) to update the client-side game state. The syncUI function in app.js then reads this updated WASM state and re-renders the index.html (styled by styles.css/.scss) accordingly.
User Interaction: A user action (e.g., clickGrid in game.js) calls a WASM function (window.PlaceCard). If successful and in multiplayer, game.js sends a message back to the Go backend via network.js WebSocket, completing the loop.
Data Persistence: While the Go backend manages a live in-memory representation, actual persistent storage for critical game data (like tournament results, player reputations, NFT ownership) is offloaded to blockchain indexers. The Go services (oracle_service.go, economy_service.go) query these indexers, and the results are then communicated to the frontend. Public/js/utils.js (e.g., resolveEnvoiName) handles client-side caching and resolution of some blockchain data.
Deployment: The Dockerfile bundles all these disparate components (Go executable, WASM binary, JS, HTML, CSS) into a single artifact, ensuring that the entire application stack can be deployed reliably and consistently in any Docker-compatible environment.
This overview paints a broad picture. To provide you with a more detailed module hierarchy flow document, I need to focus on specific sections.

From the AI-Brain/DIR.md document, which of these main groups would you like me to analyze in more detail regarding its internal structure and how it flows with other parts of the system?

4. Public-js-Scripts
5. Root DIR/A. .go-files (or specify A1. Server or A2. Game-interaction)
5. Root DIR/B. System-Root-system (for configuration and deployment files)
3. Public Visuals (if you'd like to understand the asset integration flow, though less about module hierarchy)
2. Public Audio Asset Files (similar to visuals)


## Public-Js-Detail
Public-js-Scripts directory contains the entire client-side application logic for the Virtualbabes Arena. It's designed as a modular Single-Page Application (SPA) that interacts with a Go WebAssembly (WASM) game engine for core logic and a Go backend server via WebSockets and HTTP APIs for real-time updates and blockchain interactions.

Here's a detailed breakdown of the module hierarchy and interaction flow:

Module Hierarchy and Interaction Flow
The core principle is that Public/app.js acts as the central orchestrator, initializing the Go WASM engine and WebSocket connection, and then coordinating updates across various specialized JavaScript modules.

1. Public/app.js - The Central Orchestrator
Purpose: This is the main client-side application entry point. It's responsible for:
Loading and initializing the Go WASM game engine (main.wasm).
Establishing and managing the WebSocket connection to the Go backend.
Initializing WalletConnect for blockchain interactions.
Orchestrating UI updates by calling its syncUI function, which reads state from the WASM engine and updates various DOM elements.
Exposing global functions to the window object for inline HTML event handlers.
Key Imports: Almost all other JS modules are imported here to centralize their functionality. This includes collectiveIntelligence, CONFIG, initWebSocket, handleServerMessage, showToast, initWalletConnect, fetchLeaderboard, toggleMatchmakingQueue, openDeckManager, openShopsOverlay, openCourthouse, setMasterVolume, initParticleSystem, getCachedEnvoiName, etc.
Key Exports: None directly, but it exposes many functions globally to window (e.g., window.syncUI, window.handleWalletAction, window.openDeckManager) for index.html to call.
Interaction Flow:
Initialization: On window.onload, it loads main.wasm (via wasm_exec.js), sets up the Go WASM engine, configures CONFIG.API_BASE and CONFIG.ASSET_URL within WASM, then calls initWebSocket (from network.js) and initWalletConnect (from wallet.js).
UI Loop (syncUI): This is the heart of the frontend. It frequently calls window.GetGameState() (from WASM) to retrieve the current application state. Based on this state, it then:
Determines which UI overlays to show/hide (hideAllOverlays).
Updates player information, scores, and game board.
Refreshes dashboards (rewards, faucet, latency).
Triggers updates in other modules (e.g., updateAdminRewardList, renderRumorBoard).
WASM Bridge: Directly calls functions exposed by the Go WASM engine (e.g., window.GetGameState, window.SetAvatar, window.PlaceCard, window.SetMaintenanceState, window.SetTestingMode, window.SetMusicVolume).
Backend Communication: Delegates sending WebSocket messages to network.js (e.g., register_wallet, join_queue, move).
Module Coordination: Calls functions from other imported modules to perform specific tasks (e.g., fetchLeaderboard, renderDeckManager, openHeistPlanningOverlay).
2. Public/wasm_exec.js - Go WASM Runtime
Purpose: This is the standard Go WebAssembly glue code. It provides the necessary JavaScript environment for main.wasm to run in the browser, handling memory, system calls, and the bridge between Go and JavaScript.
Key Imports: None (it's a standalone script).
Key Exports: The Go class, which app.js instantiates.
Interaction Flow: Loaded by index.html, it enables app.js to load and execute main.wasm. All direct calls between JavaScript and the Go WASM engine (e.g., window.GetGameState()) are facilitated by this script.
3. Public/js/config.js - Global Configuration
Purpose: Defines static and dynamically updated global configuration constants (backend URLs, asset IDs, WalletConnect project ID, blockchain chain IDs).
Key Imports: None.
Key Exports: CONFIG object.
Interaction Flow: Imported by almost all other JS modules that need configuration data. network.js dynamically updates CONFIG.VAULT_ADDRESS, CONFIG.VBV_ASSET_ID, CONFIG.AVOI_ASSET_ID based on server messages.
4. Public/js/network.js - WebSocket Communication
Purpose: Manages the WebSocket connection to the Go backend. It handles incoming server messages, dispatches them to appropriate client-side handlers, and manages connection/reconnection logic.
Key Imports: CONFIG, showToast, setTransactionStatus, updateWalletUI, handleTournamentUI, updatePlayerList, updateMarketTicker, handleMaintenanceUI, updateAdminNetworkUI, updateActiveRumors, handleHeistResult, showKidnapOverlay, startRecoveryTimer, setLastLobbyPlayers, setMyPlayerIndex, setCurrentOpponentId, setSpectatorMatchState, renderChatMessage, saveMatchResult, setMatchHistorySaved.
Key Exports: socket, myClientId, nonceResolver, initWebSocket, sendPing, handleServerMessage.
Interaction Flow:
Connection: app.js calls initWebSocket to establish the connection.
Message Dispatch: handleServerMessage is the central dispatcher for server messages. It uses a switch statement to route messages (e.g., lobby_update, move, chat, identity, rewards_update) to specific handler functions in other modules (e.g., updatePlayerList in game.js, handleTournamentUI in leaderboard.js).
WASM Updates: Many server messages trigger calls to WASM functions (e.g., window.SyncFullProfile, window.SyncRules, window.SyncMove).
UI Batching: Uses requestBatchedSync to optimize syncUI calls, preventing UI thrashing.
5. Public/js/wallet.js - Wallet Management
Purpose: Handles all client-side wallet interactions: connecting, disconnecting, signing transactions, and WalletConnect integration.
Key Imports: CONFIG, showToast, setTransactionStatus, hideAllOverlays, showMainGameContainer, getNetworkConfig, socket, setNonceResolver, fetchUserNFTs.
Key Exports: userAddress, isVerified, linkedWallets, walletProvider, signClient, wcModal (and their setters), initWalletConnect, handleWalletAction, connectWith, disconnectUserWallet, updateWalletUI.
Interaction Flow:
Initialization: app.js calls initWalletConnect.
Connection: connectWith handles various wallet providers (Nautilus, Kibisis, WalletConnect), obtains the user's address, and calls window.connectWallet (WASM).
Authentication: Used by admin.js for admin panel authentication and by criminality.js/leaderboard.js for signing blockchain transactions.
Backend Communication: Sends register_wallet and link_wallet_request messages via network.js's socket.
6. Public/js/ui.js - General UI Rendering & Feedback
Purpose: Provides generic UI functions like displaying toasts, transaction statuses, managing overlays, and rendering common UI elements (e.g., card HTML, tooltips).
Key Imports: CONFIG, myClientId, currentLatency, userAddress, myPlayerIndex, currentOpponentId, masterVolume, updateAdminRewardList, updateActiveRumors, startSeasonTimer, getAssetSymbol.
Key Exports: tooltipEl, maintenanceTicker, showToast, setTransactionStatus, hideAllOverlays, showMainGameContainer, highlightStartButton, handleMaintenanceUI, showTournamentTransition, updateDynamicArenaFloor, renderCardHTML, movePowerTooltip, hidePowerTooltip, showQuickCastMenu, handleLocalBanUI, showMatchPreview, shareTournamentVictory, openSettingsOverlay, closeSettingsOverlay.
Interaction Flow:
WASM Interaction: Calls window.GetGameState() for state, window.SetMaintenanceState, window.GetLevelLabelForDisplay, window.PlaySound (via global exposure).
DOM Manipulation: Directly manipulates the DOM to update text, show/hide elements, and apply styles.
Timers: Manages countdown timers for maintenance and local bans.
7. Public/js/game.js - Core Game Logic
Purpose: Implements client-side game logic for active matches, matchmaking, chat, and challenge handling. It's the primary interface for user interaction with the WASM game engine.
Key Imports: CONFIG, socket, myClientId, showToast, hideAllOverlays, showMatchPreview, renderCardHTML, collectiveIntelligence, userAddress, getCachedEnvoiName, resolveEnvoiName.
Key Exports: activeCardId, myPlayerIndex, lastLobbyPlayers (and their setters), buildEmptyBoard, toggleMatchmakingQueue, sendChatMessage, clickGrid, calculateDeckRating, showPowerTooltip, etc.
Interaction Flow:
WASM Interaction: Calls window.GetGameState() for game state. Calls window.SetInMatchmakingQueue, window.SetPhase, window.SetLocalPlayerIndex, window.SyncOpponentProfile, window.SyncOpponentDeck, window.StartMatch, window.ResetGame, window.SetBoardState, window.ForceActive, window.PlaceCard, window.ApplyArtifactToBoard, window.PlaySelectSound, window.syncUI (via global exposure).
Backend Communication: Sends join_queue, leave_queue, chat, challenge (invite, accept, decline, sync_back), report_gloat, spectate, move, use_item messages via network.js's socket.
UI Updates: Updates player lists, chat display, match history, and manages tooltips.
8. Public/js/deck.js - Deck & Avatar Management
Purpose: Manages the player's card inventory, deck building, avatar selection, and avatar cropping/upload.
Key Imports: CONFIG, socket, showToast, renderCardHTML, userAddress, linkedWallets, getNetworkConfig, calculateDeckRating, activeCardId.
Key Exports: userNFTs, currentAvatarUrl, cropState, isCropInitialized (and their setters), openDeckManager, closeDeckManager, renderDeckManager, renderAvatarGrid, applyAvatarFilters, selectAvatar, setupCropEvents.
Interaction Flow:
WASM Interaction: Calls window.GetGameState() for current deck/inventory. Calls window.selectCard, window.RemoveFromDeck, window.AddToDeck, window.SelectDeck, window.SetAvatar to modify WASM game state. Triggers window.syncUI() (via global exposure).
Backend API Calls: refreshInventory fetches NFT data from indexers (via fetch) for the user's primary and linked wallets.
Backend Communication: closeDeckManager sends update_rating message. setupCropEvents sends register_avatar message via network.js's socket.
UI Rendering: Populates inventory/deck displays and the avatar selection grid. Handles interactive image cropping.
9. Public/js/economy.js - Economic Features
Purpose: Manages client-side logic for shops, black market, portfolio view, and share trading.
Key Imports: CONFIG, socket, showToast, hideAllOverlays, userAddress, walletProvider, signClient, getCachedEnvoiName, getNetworkConfig, resolveEnvoiName, globalClubs, lastLobbyPlayers, syncUI.
Key Exports: tradeShares, openBlackMarket. (Many other functions are exposed globally by app.js after being imported here).
Interaction Flow:
WASM Interaction: Calls window.GetGameState() for player stats and game state. Calls window.syncUI() (via global exposure).
Backend API Calls: Fetches data from /api/black-market, /api/auctions, and sends POST requests for buyBlackMarketItem, promptBid, submitConsignment, submitClubFoundry.
Blockchain Interaction: submitClubFoundry constructs and signs Algorand/Voi transactions.
Backend Communication: Sends trade_shares, purchase_item, create_club messages via network.js's socket.
UI Rendering: Dynamically creates and appends various economic overlays.
10. Public/js/criminality.js - Criminality Features
Purpose: Manages client-side logic and UI for features like the Courthouse, Heists, Kidnapping, Bounty Board, and Rumor Mill.
Key Imports: CONFIG, socket, myClientId, showToast, hideAllOverlays, userAddress, walletProvider, signClient, getCachedEnvoiName, getNetworkConfig, globalClubs, lastLobbyPlayers, myPlayerIndex, setCurrentOpponentId, setMyPlayerIndex, syncUI.
Key Exports: rumorTimers, activeRumors, updateActiveRumors, openCourthouse, submitCourthouseFine, initiateBail, openSecuritySentry, deployTrap, openBountyBoard, openRumorMill, spreadRumor, openTrophyView, openSocialPanelOverlay, switchSocialTab, openHeistPlanningOverlay, updateHeistRiskAssessment, executeHeistStrike, handleHeistResult, openKidnapSelectionOverlay, executeKidnap, showKidnapOverlay, payRansom, releaseHostage, startRecoveryTimer.
Interaction Flow:
WASM Interaction: Calls window.GetGameState() for player stats and game state. Calls window.syncUI() (via global exposure).
Backend API Calls: submitCourthouseFine makes a POST request to /api/courthouse/reset.
Blockchain Interaction: submitCourthouseFine and initiateBail construct and sign Algorand/Voi transactions.
Backend Communication: Sends bail_card, use_item, heist, kidnap_request, pay_ransom, release_hostage, spread_rumor messages via network.js's socket.
UI Rendering: Dynamically creates and appends various criminality overlays and updates timers.
11. Public/js/admin.js - Admin Panel
Purpose: Provides client-side functionality for the admin control panel, including fetching logs, managing networks, rewards, bans, and system settings.
Key Imports: CONFIG, socket, setNonceResolver, showToast, setTransactionStatus, userAddress, walletProvider, signClient, linkedWallets, getAssetSymbol, getNetworkConfig, fetchLeaderboard.
Key Exports: availableNetworks, globalClubs, adminFocusNetwork, ignoredReporters (and their setters), getAdminHeaders, ignoreReporter, fetchAdminLogs, adminRefillVault, updateAdminRewardList, adminAddReward, adminRemoveReward, adminAddNetwork, adminBroadcast, adminUpdateRules, adminBanWallet, adminAvatarBan, adminBanWalletFromLog, adminUpdatePowerScaling, adminToggleMaintenance, adminToggleDevMode, adminResetStats, adminSimulateTournament, adminLogTicker, startAdminLogPolling, fetchLastAdminAction, updateAdminNetworkUI, onAdminNetworkSelectChange.
Interaction Flow:
Authentication: getAdminHeaders requests a nonce from the backend via WebSocket, then signs it with the user's wallet for HTTP header authentication.
Backend API Calls: Most admin functions make fetch requests to /api/admin/* endpoints on the Go backend.
WASM Interaction: adminToggleDevMode calls window.SetTestingMode.
UI Updates: Provides feedback via showToast and setTransactionStatus, and updates admin-specific UI elements.
12. Public/js/leaderboard.js - Leaderboard & Tournaments
Purpose: Manages client-side logic for displaying leaderboards, tournament history, and current tournament status.
Key Imports: CONFIG, socket, showToast, showTournamentTransition, tooltipEl, userAddress, walletProvider, signClient, getCachedEnvoiName, resolveEnvoiName, getNetworkConfig.
Key Exports: totalTournaments, lastTournamentData, seasonEnd (and their setters), fetchLeaderboard, startSeasonTimer, switchHofTab, toggleTournamentDetails, registerForTournament, openTournamentBracket, closeTournamentBracket.
Interaction Flow:
WASM Interaction: Calls window.GetGameState() for state. Calls window.SetPhase() and window.syncUI() (via global exposure).
Backend API Calls: Fetches data from /api/leaderboard, /api/tournament/history, /api/season/history. registerForTournament makes a POST request to /api/tournament/register after a blockchain transaction.
Blockchain Interaction: registerForTournament constructs and signs Algorand/Voi transactions.
UI Rendering: Populates leaderboard lists, tournament history, and manages pagination/timers.
13. Public/js/audio.js - Audio Controls
Purpose: Manages global audio settings (master, music, SFX volumes) and provides functions to toggle mute states.
Key Imports: ../app.js (syncUI - now exposed globally).
Key Exports: masterVolume, musicVolume, sfxVolume (and their setters).
Interaction Flow:
Local Storage: Loads initial volume settings.
WASM Interaction: Calls window.SetMasterVolume, window.SetMusicVolume, window.SetSfxVolume to update the WASM engine.
UI Updates: Triggers syncUI (via global exposure) to reflect changes.
14. Public/js/particles.js - Particle System
Purpose: Manages the canvas-based particle system for visual effects (e.g., card capture sparks).
Key Imports: None.
Key Exports: initParticleSystem, animateParticles, triggerCaptureParticles, particles, particleCanvas, particleCtx, particleAnimationId.
Interaction Flow:
Initialization: app.js calls initParticleSystem on load.
Animation Loop: animateParticles is called via requestAnimationFrame.
WASM Interaction: window.PlayCaptureEffect (exposed by WASM) calls triggerCaptureParticles to create particles.
15. Public/js/utils.js - Utility Functions
Purpose: Provides general utility functions for caching and resolving blockchain-related data (asset symbols, Envoi names) and network configurations.
Key Imports: CONFIG, socket, userAddress.
Key Exports: assetCache, envoiCache, getAssetSymbol, resolveAssetSymbol, getCachedEnvoiName, resolveEnvoiName, getNetworkConfig.
Interaction Flow:
Caching: Maintains assetCache and envoiCache to reduce API calls.
Backend API Calls: Makes fetch requests to backend API endpoints (e.g., /api/asset-symbol, /api/envoi-name) for data.
Used by: Many other modules to display human-readable names for assets and wallets.

## 2A. Public-js-mermaidmap
graph TD
    subgraph Browser Environment
        HTML[Public/index.html] --> AppJS(Public/app.js)
        AppJS -- Loads & Runs --> WASM_EXEC[Public/wasm_exec.js]
        WASM_EXEC -- Provides Runtime --> WASM_ENGINE(Public/main.wasm - Go WASM Engine)
    end

    subgraph Client-Side JavaScript Modules
        AppJS -- Imports & Coordinates --> Config(Public/js/config.js)
        AppJS -- Imports & Coordinates --> Network(Public/js/network.js)
        AppJS -- Imports & Coordinates --> Wallet(Public/js/wallet.js)
        AppJS -- Imports & Coordinates --> UI(Public/js/ui.js)
        AppJS -- Imports & Coordinates --> Game(Public/js/game.js)
        AppJS -- Imports & Coordinates --> Deck(Public/js/deck.js)
        AppJS -- Imports & Coordinates --> Economy(Public/js/economy.js)
        AppJS -- Imports & Coordinates --> Criminality(Public/js/criminality.js)
        AppJS -- Imports & Coordinates --> Admin(Public/js/admin.js)
        AppJS -- Imports & Coordinates --> Leaderboard(Public/js/leaderboard.js)
        AppJS -- Imports & Coordinates --> Audio(Public/js/audio.js)
        AppJS -- Imports & Coordinates --> Particles(Public/js/particles.js)
        AppJS -- Imports & Coordinates --> Utils(Public/js/utils.js)
        AppJS -- Imports & Coordinates --> CollectiveAI(Public/collective-intelligence.js)

        Network -- Uses --> Config
        Wallet -- Uses --> Config
        UI -- Uses --> Config
        Game -- Uses --> Config
        Deck -- Uses --> Config
        Economy -- Uses --> Config
        Criminality -- Uses --> Config
        Admin -- Uses --> Config
        Leaderboard -- Uses --> Config
        Utils -- Uses --> Config

        Wallet -- Sends WS messages --> Network
        Admin -- Sends WS messages --> Network
        Game -- Sends WS messages --> Network
        Deck -- Sends WS messages --> Network
        Economy -- Sends WS messages --> Network
        Criminality -- Sends WS messages --> Network
        Leaderboard -- Sends WS messages --> Network

        Network -- Dispatches messages to --> Wallet
        Network -- Dispatches messages to --> UI
        Network -- Dispatches messages to --> Game
        Network -- Dispatches messages to --> Admin
        Network -- Dispatches messages to --> Economy
        Network -- Dispatches messages to --> Criminality
        Network -- Dispatches messages to --> Leaderboard

        Wallet -- Calls --> Utils
        Admin -- Calls --> Utils
        Economy -- Calls --> Utils
        Criminality -- Calls --> Utils
        Leaderboard -- Calls --> Utils
        Game -- Calls --> Utils

        Game -- Calls --> CollectiveAI

        UI -- Calls --> Audio
        AppJS -- Calls --> Audio

        Game -- Calls --> Particles
    end

    subgraph Go Backend
        Backend(Go Backend Server)
    end

    WASM_ENGINE -- Calls JS functions (window.syncUI, window.PlaySound) --> AppJS
    WASM_ENGINE -- Calls JS functions (window.triggerCaptureParticles) --> Particles
    AppJS -- Calls WASM functions (window.GetGameState, window.SetAvatar) --> WASM_ENGINE

    Network -- WebSocket --> Backend
    Admin -- HTTP API --> Backend
    Economy -- HTTP API --> Backend
    Criminality -- HTTP API --> Backend
    Leaderboard -- HTTP API --> Backend
    Deck -- HTTP API --> Backend
    Utils -- HTTP API --> Backend
    Wallet -- HTTP API (onboarding) --> Backend

## 3.