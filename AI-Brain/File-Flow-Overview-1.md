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

## 3 Go-File-system-flow
A1. Server and A2. Game-interaction sections of your project.

A1. Server (.go files)
The A1. Server directory contains the backend services that collectively manage the game's state, logic, and interactions with the blockchain. server.go acts as the entry point, delegating responsibilities to lobby_manager.go, which then orchestrates various specialized service files.

Here's a detailed analysis:

server.go

Hierarchy: Core/Entry Point.
Purpose: Initializes the entire server, sets up WebSocket and HTTP routes, loads configurations (like network settings and persistent card cache), and starts the main lobby event loop. It's the central orchestrator for incoming client requests.
Flow:
main() function initializes Lobby via newLobby().
Starts lobby.run() in a goroutine.
Registers HTTP handlers for various API endpoints (e.g., /ws, /api/leaderboard, /api/tournament/register, admin routes).
Serves static files from the ./Public directory.
Synergy: It's the foundation upon which the entire backend operates, connecting the client-side (via WebSockets and HTTP) to the server-side logic and services.
lobby_manager.go

Hierarchy: Core/State Manager.
Purpose: Manages the central state of the game lobby, including connected clients, active matches, matchmaking queues, player statistics (leaderboard), clubs, loans, auctions, and rumors. It handles client registration/unregistration and broadcasts global updates. It also contains periodic cleanup and processing routines.
Flow:
Receives client connections (l.register) and disconnections (l.unregister).
Processes all incoming WebSocket messages (l.broadcast) by delegating to l.handleGameProtocol().
Runs periodic tickers (cleanupNonces, processMatchmaking, checkVaultBalanceOnChain, processAuctions, processLoans, processRumors, processPlaystyleDecay, processMojoDecay, processInsuranceRecovery, processLeaseExpirations, observeGlobalSentiments, archiveSeason, refreshRegionalRoles, broadcastHealthReport, savePersistentCardCache, saveRegisteredTxIDs, saveLinkedWallets).
Calls various service functions (e.g., l.updateLeaderboard, l.incrementDNF, l.sendToClient, l.logAdminAudit).
Synergy: It's the heart of the real-time game world, maintaining consistency across all connected clients and coordinating interactions between different game systems. It ensures that all game state changes are reflected globally and persistently where needed.
common_types.go

Hierarchy: Utility/Data Structures.
Purpose: Defines all shared data structures (structs) used across the entire application, including Client, Lobby, PlayerStats, MatchState, NetworkConfig, TournamentState, Club, Loan, Auction, Rumor, KidnapState, LinkedWallet, WalletLinkInfo, ServerCard, Envelope, MoveData, UseItemData, BailCardData, NonceData, RateBucket, HoldingBonus, FaceplateStats, PlaystyleTendencies, CapturedCardInfo, MatchHistory, TournamentMatch, TournamentSummary, MetadataAttribute, ARC72Metadata.
Flow: Primarily provides definitions; does not contain active logic.
Synergy: Essential for maintaining data consistency and type safety across all Go files, both backend services and the WASM game engine. It acts as the common language for data exchange.
achievement_service.go

Hierarchy: Service.
Purpose: Manages the unlocking and notification of player achievements.
Flow:
unlockAchievement() and unlockAchievementLocked() are called by other services (e.g., battle_service.go, courthouse_service.go, club_service.go) when a player meets achievement criteria.
Updates PlayerStats.Achievements.
Calls l.logAdminAuditLocked() to record the event.
Sends admin_notification messages to the client(s) and broadcasts a lobby_update.
Synergy: Adds a meta-game layer, rewarding players for specific actions and contributing to their social standing (Reputation).
auction_service.go

Hierarchy: Service.
Purpose: Manages the Art Gallery (auction system), allowing players to list item bundles for sale, place bids, and handles auction settlement.
Flow:
handleGetAuctions() responds to HTTP requests for active auctions.
handleCreateAuction() processes requests to list items, verifies nonce, escrows items from seller's inventory, and creates a new Auction entry.
handlePlaceBid() processes bid requests, verifies nonce, deducts bid from bidder, refunds previous bidder, updates auction state, and adjusts l.faucetBalance.
processAuctions() (called by lobby_manager.go ticker) checks for expired auctions and settles them (transferring items/funds).
Uses l.ResolveEnvoiName() for display names.
Synergy: Creates a player-driven marketplace for unique item bundles, contributing to the high-finance layer of the game's economy.
battle_service.go

Hierarchy: Service.
Purpose: Implements the core game combat logic, including server-side power calculation, capture mechanics (Same, Plus, Combo), win verification, and post-match processing (jailing, item buff expiration).
Flow:
getEffectiveServerPower() calculates a card's power, accounting for player stats (Wanted Level, Cunning, Nurturing), card stats (Fatigue, Loyalty, Mood), and active item buffs.
serverCheckCaptures() simulates card captures on the server, returning flipped cards and their details.
verifyWinner() determines the match outcome, handles Sudden Death, applies bounty rewards, processes item buff expirations, and triggers jailing rules (processFallenPenaltyJailLocked, processPrisonerRuleLocked).
initiateSuddenDeath() shuffles and redistributes cards for tie-breakers.
finalizeMatchResultLocked() updates player leaderboards and triggers tournament result processing.
calculateDeckRating() and isBetterRating() are used for leaderboard metrics.
Synergy: Ensures fair and authoritative gameplay, preventing client-side cheating by re-validating all moves and outcomes on the server. It directly impacts player progression and economic consequences.
black_market_service.go

Hierarchy: Service.
Purpose: Manages the Black Market, where liquidated collateral from defaulted loans can be bought and sold by high-infamy players.
Flow:
handleGetBlackMarket() returns available liquidated loans, gated by player's Cunning and Wanted Level.
handleSellMarketTokens() allows players to convert MarketTokens (received from defaulted loans) into $VBV, applying a "scavenger tax."
handleBuyBlackMarket() allows players to purchase liquidated bundles, deducting $VBV, adding items to inventory, increasing Wanted Level, and returning proceeds to the faucet.
Synergy: Creates a high-risk, high-reward secondary market, adding depth to the criminality and high-finance layers, and contributing to the Industrial Loop.
bridge_service.go

Hierarchy: Placeholder.
Purpose: Reserved for future multi-chain bridge services.
Flow: Currently empty. Onboarding logic moved to onboarding_service.go.
Synergy: Future expansion for broader blockchain interoperability.
career.go

Hierarchy: Service.
Purpose: Manages the daily salary dispenser for club employees.
Flow:
startSalaryDispenser() runs a daily ticker.
Iterates through players, checks if they are employed, if their club has sufficient treasury, and if 24 hours have passed since the last payment.
Deducts salary from club treasury, adds to player rewards, and updates LastSalaryPayment.
Logs audit events and notifies employees.
Synergy: Reinforces the Industrial/Trust layer by automating employment benefits and creating a consistent economic flow for club members.
club_service.go

Hierarchy: Service.
Purpose: Manages club creation, joining, territory acquisition, and heist mechanics.
Flow:
handleHeist() processes player heist attempts, calculates success chance (based on Cunning, Security Level, Traps), distributes loot (with "Fence Fee" to faucet), increases Wanted Level, and handles jailing if a Guard Dog is present.
handleCreateClub() processes requests to found a new club, verifies payment, and initializes club state.
handleJoinClub() processes requests to join a club, verifies payment, adds member, and contributes to club treasury.
handlePurchaseTerritory() processes requests to acquire new territories, verifies payment, and updates club territories.
handleRestockInventory() allows authorized staff to restock club shop items, deducting from treasury.
distributeCourthouseFineToClubsLocked() distributes fines to clubs and governors.
handleCreateLease() allows members to list cards for rent.
handleTakeLease() allows members to rent cards, handling payment distribution.
processLeaseExpirations() (called by lobby_manager.go ticker) returns expired leased cards.
Synergy: Central to the Industrial/Trust and Criminality layers, enabling player organizations, economic influence, and high-stakes criminal activities.
courthouse_service.go

Hierarchy: Service.
Purpose: Allows players to reset their Wanted Level by paying a $VBV fine.
Flow:
handleCourthouseReset() processes requests, calculates fine based on Wanted Level, verifies payment via verifyBuyInTransaction().
Resets WantedLevel to 0, adds half the fine to l.faucetBalance, and distributes the other half to clubs via l.distributeCourthouseFineToClubsLocked().
Triggers achievement unlock (REHABILITATED).
Synergy: Provides a mechanism for players to manage their infamy, contributing to the Industrial Loop by redistributing fines to clubs.
economy_processing.go

Hierarchy: Service.
Purpose: Handles temporal economic processes like loan defaults and collateral liquidation.
Flow:
processLoans() (called by lobby_manager.go ticker) checks for defaulted loans.
If a loan defaults, it changes its status, calculates MarketTokens for the borrower, updates their Reputation, adds the loan to l.blackMarket, and removes it from active loans.
Synergy: Automates the consequences of financial risk, feeding into the Black Market and influencing player reputation.
economy_service.go

Hierarchy: Service.
Purpose: Manages the overall economic health of the arena, including dynamic reward scaling, season metadata persistence, and on-chain transaction notes.
Flow:
applyDynamicScaling() and applyDynamicScalingLocked() adjust reward amounts based on l.faucetBalance relative to l.maxFaucetCapacity.
saveSeasonMetadataLocked() persists season start, number, and initial rewards to season.json.
sendNoteTx() sends generic note transactions to the blockchain (used for tournament summaries, season archives).
recordWinOnChain() and recordDNFOnChain() log match outcomes on-chain.
logWinAudit() records detailed win audit logs.
CalculateReputation() computes a player's social standing based on various factors (wins, DNFs, wanted level, achievements, playstyle, employment, cosmetics).
Synergy: Central to maintaining a balanced and transparent in-game economy, ensuring rewards are sustainable and player progression is meaningful.
employment_service.go

Hierarchy: Service.
Purpose: Manages player employment within clubs, including hiring and setting salaries.
Flow:
handleHirePlayer() processes requests from club owners to hire players, updates PlayerStats.JobRole and PlayerStats.EmployerClubID, and updates Club.Staff.
handleSetSalary() allows club owners to set salaries for employees, updating PlayerStats.Salary.
Notifies employees of their new roles/salaries.
Synergy: Deepens the Industrial/Trust layer by formalizing player roles within clubs and establishing economic relationships.
faucet_service.go

Hierarchy: Service.
Purpose: Securely handles reward payouts to players using the "Switchboard Pattern" (server-side signing with client-side nonce verification).
Flow:
handleReward() receives payout requests, applies rate limiting, verifies client score against l.matchHistory, verifies client signature (EVM or Algorand) against a nonce.
verifyVoiPayoutOptIn() checks if the recipient is opted into the VBV asset.
dispatchReward() constructs and sends atomic Algorand application calls for all active reward assets, applying reputation bonuses, and handling skipped assets due to opt-in or insufficient vault balance.
Updates l.faucetBalance and triggers l.applyDynamicScalingLocked().
logWinAudit() records detailed payout information.
Synergy: Critical for the game's economy, ensuring secure and verifiable distribution of rewards while protecting the faucet's private keys.
handlers_admin.go

Hierarchy: Handler/Admin.
Purpose: Provides administrative functionalities through HTTP endpoints, protected by signature-based authentication.
Flow:
checkAdminAuth() and verifyAdminSignature() authenticate admin requests using multi-chain signatures (EVM or Algorand) against a nonce.
logAdminAudit() records all admin actions.
broadcastToAdmins() sends messages to connected admin clients.
Handles various admin actions: handleRefillVault, handleUpdateRules, handleAdminAddReward, handleAdminRemoveReward, handleSetActiveNetwork, handleAddNetwork, handleUpdatePowerScaling, handleSystemMessage, handleBanPlayer, handleGloatBan, handleAvatarBan, handleResetStats, handleUpdateBaseReward, handleMaintenanceMode, handleUpdateRewardAsset, handleStartTournament, handleOpenRegistration, handleSimulateTournament, handleGetAdminLogs.
Synergy: Essential for game management, balancing, and moderation, ensuring the platform can be maintained and adapted by authorized personnel.
handlers_criminality.go

Hierarchy: Handler/Criminality.
Purpose: Manages criminal actions like kidnapping gambits, ransom payments, and bailing jailed cards.
Flow:
handleKidnapRequest() processes requests to kidnap a card (favorite or rarest), removes it from victim's inventory, adds to perp's KidnappedCards and victim's HeldHostageCards, and sets an expiration for InsuranceRecovery.
handlePayRansom() processes ransom payments, verifies funds, deducts from victim, distributes to perp (with "Laundering Tax" to faucet), returns card to victim, and removes from tracking.
handleReleaseHostage() allows a kidnapper to voluntarily release a card.
handleBailCard() allows players to pay a fine to release a jailed card, verifying payment and distributing to the jailing club.
processInsuranceRecovery() (called by lobby_manager.go ticker) automatically returns expired kidnapped cards.
Synergy: Drives the high-stakes criminality layer, creating dynamic player interactions and economic consequences.
handlers_public.go

Hierarchy: Handler/Public API.
Purpose: Provides public-facing HTTP endpoints for game statistics and information, accessible by external services or non-authenticated clients.
Flow:
handleLeaderboard() returns sorted player statistics.
handlePublicStatus() provides general server status (faucet balance, active matches, maintenance mode).
handleHealthCheck() returns a simple "ok" status.
handleCardStats() and handleGetCardDetails() retrieve verified NFT metadata using l.getVerifiedCards().
handleReSyncStats() triggers a manual blockchain sync for player stats.
Synergy: Enables transparency and external integration, allowing the game's status and player achievements to be showcased outside the application.
handlers_rumor.go

Hierarchy: Handler/Rumor System.
Purpose: Manages the spreading of rumors about players, influencing their market value.
Flow:
handleSpreadRumor() processes requests to spread a rumor, deducts cost from spreader's rewards, updates RumorCount, creates a Rumor object with an expiration, and broadcasts the update.
processRumors() (called by lobby_manager.go ticker) removes expired rumors.
Synergy: Introduces market manipulation mechanics, adding depth to the high-finance layer and social interactions.
item_service.go

Hierarchy: Service.
Purpose: Centralizes the logic for applying item effects, both in-match and persistent.
Flow:
applyItemEffect() is called by lobby_manager.go:handleGameProtocol (specifically the use_item case).
Applies effects based on item.ClubType:
Vitality items (e.g., stamina_stim, loyalty_pledge) modify ServerCard stats (Fatigue, Loyalty) and l.persistentCardCache.
Elemental/Tactical items (e.g., mood_catalyst, grounded_shield, rule_breaker, intel_report) modify MatchState.ActiveItemBuffs and potentially MatchState.Rules.
Hardware items (e.g., tripwire, sentry_turret, guard_dog) are deployed as Club.ActiveBuffs with expirations.
Updates PlayerStats.Playstyle.PreferredItems.
Synergy: Provides a modular way to manage the diverse effects of in-game items, integrating them into combat, club defense, and player progression.
loan_service.go

Hierarchy: Service.
Purpose: Manages the Second-Hand Store (loan system), allowing players to take collateralized loans and repay them.
Flow:
handleGetLoans() returns active loans, optionally filtered by borrower.
handleTakeLoan() processes loan requests, verifies nonce, checks faucet liquidity, escrows collateral from player's inventory, creates a Loan object, deducts principal from l.faucetBalance, and adds to player's rewards.
handleRepayLoan() processes loan repayments, verifies payment transaction, adds repayment (principal + interest) to l.faucetBalance, returns collateral to player, and deletes the loan.
Uses l.ResolveEnvoiName() for display names.
Synergy: Introduces a credit market, enabling players to manage their liquidity at the risk of losing collateral, feeding into the Black Market.
market_service.go

Hierarchy: Service.
Purpose: Manages the Entity Market (share trading) and observes global player sentiments.
Flow:
handleTradeShares() processes buy/sell requests for player/NPC shares, calculates price based on player stats (Wins, Reputation, Rumors), deducts/adds $VBV from player rewards, and adjusts l.faucetBalance (Industrial Loop).
observeGlobalSentiments() (called by lobby_manager.go ticker) aggregates player playstyle data (Aggressiveness, Risk Tolerance, Preferred Rules) to identify meta-trends.
generateNPCCommentary() provides narrative hooks based on player style and global sentiments.
Synergy: Creates a dynamic, player-driven stock market and enhances narrative immersion through AI-driven commentary.
onboarding_service.go

Hierarchy: Service.
Purpose: Provides a Sybil-protected "Starter Pack" to Algorand users to bridge them to Voi.
Flow:
handleVoiOnboarding() processes onboarding requests, checks l.SybilSyncComplete, uses a per-wallet lock and global semaphore to prevent abuse.
Checks l.onboardedWallets for historical claims.
Checks if the recipient already has native VOI.
Atomically decrements l.faucetBalance.
Dispatches a grouped transaction (1 VOI + 1 VBV) to the recipient.
Marks the wallet as onboarded in l.onboardedWallets.
Synergy: Facilitates new player adoption by providing initial resources while implementing robust Sybil protection.
oracle_service.go

Hierarchy: Service/Blockchain Interaction.
Purpose: Acts as the primary interface for reading authenticated data from various blockchain indexers and nodes, caching NFT metadata, and reconstructing game state.
Flow:
getVerifiedCards() and getVerifiedCardsCrossChain() fetch and cache NFT metadata (ARC-72, EVM, Solana) from configured indexers/nodes, applying power scaling.
syncStatsFromBlockchain() and refreshGlobalLeaderboard() reconstruct player wins/DNFs from on-chain transfer metadata.
loadOnboardedWalletsFromIndexer() reconstructs historical Sybil protection state by scanning for past onboarding transactions.
ResolveEnvoiName() resolves wallet addresses to human-readable names (e.g., .voi names) using a local cache.
verifyBuyInTransaction() verifies payment transactions on Voi (ARC-200) or Algorand (ASA).
checkVaultBalanceOnChain() and checkNativeVaultBalanceOnChain() synchronize internal faucet balances with on-chain pools.
savePersistentCardCache() persists the card cache to disk.
handleSeasonHistory() fetches archived seasonal standings from the blockchain.
handleReSyncStats() triggers a manual sync for a specific wallet.
mapChainToNetworkName() translates chain codes.
checkAssetOptIn() verifies if a wallet is opted into a specific asset.
Synergy: Crucial for the game's decentralized nature, ensuring that all critical game data is verifiable on-chain and that the server's internal state remains synchronized with the blockchain. It also enables cross-chain NFT integration.
shop_registry.go

Hierarchy: Data/Configuration.
Purpose: Defines the static registry of all purchasable shop items, including their properties like price, club type, description, and heist modifiers.
Flow: GlobalShopRegistry is a global map accessed by services like item_service.go and club_service.go.
Synergy: Provides a centralized and consistent source of truth for all in-game items, enabling various game mechanics to interact with them.
tournament_manager.go

Hierarchy: Service.
Purpose: Manages the lifecycle of automated tournaments, from registration to finalization and on-chain archiving.
Flow:
handleTournamentRegister() processes player registrations, verifies eligibility (elite status or buy-in payment), and adds participants.
handleTournamentHistory() fetches archived tournament summaries from the blockchain.
processTournamentResult() updates match winners, checks for round completion, and triggers advanceTournamentRound().
advanceTournamentRound() progresses the tournament bracket, creating new matches or finalizing the tournament.
determineTop5() identifies the top-ranked players based on bracket progression.
finalizeTournament() calculates payouts, dispatches multi-asset rewards via dispatchTournamentRewards(), and records the tournament summary on-chain via recordTournamentOnChain().
dispatchTournamentRewards() constructs and sends atomic Algorand application calls for tournament payouts, applying granular opt-in checks.
broadcastTournamentState() sends real-time updates to clients.
isWalletRegistered() checks if a wallet is already registered.
Synergy: Drives competitive gameplay, creates high-stakes events, and ensures transparent, verifiable results through blockchain archiving.
A2. Game-interaction (main.go)
main.go
Hierarchy: Client-side Game Engine (WASM).
Purpose: Implements the core game logic that runs in the browser via WebAssembly. This includes the game board, card mechanics, AI decision-making, player state, and UI synchronization. It exposes functions to JavaScript for interaction.
Flow:
Initializes the Game (Engine) struct, which holds the entire client-side game state.
registerFunctions() exposes Go functions to the JavaScript global scope (e.g., connectWallet, PlaceCard, GetGameState, SyncTournament).
connectWallet(), disconnectWallet(), toggleNetwork(), SetAvatar(), SendReward() handle basic client-side wallet and identity management.
ToggleRule(), SelectDeck(), RemoveFromDeck(), AddToDeck(), SetPlayerReady(), AutoBuildDeck() manage deck building and lobby readiness.
SyncPlayerStats(), SyncFullProfile(), SyncPortfolio(), SyncPlaystyle(), SyncOpponentProfile(), SyncOpponentDeck(), SyncOpponentWanted(), SyncVaultBalance(), SyncRewards(), SyncRules(), SyncServerLoad(), SyncLatency(), SyncTournament() receive state updates from the backend.
StartMatch(), PlaceCard(), PerformAIMove(), checkCaptures(), flipCard(), simulateCaptures(), simulateCapturesOnBoard(), calculateMaxPlayerPotential(), checkWinCondition(), initiateSuddenDeath() implement the core battle logic.
GetGameState() provides a snapshot of the client-side state to JavaScript for UI rendering.
SetPhase(), SetTestingMode(), SetHardMode(), SetAdminState(), SetMaintenanceState(), SetLocalPlayerIndex() allow JavaScript to control game engine settings.
UpdateAmbientMusic(), PlayAmbient(), StopAmbient(), PlaySound(), SetMasterVolume(), SetMusicVolume(), SetSfxVolume(), SetAssetBase(), SetApiBase() manage audio and asset loading.
GetTierInfo(), GetLevelLabelForDisplay(), calculateDeckRating(), isBetterRating(), calculateLoadColor(), GetServerLoadColor(), ToggleLeaderboard(), GetTournamentArchiveBadge() provide UI-specific data.
ImportARC72Card() fetches card details from the backend.
ApplyArtifactToBoard() applies item effects to cards on the board.
Synergy: Provides the client-side game engine, ensuring deterministic and responsive gameplay. It offloads complex calculations from the server, reduces latency, and allows for rich interactive experiences directly in the browser, while still relying on the backend for authoritative state synchronization and blockchain interactions.

## 4. Go Backend-mermaidmap
graph TD
    subgraph Entry & Routing
        SVR(server.go)
    end

    subgraph Core Orchestration
        LOBBY(lobby_manager.go)
        TYPES(common_types.go)
        REG(shop_registry.go)
    end

    subgraph Battle Systems
        BATTLE(battle_service.go)
        ITEM(item_service.go)
        ACHIEVE(achievement_service.go)
    end

    subgraph Industrial Economy
        ECON_S(economy_service.go)
        FAUCET(faucet_service.go)
        MARKET(market_service.go)
        AUCTION(auction_service.go)
        LOAN(loan_service.go)
    end

    subgraph Criminality & Social
        CLUB(club_service.go)
        CRIM(handlers_criminality.go)
        RUMOR(handlers_rumor.go)
        COURT(courthouse_service.go)
    end

    subgraph Infrastructure
        ORACLE(oracle_service.go)
        ADMIN(handlers_admin.go)
        TOURN(tournament_manager.go)
    end

    subgraph Client Determinism
        WASM(main.go - WASM)
    end

    SVR -- Initializes --> LOBBY
    SVR -- API Routes --> ADMIN
    SVR -- API Routes --> FAUCET
    
    LOBBY -- State Loop --> BATTLE
    LOBBY -- State Loop --> TOURN
    LOBBY -- WS Switchboard --> CLUB
    LOBBY -- WS Switchboard --> CRIM
    
    BATTLE -- Deterministic Sync --> WASM
    BATTLE -- Power Penalties --> TYPES
    
    ECON_S -- Dynamic Scaling --> FAUCET
    FAUCET -- Signature Auth --> ORACLE
    
    CLUB -- Revenue --> ECON_S
    CRIM -- Fines --> COURT
    COURT -- Redistribution --> CLUB
    
    ORACLE -- Indexer Data --> TOURN
    ORACLE -- NFT Metadata --> TYPES
    

## 5. UI-File-sys-Flow

The UI of Virtualbabes Arena is built with a strong emphasis on a "neon-glass" aesthetic, dynamic content, and responsiveness. It leverages a combination of static assets (images, videos), structural HTML, and a highly modular SCSS architecture to deliver an immersive user experience.

#### A: Card_images (`Public\Assets\Images\Cards\*.webp`)

*   **Purpose**: These `.webp` image files serve as the visual representation of the collectible game cards. Each file corresponds to a unique card character in the game, displaying their artwork.
*   **Flow**: These images are loaded by the browser as `<img>` tags or as `background-image` properties for HTML elements. Their specific paths are determined dynamically by the client-side JavaScript (`deck.js`, `game.js`) based on card IDs or metadata received from the Go WASM engine or the backend.
*   **Hierarchy**: These are low-level static visual assets, forming the core visual identity of the game's primary interactive elements (the cards).
*   **Synergy**:
    *   **`Public/app.js`, `Public/js/game.js`, `Public/js/deck.js`**: These JavaScript modules are responsible for dynamically creating and updating the HTML elements that display these card images (e.g., in the player's hand, on the game board, or in the deck manager). They construct the image `src` attributes using these file paths.
    *   **`Public/main.wasm` (Go WASM Engine)**: The WASM engine holds the game state, including which cards are in a player's hand or on the board. It provides card metadata (like card ID) to JavaScript, which then maps to the correct image file.
    *   **`Public/src/scss/components/_cards.scss`**: This SCSS file defines the visual styling for how these card images are presented, including their dimensions, borders, shadows, and animations (e.g., `.playing-card`, `.card-mini`).
    *   **Backend (`oracle_service.go`)**: Fetches and provides metadata for these cards (including their image URLs) from blockchain indexers, ensuring that the client displays authenticated assets.

#### B: Fan_fare_Avatars (`Public\Assets\Images\portraits\*\*.mp4`, `*.webp`, `*.png`)

*   **Purpose**: This collection provides visual assets for player avatars and NPC portraits. It includes both static (`.webp`, `.png`) and animated (`.mp4`) formats to offer dynamic and expressive character representations. The different subdirectories (`Boss`, `cute`, `Lady`, `Mini-Boss`, `Witch`) categorize avatars by character type or role.
*   **Flow**:
    *   Static images (`.webp`, `.png`): Loaded into `<img>` tags for display in various UI components (e.g., player profiles, leaderboards).
    *   Animated videos (`.mp4`): Loaded into `<video>` tags, typically configured for looping and autoplay, to provide dynamic flair for key characters or player selections.
    *   The `deck.js` module specifically handles the selection, preview, and cropping of these avatars during player setup.
*   **Hierarchy**: These are static/animated visual assets representing player and NPC identities, used across various UI screens.
*   **Synergy**:
    *   **`Public/app.js`, `Public/js/ui.js`, `Public/js/deck.js`**: These modules dynamically render avatars in elements like `#p1-avatar`, `#p2-avatar`, and the avatar selection grid in the setup overlay. `deck.js` manages the interactive cropping and selection process, potentially sending the chosen avatar URL to the backend.
    *   **`Public/main.wasm` (Go WASM Engine)**: Stores the player's selected avatar URL as part of their profile, which JavaScript retrieves for display.
    *   **`Public/src/scss/layouts/_dashboard.scss` (specifically `.avatar-frame`)**: Styles the display of avatars, including their circular frames, borders, and sizes.
    *   **Backend (`oracle_service.go`, `deck.go` - if avatar registration is a backend call)**: Stores and retrieves the selected avatar URLs, and may handle the storage of custom-cropped avatars.

#### C: Textures (`Public\Assets\Textures\*.png`)

*   **Purpose**: These `.png` files provide background textures for the game arena, allowing the visual theme of the battleground to change dynamically based on the match context (e.g., standard, challenge, tournament).
*   **Flow**: These images are typically set as `background-image` properties for specific HTML elements (e.g., the game board container). The choice of texture is dynamic, based on the current game mode.
*   **Hierarchy**: Background visual assets, providing environmental context.
*   **Synergy**:
    *   **`Public/app.js`, `Public/js/ui.js`**: The `ui.js` module (specifically the `updateDynamicArenaFloor` function, as indicated by its import in `ui.js`) is responsible for dynamically changing the `background-image` of the game board element in `index.html` based on the match type.
    *   **`Public/src/scss/layouts/_dashboard.scss`**: May define base styling for the arena floor element, which is then overridden or augmented by JavaScript to apply specific textures.

#### D: UI_filesys

This category encompasses the core structural and styling files that define the entire frontend user interface.

*   **`Public\index.html`**
    *   **Purpose**: This is the single entry point for the Virtualbabes Arena web application. It defines the fundamental HTML structure, loads all essential scripts (WASM runtime, main JavaScript application, blockchain SDKs), and links the primary stylesheet. It contains static UI elements and placeholders (`div`s with IDs) where dynamic content will be injected by JavaScript.
    *   **Flow**: The browser first loads this file. It then sequentially loads `styles.css`, `wasm_exec.js`, `app.js`, and various external SDKs (Buffer, Algorand SDK, WalletConnect). It also contains inline `onclick` event handlers that trigger functions defined in `app.js`.
    *   **Hierarchy**: The root of the entire client-side application's DOM structure. All other UI components and scripts are loaded into or interact with elements defined here.
    *   **Synergy**:
        *   **`Public/app.js`**: The primary JavaScript file that manipulates the DOM elements defined in `index.html`. It populates dynamic content, attaches event listeners, and controls the visibility of various sections and overlays.
        *   **`Public/styles.css`**: Provides the visual styling for all elements within `index.html`.
        *   **`Public/wasm_exec.js`**: Loaded by `index.html` to enable the execution of the Go WASM game engine.
        *   **`Public/js/wallet.js`, `Public/js/leaderboard.js`, `Public/js/deck.js`, `Public/js/economy.js`, `Public/js/criminality.js`, `Public/js/admin.js`, `Public/js/ui.js`**: These modules contain functions that directly interact with specific DOM elements (e.g., buttons, input fields, display areas) defined in `index.html` to render data, handle user input, and manage UI state.
        *   **`Public/Assets/Images/portraits/*.svg` (inline in `wallet-selector-overlay`)**: The SVG data for wallet icons is directly embedded in the HTML, providing immediate visual feedback for wallet options.

*   **`Public\styles.css`**
    *   **Purpose**: This is the compiled CSS file that applies all the visual styling to the `index.html`. It's generated from the SCSS source files.
    *   **Flow**: Loaded by `index.html` early in the page load process, ensuring that styles are applied before JavaScript renders dynamic content.
    *   **Hierarchy**: The final output of the SCSS pre-processing, directly consumed by the browser.
    *   **Synergy**:
        *   **`Public/index.html`**: The target for all its styles.
        *   **`Public/src/scss/*.scss`**: Its source code. Any changes to SCSS files are compiled into this `styles.css`.
        *   **`Public/app.js`, `Public/js/ui.js`**: These JavaScript files might dynamically add or remove CSS classes (e.g., `hidden`, `active`, `error`, `success`) to HTML elements, which then trigger styles defined in `styles.css`.

*   **`Public\src\scss\main.scss`**
    *   **Purpose**: This is the main entry point for the SCSS compilation process. It imports all other SCSS partials, organizing them into a logical structure.
    *   **Flow**: A SCSS pre-processor reads `main.scss`, resolves all `@import` statements, and compiles the entire stylesheet into a single `Public/styles.css` file.
    *   **Hierarchy**: The root of the SCSS architecture, defining the order in which styles are processed.
    *   **Synergy**:
        *   **All other `Public/src/scss/*.scss` files**: It imports them, bringing all styling rules together.
        *   **Build process (e.g., `npm run build-css` or similar script)**: This file is the input for the SCSS compiler.

*   **`Public\src\scss\base\_reset.scss`**
    *   **Purpose**: Provides a CSS reset to ensure consistent styling across different browsers and establishes fundamental base styles for common HTML elements (e.g., `body`, `h1-h6`, `p`, `a`, `ul`, `ol`, `button`, `img`, `table`). It also defines custom scrollbar styles for Webkit browsers.
    *   **Flow**: Imported by `main.scss` early in the compilation process to apply foundational styles before more specific component or layout styles.
    *   **Hierarchy**: Base-level styling, affecting the entire document.
    *   **Synergy**:
        *   **`Public/index.html`**: Sets the default appearance for all raw HTML elements.
        *   **`Public/src/scss/base/_variables.scss`**: Utilizes variables like `$font-body`, `$color-text-main`, `$spacing-md`, `$border-radius-md` for consistent theming.

*   **`Public\src\scss\base\_typography.scss`**
    *   **Purpose**: Defines specific typographic styles, including font families, sizes, weights, colors, and text transformations, with a focus on the "neon-glass" aesthetic. It includes utility classes for common text styles and responsive adjustments.
    *   **Flow**: Imported by `main.scss` after `_reset.scss` to apply specific text styling rules.
    *   **Hierarchy**: Base-level styling, focusing on text presentation.
    *   **Synergy**:
        *   **`Public/index.html`**: Styles headings, paragraphs, and other text content.
        *   **`Public/src/scss/base/_variables.scss`**: Heavily relies on variables for `$font-heading`, `$font-body`, `$color-neon-cyan`, `$font-size-xl`, etc.
        *   **`Public/app.js`, `Public/js/ui.js`**: JavaScript might dynamically add utility classes (e.g., `text-neon-green`, `font-bold`) to text elements.

*   **`Public\src\scss\base\_variables.scss`**
    *   **Purpose**: Acts as the central repository for all design tokens and global constants, including color palettes, font definitions, spacing scales, border radii, shadows, z-indices, transitions, and breakpoints. It also defines component-specific variables like avatar and card sizes.
    *   **Flow**: Imported by almost all other SCSS files. It must be imported first within any file that uses its variables.
    *   **Hierarchy**: The absolute foundation of the visual design system.
    *   **Synergy**:
        *   **All other `Public/src/scss/*.scss` files**: Provides consistent values for styling throughout the application.
        *   **`Public/app.js`**: Dynamically sets CSS variables like `--arena-mood-color` based on game state, which are then used in SCSS.

*   **`Public\src\scss\components\_buttons.scss`**
    *   **Purpose**: Defines the styling for all interactive buttons in the application, including base styles, primary glowing buttons, outline buttons, secondary buttons, and specific styles for success, danger, and warning actions. It also includes size variants and styles for wallet connection options.
    *   **Flow**: Imported by `main.scss` to apply styling to button elements.
    *   **Hierarchy**: Component-level styling, specific to buttons.
    *   **Synergy**:
        *   **`Public/index.html`**: Buttons defined in the HTML (`<button>`) will automatically pick up these styles.
        *   **`Public/app.js`, `Public/js/ui.js`, `Public/js/wallet.js`**: JavaScript controls button states (e.g., `disabled`, adding/removing classes like `active`, `loading`) which are styled here.
        *   **`Public/src/scss/base/_variables.scss`**: Uses color variables (`$color-neon-purple`, `$color-error-red`), spacing (`$spacing-md`), and border radii (`$border-radius-md`).
        *   **`Public/src/scss/themes/_neon-glass.scss`**: The `.wallet-option` uses the `neon-glass-panel` mixin.

*   **`Public\src\scss\components\_cards.scss`**
    *   **Purpose**: Styles the visual presentation of game cards, including their dimensions, appearance, rarity indicators, type icons, debuff badges, and interactive states (hover, selected, disabled). It also defines styles for card grids and tooltips.
    *   **Flow**: Imported by `main.scss` to style card elements.
    *   **Hierarchy**: Component-level styling, specific to game cards.
    *   **Synergy**:
        *   **`Public/app.js`, `Public/js/game.js`, `Public/js/deck.js`**: These JavaScript modules dynamically generate the HTML for cards using `renderCardHTML` and apply classes (e.g., `selected-card`, `disabled`, `common`, `rare`, `epic`, `legendary`) that are styled here.
        *   **`Public/main.wasm` (Go WASM Engine)**: Provides card data (power, rarity, mood, artifact) that JavaScript uses to determine which classes to apply.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$card-width`, `$card-height`, `$color-neon-cyan`, `$glass-blur`, etc.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines keyframe animations (`card-enter`, `card-exit`, `card-flip`) used for card transitions.

*   **`Public\src\scss\components\_overlays.scss`**
    *   **Purpose**: Provides generic and specific styling for all modal overlays and pop-up windows in the application (e.g., settings, wallet selector, deck manager, admin panel, tournament bracket, match preview, kidnap gambit, Hall of Fame). It includes base overlay styles, content containers, headers, bodies, and footers.
    *   **Flow**: Imported by `main.scss` to style overlay elements.
    *   **Hierarchy**: Component-level styling, specific to overlays.
    *   **Synergy**:
        *   **`Public/index.html`**: Defines the base HTML structure for all overlays (e.g., `<div class="overlay hidden">`).
        *   **`Public/app.js`, `Public/js/ui.js`, and other feature-specific JS files**: JavaScript controls the visibility of these overlays by adding/removing the `hidden` class. It also dynamically populates their content.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$z-index-modal`, `$glass-blur`, `$spacing-lg`, `$border-radius-xl`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Heavily uses the `neon-glass-panel` mixin for the distinctive UI aesthetic.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines animations like `animate-modal` for overlay transitions.

*   **`Public\src\scss\features\_criminality.scss`**
    *   **Purpose**: Styles the UI elements related to the game's criminality features, such as the criminality panel, heist actions, target selection, risk assessment, and results display. It emphasizes a red/orange color palette to convey danger and warning.
    *   **Flow**: Imported by `main.scss` to style criminality-specific UI.
    *   **Hierarchy**: Feature-specific styling.
    *   **Synergy**:
        *   **`Public/js/criminality.js`**: This JavaScript module dynamically generates and manipulates the HTML for criminality features, applying classes and IDs that are styled here.
        *   **`Public/app.js`**: Calls functions in `criminality.js` to open and manage these overlays.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$color-error-red`, `$color-warning-orange`, `$color-neon-purple`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines animations like `progress-shine` and `risk-pulse`.

*   **`Public\src\scss\features\_economy.scss`**
    *   **Purpose**: Styles the UI elements for economic features, including the economy panel, market ticker, auction gallery, second-hand store (loans), and black market. It uses green and cyan tones to represent wealth and digital interfaces.
    *   **Flow**: Imported by `main.scss` to style economy-specific UI.
    *   **Hierarchy**: Feature-specific styling.
    *   **Synergy**:
        *   **`Public/js/economy.js`**: This JavaScript module dynamically generates and manipulates the HTML for economic features, applying classes and IDs that are styled here.
        *   **`Public/app.js`**: Calls functions in `economy.js` to open and manage these overlays.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$color-neon-green`, `$color-neon-cyan`, `$color-neon-purple`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines animations like `ticker-scroll`.

*   **`Public\src\scss\features\_shops.scss`**
    *   **Purpose**: Styles the UI for district shops, including shop panels, categories, item grids, filters, shopping cart, and special offers. It often uses purple and cyan tones.
    *   **Flow**: Imported by `main.scss` to style shop-specific UI.
    *   **Hierarchy**: Feature-specific styling.
    *   **Synergy**:
        *   **`Public/js/economy.js`**: This JavaScript module (specifically `openShopsOverlay`, `switchShopCategory`, `buyClubItem`) dynamically generates and manipulates the HTML for shop features, applying classes and IDs that are styled here.
        *   **`Public/app.js`**: Calls functions in `economy.js` to open and manage these overlays.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$color-neon-purple`, `$color-neon-cyan`, `$color-gold`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin.

*   **`Public\src\scss\features\_social.scss`**
    *   **Purpose**: Styles the UI for social features, including the social panel, achievement system (trophies), career paths, entity portfolio, and social network connections. It uses a mix of gold, blue, and pink tones.
    *   **Flow**: Imported by `main.scss` to style social-specific UI.
    *   **Hierarchy**: Feature-specific styling.
    *   **Synergy**:
        *   **`Public/js/criminality.js` (for `openSocialPanelOverlay`, `switchSocialTab`)**: This JavaScript module dynamically generates and manipulates the HTML for social features, applying classes and IDs that are styled here.
        *   **`Public/app.js`**: Calls functions in `criminality.js` to open and manage these overlays.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$color-gold`, `$color-neon-blue`, `$color-neon-pink`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines animations like `badge-glow`.

*   **`Public\src\scss\features\_territory.scss`**
    *   **Purpose**: Styles the UI for territory management, including the territory panel, 3D world map, districts, club foundry, regional governor status, and conflicts. It uses purple, cyan, and blue tones for a futuristic, strategic feel.
    *   **Flow**: Imported by `main.scss` to style territory-specific UI.
    *   **Hierarchy**: Feature-specific styling.
    *   **Synergy**:
        *   **`Public/app.js` (for `openTerritoryMapOverlay`, `adjustMapZoom`)**: This JavaScript module dynamically generates and manipulates the HTML for territory features, applying classes and IDs that are styled here.
        *   **`Public/js/economy.js` (for `openClubFoundry`, `submitClubFoundry`, `openTerritoryView`)**: These functions interact with the territory UI.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$color-neon-purple`, `$color-neon-cyan`, `$color-neon-blue`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines animations like `contested-pulse`.

*   **`Public\src\scss\layouts\_dashboard.scss`**
    *   **Purpose**: Defines the layout and styling for the main game dashboard, including the overall container, columns, player lists, chat interface, matchmaking box, cooldown displays, tournament banners, action bars, and match history.
    *   **Flow**: Imported by `main.scss` to structure the primary game screen.
    *   **Hierarchy**: Layout-level styling, organizing major UI sections.
    *   **Synergy**:
        *   **`Public/index.html`**: Provides the structural `div`s (e.g., `.dashboard`, `.column`) that these styles target.
        *   **`Public/app.js`, `Public/js/game.js`, `Public/js/ui.js`**: JavaScript dynamically populates content within these layout elements (e.g., player names, chat messages, history items) and controls their visibility.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$spacing-lg`, `$border-radius-lg`, `$color-neon-cyan`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin to various dashboard elements.

*   **`Public\src\scss\layouts\_main-layout.scss`**
    *   **Purpose**: Defines the overarching layout for the entire application, including the main game container and the top navigation bar.
    *   **Flow**: Imported by `main.scss` to establish the highest-level structural styles.
    *   **Hierarchy**: Global layout styling.
    *   **Synergy**:
        *   **`Public/index.html`**: Provides the root layout elements (`.main-game-container`, `.top-bar`).
        *   **`Public/app.js`, `Public/js/ui.js`**: JavaScript controls elements within this layout (e.g., wallet connection button, maintenance bar visibility).
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$spacing-md`, `$glass-border-color`, `$glass-blur`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin to elements like status widgets.

*   **`Public\src\scss\themes\_neon-glass.scss`**
    *   **Purpose**: Defines the core "neon-glass" aesthetic through a reusable mixin (`neon-glass-panel`) and applies it to base elements. It also includes a mixin for neon text glow.
    *   **Flow**: Imported by `main.scss` and then explicitly `@include`d by other component and feature SCSS files to apply the glassmorphism effect.
    *   **Hierarchy**: Thematic styling, providing a consistent visual language.
    *   **Synergy**:
        *   **`Public/src/scss/base/_variables.scss`**: Relies entirely on variables like `$glass-bg-color`, `$glass-border-color`, `$glass-blur`, `$glass-shadow` to define the glassmorphism properties.
        *   **All other component/feature SCSS files**: Consumes the `neon-glass-panel` mixin to apply the theme.

*   **`Public\src\scss\utilities\_animations.scss`**
    *   **Purpose**: Provides a comprehensive set of CSS animations and keyframe definitions for various UI effects (fade, slide, scale, bounce, pulse, glow, shimmer, float, spin). It also includes utility classes for applying these animations and controlling their properties (duration, delay, fill mode).
    *   **Flow**: Imported by `main.scss` to make animation classes available globally.
    *   **Hierarchy**: Utility-level styling, providing reusable animation effects.
    *   **Synergy**:
        *   **`Public/app.js`, `Public/js/ui.js`, `Public/js/particles.js`**: JavaScript dynamically adds/removes animation classes to trigger visual effects (e.g., `animate-modal`, `animate-capture-burst`).
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$transition-base`, `$color-neon-cyan`, etc. for animation properties.

*   **`Public\src\scss\utilities\_spacing.scss`**
    *   **Purpose**: Provides a utility-first approach for common layout and spacing properties, including display types, flexbox, grid, margin, padding, gap, width, height, position, z-index, overflow, text alignment, font properties, opacity, visibility, borders, shadows, cursors, object-fit, transforms, and transitions. It also includes responsive utilities.
    *   **Flow**: Imported by `main.scss` to provide a wide range of atomic utility classes.
    *   **Hierarchy**: Utility-level styling, offering granular control over layout and appearance.
    *   **Synergy**:
        *   **`Public/index.html`**: HTML elements are directly annotated with these utility classes (e.g., `flex-row`, `gap-15`, `mb-20`, `w-full`) to define their layout and spacing.
        *   **`Public/src/scss/base/_variables.scss`**: Relies on the `$spacing-scale` and other variables for consistent sizing.