// PILLAR 5: Modular Orchestration.
// app.js acts as the central hub, importing authoritative domain logic
// and exposing it to the global window scope for index.html accessibility.

import { CONFIG } from './js/config.js';
import './js/portfolio.js'; // PORTFOLIO — read-only analytics surface (binds window.openPortfolio)
import { initWebSocket, handleServerMessage, sendPing, requestMatchSync } from './js/network.js';
import { 
    hideAllOverlays, updateDynamicArenaFloor, renderCardHTML, syncBoardParticles, 
    showToast, setTransactionStatus, openSettingsOverlay, closeSettingsOverlay, 
    showTournamentTransition, shareTournamentVictory, showQuickCastMenu, openTerritoryMapOverlay,
    updateTournamentPaginationUI, openTerritoryView, updateSpectatorHUD, adjustMapZoom, 
    updateMapStatusIndicators, handleMaintenanceUI, handleTournamentUI, renderTournamentBracket, 
    openTournamentBracket, closeTournamentBracket, switchHofTab, toggleTournamentDetails, showMutationStabilityTooltip, hidePowerTooltip,
    updateAvatarIdentityStyle, updateMoodCatalystVisuals, updateStaffTrainingVisuals
} from './js/ui.js';
import { initWalletConnect, handleWalletAction, updateWalletUI, openPayoutSettings, savePayoutAddress, userAddress, connectWith, updatePayoutUI, closeWalletSelector, addXChainWallet, submitLinkWallet } from './js/wallet.js';
import { fetchLeaderboard, fetchTournamentHistory, fetchSeasonHistory, filterSeasonHistory } from './js/leaderboard.js';
import { buildEmptyBoard, toggleMatchmakingQueue, sendChatMessage, handleChatKey, proceedToWarRoom, sendChallenge, selectCard, clickGrid, executeQuickCast, acceptChallenge, declineChallenge, triggerToggleNetwork, sendSpectate, showPowerTooltip, rejoinActiveMatch, setPendingQuickCastId, spectatorMatchState } from './js/game.js';
import { openDeckManager, closeDeckManager, renderDeckManager, setupCropEvents, applyAvatarFilters, selectAvatar, refreshInventory, renderAvatarGrid } from './js/deck.js';
import { 
    globalClubs,
    adminAddReward, adminRemoveReward, adminAddNetwork, 
    adminBroadcast, adminUpdateRules, adminBanWallet, adminUpdatePowerScaling, adminSimulateMutationSuccess, adminSimulateMutationFailure,
    adminToggleMaintenance, adminToggleDevMode, adminResetStats, adminSimulateTournament, adminAssetForfeiture, adminForcePayout, adminSimulateMojoDecay, adminCyberSecurityAudit,
    onAdminNetworkSelectChange, adminSetActiveNetwork, adminSeasonRollover, adminExportAuditLog, adminCommissionAudit, adminTaxAudit, adminRestockDLC, adminDistrictTaxAudit,
    updateAdminRewardRegistry, toMicro
} from './js/admin.js'; // Note: adminDistrictTaxAudit should be added here
import { openShopsOverlay, buyClubItem, openClubFoundry, submitClubFoundry, openArtGalleryOverlay, openConsignmentOverlay, selectConsignmentItem, submitConsignment, promptBid, openPortfolioView, tradeShares, openBlackMarket, buyBlackMarketItem, openClubLeaseBoard, switchPortfolioTab, takeLease, openCreateLeaseOverlay, submitCreateLease, openMutationHistoryOverlay, openVaultInteraction, submitDistrictTax, openRecoveryBountyOverlay, adjustMutationVector, submitVectorRealignment, submitMoodRecalibration, submitMutationLoyaltySynthesis, submitBid, switchFoundryTab, switchShopCategory } from './js/economy.js';
import { openCourthouse, submitCourthouseFine, initiateBail, openSecuritySentry, deployTrap, openBountyBoard, openRumorMill, spreadRumor, openSocialPanelOverlay, switchSocialTab, openHeistPlanningOverlay, updateHeistRiskAssessment, executeHeistStrike, openKidnapSelectionOverlay, executeKidnap, releaseHostage, payRansom, showKidnapOverlay, startRecoveryTimer, sendAllianceInvite, acceptAlliance, dissolveAlliance, openTrophyView, reportPlayer, openLaunderingTerminal, sendHeistRequest, initiateRegionalSabotage } from './js/criminality.js';
import { 
    playProcedureInterruptedSFX, playLongWarningSFX, playCloakFailureSFX, playCloakDisruptorSFX, playMutationSoundscape, stopMutationSoundscape, playMutationSuccessSFX, playEcosystemAlertSFX, toggleMuteMusic, toggleMuteMaster, toggleMuteSfx, updateMasterVolume, updateMusicVolume, updateSfxVolume
} from './js/audio.js';
import { initAudioContextManager, getAudioContextManager } from './js/audio_context.js';
import { RivalryEngine } from './js/rivalry.js';
import { 
    initParticleSystem, triggerCaptureParticles, triggerGlobalKidnapEffect, 
    triggerMutationScarEffect, triggerCloakFailureParticles, triggerCloakDisruptorParticles
} from './js/particles.js';
import { getAssetSymbol, getCachedEnvoiName, resolveEnvoiName, reportGloat, shortenAddress, resolveAssetSymbol, getNetworkConfig } from './js/utils.js'; // Removed playMutationSoundscape, stopMutationSoundscape

// --- Menu & Controller System (PILLAR 5: UI Orchestration) ---
import { init as initControllerNav, STATE as ControllerState, CONTROLLER_TYPES, handleAction as handleControllerAction, setFocus as setControllerFocus, showFocusRing, hideFocusRing, hapticFeedback, injectConsoleHints, getConsoleHint, enable as enableController, disable as disableController, isActive as isControllerActive, getType as getControllerType, getFocusIndex, getActiveGamepad, detectControllerType } from './js/controller_nav.js';
import { SHAPES, SIZES, GRIDS, PATHWAY_SHAPES, SYSTEM_SHAPES, CONTROLLER_MAP, getShapeForSystem, getPathwayShape, getSize, getGrid, computePositions, getClipPath, getControllerNav, getAllShapeIds, getAllGridIds, getAllSizeIds, RADIAL_GRIDS, isRadialGrid, FLOW_GRIDS, isPositionedGrid } from './js/menu_customization.js';
import { PATHWAYS, calculateTier, getTierProgress, getCSSVariables, injectAnimationStyles, getPathway, getAllPathways } from './js/pathway_avenues.js';
import { get as getUserPrefs, onChange as onUserPrefsChange, isStarred, toggleStar, getStarred, getStarredByTab, reorderStarred, unstar, bindAsset, getBoundAsset, getAllBoundAssets, getLayout, setLayout, getMainMenu, setMainMenu, getMenuLayout, setMenuLayout, getButtonOverride, setButtonOverride, removeButtonOverride, getDefaultShape, getDefaultSize, setDefaultShape, setDefaultSize, getGridType, setGridType, exportPrefs, importPrefs, resetAll, syncFromWasm, syncToWasm } from './js/user_preferences.js';
import { open as openMenuCustomization, close as closeMenuCustomization, switchTab as switchCustomTab, selectShape, selectSize, onGridChange, setControllerScheme, testController, selectPerButton, onPerButtonShapeChange, onPerButtonSizeChange, removePerButtonOverride, saveMenuCustomization, resetMenuCustomization, exportMenuCustomization, importMenuCustomization, renderPreview as renderMenuPreview, renderPerButtonList, renderControllerTab, renderGridDiagram } from './js/menu_customization_panel.js';
import { init as initMenuConstellation, renderStarredItems, openQuickPlay, closeQuickPlay, acceptMatch, openTutorial, closeTutorial, playVideo } from './js/menu-constellation.js';
import { renderGameScreen } from './js/game_screen.js'; // Dynamic board/game screen (was static in index.html)

// --- World Dashboard Tab UIs (PILLAR 5: UI Orchestration) ---
import { initCareerTree } from './js/career_tree.js';
import { initClubFoundry } from './js/club_foundry.js';
import { initTournamentBrackets } from './js/tournament_brackets.js';
import { initJusticeDashboard } from './js/justice_dashboard.js';
import { initLoanTerms } from './js/loan_terms.js';
import { initAmmChart } from './js/amm_chart.js';
import { initPetBreeder } from './js/pet_breeder.js';
import './js/bonded_branding.js'; // §23.5 Branding Studio (binds window.openBondedBranding)
import './js/card_view_skins.js'; // §23.5.5 viewer-scoped card display (binds window.CardViewSkins)
import './js/slide_theming.js'; // §10.6 slide theming: boot/main-menu + WD backgrounds (binds window.SlideTheming)
import './js/note_vocabulary.js'; // note vocabulary: the client's read-only view of the server's note prefixes (binds window.NoteVocab)
import './js/tx_journal.js'; // memo of this browser's OWN transactions; pending is never shown as settled (binds window.VbtTxJournal)
import './js/theme_engine.js'; // §27 client half: sets --theme-accent* from the player's element (binds window.themeEngine). _theme_engine.scss + _variables.scss already consume those tokens, so until this import existed the CSS contract was INERT. Default element is 'fire', which is exactly the token default the compiled :root already ships — so composing it changes nothing until setElement() is called.
import { initChurchStorefront } from './js/church_storefront.js';
import { initRivalryChallenge } from './js/rivalry_challenge.js';
import { initAchievementProgress } from './js/achievement_progress.js';
import { initReportHistory } from './js/report_history.js';
import { initGovernanceChambers } from './js/governance_chambers.js';
import { initTreasureMap } from './js/treasure_map.js';
import { initSeasonalEvents } from './js/seasonal_events.js';
import { initSeasonCountdown } from './js/season_countdown.js';
import { initReplayViewer } from './js/replay_viewer.js';
import { initIdentityEditor } from './js/identity_editor.js';
import { initDividendYield } from './js/dividend_yield.js';
import { initUnderworldContracts } from './js/underworld_contracts.js';
import { initBlackMarket } from './js/black_market.js';
import { initItemsEquip } from './js/items_equip.js';
// The Faction Quartermaster: owner of /api/faction/shop + /api/faction/shop/buy, which had no UI
// anywhere in the client. Placed as a Careers & Factions leaf because the engine gates buying on
// the caller's career role.
import './js/faction_shop.js';
import { initEventsArena } from './js/events_arena.js';
import { initTerritoryMap } from './js/territory_map.js';
import { initRewardsCenter } from './js/rewards_center.js';

import { initCounterfeitScanner } from './js/counterfeit_scanner.js';
import { initCreatorStudio } from './js/creator_studio.js';
import { initFaithWarGambit } from './js/faith_war_gambit.js';
import { initIndustrialFlow } from './js/industrial_flow.js';
import { initMatchCreator } from './js/match_creator.js';
import { initCardAnimations } from './js/card_animations.js';
import { initCampaignMode } from './js/campaign_mode.js';
import { initDeckManager } from './js/deck_manager.js';
import { initEquipmentSystem } from './js/equipment_system.js';
import { initCardProgression } from './js/card_progression.js';
import { initGameBoard } from './js/game_board.js';
import { initGameMultiplayer } from './js/game_multiplayer.js';
import { initCardTitles } from './js/card_titles.js';
import { initPostMatchWagers } from './js/post_match_wagers.js';
import { initCharacterMood } from './js/character_mood.js';
import { initTeaHouse } from './js/tea_house.js';
import { initZenGarden } from './js/zen_garden.js';
import { initNpcTaunts } from './js/npc_taunts.js';
import { initGameModes } from './js/game_modes.js';
import { initGameLocations } from './js/game_locations.js';
import { initTutorialSystem } from './js/tutorial_system.js';
import { initLaunchpad, initAds, initVehicles, initStatsOverlay, initWorldContent, initGamingOS, initLocalModel, initGovernor, initLeaderboard, initCompliance, initMaintenance, initSystemMsg, initOrphans, initRegions } from './js/remaining_tabs.js';
// Orphan Cleaner — standalone overlay reached from the World Dashboard
// (WD_ROUTES.orphan = 'openOrphanCleaner'). The module existed on disk but was
// never composed into the app, so that route was DEAD: the dashboard's
// typeof-guard silently fell back to embedding `initOrphans` and the intended
// surface was unreachable. Importing it (it is a lazy, self-contained IIFE that
// only binds window.* at load) completes the Single-Entry contract and revives
// the route.
import './js/orphan_cleaner.js';
// ============================================================================
// UI MODULE IMPORTS — SINGLE-ENTRY MANDATE
// ----------------------------------------------------------------------------
// index.html is a PURE SHELL: wasm_exec.js + third-party vendor bundles +
// app.js. Every first-party script is imported HERE so there is exactly ONE
// composition root and ONE boot order, with wasm_exec as the glue.
//
// These were previously <script src="js/*.js"> tags inside index.html, which
// caused every module to be evaluated twice (once classic, once as a module)
// and produced two competing UI stacks — the World Dashboard and the Player
// Profile hub behaving like separate apps.
//
// Every file below is a self-contained IIFE that binds its public API to
// window, so import order is not load-bearing beyond app_bridge.js (which must
// define window.WasmWalletBridge before the WASM engine calls into JS).
// ============================================================================

// --- Shell glue (defines window.WasmWalletBridge / UnifiedAlertSystem) ---
import './js/app_bridge.js';

// --- Constellation / lobby shell ---
import './js/constellation_hub.js';
import './js/constellation_spectate.js';
import './js/constellation_tutorial.js';
import './js/placeholder_assets.js';

// --- World Dashboard: the single navigation surface for all gameplay UI ---
import './js/world_dashboard.js';
import './js/theme_dashboard.js';
import './js/system_dashboard.js';
import './js/owner_stats.js';
import './js/community_dashboard.js';
import './js/utilities_dashboard.js';
import './js/security_dashboard.js';
import './js/extended_dashboard.js';
// The admin surface: ONE signature-gated console (opens the Admin Suite panel). The four
// competing dashboards (admin_dashboard / operations_dashboard / final_dashboard / admin_panel)
// were deleted — they were never mounted, never called, and one of them authenticated with a
// hard-coded password.
import './js/admin_console.js';
// The Admin Suite's CONTROLS. The panel's markup used to carry inline `onclick="adminX()"`
// handlers, which resolve on `window` — and eight of those functions had no publisher, so those
// controls were inert. The markup now declares `data-admin-action` and this module binds them.
import { bindAdminSuiteControls } from './js/admin_suite.js';

// --- Living-world feature modules (World Dashboard destinations) ---
import './js/investment_dashboard.js';
import './js/creator_store.js';
import './js/creator_storefront.js';
import './js/ai_citizens.js';
import './js/world_events.js';
import './js/leaderboard_region.js';
import './js/rivalry_viewer.js';
import './js/life_assets.js';
import './js/early_tasks.js';
import './js/combined_events.js';
import './js/religion_governance.js';
import './js/stat_overlay.js';
import './js/industrial_loop.js';
import './js/faith_system.js';
import './js/children_bots.js';
import './js/entity_market.js';
import './js/pet_battle_arena.js';
import './js/persistent_identity.js';
import './js/entity_shares.js';
import './js/infrastructure_lease.js';
import './js/governance.js';
import './js/bridge_router.js';
import './js/creator_economy.js';
import './js/faith_church.js';
import './js/launchpad.js';
import './js/advertising.js';
import './js/gaming_os.js';
import './js/asset_viewer.js';
import './js/dev_game_hub.js';

// --- Criminality / bounty (immersive underworld layer) ---
import './js/underworld.js';
import './js/bounty_tracker.js'; // Live bounty overlay + Bounty Hunter tier tracking buffs

// --- Wallet / transaction surfaces ---
import './js/wallet_state.js';
import './js/wallet_modal.js';
import './js/tx_modal.js';
import './js/faucet_dashboard.js';

// --- Match / spectate surfaces ---
import './js/spectate.js';
import './js/match_arena.js';

// --- Player surfaces ---
import './js/achievements.js';
import './js/daily_challenges.js';
import './js/settings_panel.js';

// --- §25 Web-3D client (registers window.enter3DWorld / window.enterMenuWorld) ---
import './js/world3d.js';

// --- Platform services / utilities ---
import './js/panel_manager.js';
import './js/audio_engine.js';
import './js/error_handler.js';
import './js/first_run.js';



// --- Global Bridge: index.html event mapping ---
window.handleWalletAction = handleWalletAction;
window.connectWith = connectWith;
window.closeWalletSelector = closeWalletSelector;
window.openPayoutSettings = openPayoutSettings;
window.reportPlayer = reportPlayer;
window.savePayoutAddress = savePayoutAddress;
window.reportGloat = reportGloat;
window.requestMatchSync = requestMatchSync;
window.playEcosystemAlertSFX = playEcosystemAlertSFX;
window.toggleMuteMusic = toggleMuteMusic;
window.toggleMuteMaster = toggleMuteMaster;
window.toggleMuteSfx = toggleMuteSfx;
window.setMasterVolume = updateMasterVolume;
window.playMutationSuccessSFX = playMutationSuccessSFX;
window.playCloakDisruptorSFX = playCloakDisruptorSFX;
window.toggleContextualAmbients = toggleContextualAmbients;
window.playProcedureInterruptedSFX = playProcedureInterruptedSFX;
window.playCloakFailureSFX = playCloakFailureSFX;
window.playLongWarningSFX = playLongWarningSFX;
window.setMusicVolume = updateMusicVolume;
window.setSfxVolume = updateSfxVolume;
window.toggleMatchmakingQueue = toggleMatchmakingQueue;
window.sendChatMessage = sendChatMessage;
window.playMutationSoundscape = playMutationSoundscape; // Expose new audio function
window.stopMutationSoundscape = stopMutationSoundscape; // Expose new audio function
window.handleChatKey = handleChatKey;
window.filterSeasonHistory = filterSeasonHistory;
window.openTournamentBracket = openTournamentBracket;
window.closeTournamentBracket = closeTournamentBracket;
window.switchHofTab = switchHofTab;
window.fetchLeaderboard = fetchLeaderboard;
window.fetchTournamentHistory = fetchTournamentHistory;
window.fetchSeasonHistory = fetchSeasonHistory;
window.toggleTournamentDetails = toggleTournamentDetails;
window.proceedToWarRoom = proceedToWarRoom;
window.acceptChallenge = acceptChallenge;
window.declineChallenge = declineChallenge;
window.sendChallenge = sendChallenge;
window.sendSpectate = sendSpectate;
window.triggerToggleNetwork = triggerToggleNetwork;
window.openDeckManager = openDeckManager;
window.closeDeckManager = closeDeckManager;
window.setupCropEvents = setupCropEvents;
window.applyAvatarFilters = applyAvatarFilters;
window.selectAvatar = selectAvatar;
window.refreshInventory = refreshInventory;
window.openShopsOverlay = openShopsOverlay;
window.buyClubItem = buyClubItem;
window.openClubFoundry = openClubFoundry;
window.submitClubFoundry = submitClubFoundry;
window.openTerritoryMapOverlay = openTerritoryMapOverlay;
window.adjustMapZoom = adjustMapZoom;
window.openSocialPanelOverlay = openSocialPanelOverlay;
window.switchSocialTab = switchSocialTab;
window.openPortfolioView = openPortfolioView;
window.openLaunderingTerminal = openLaunderingTerminal;
window.openRecoveryBountyOverlay = openRecoveryBountyOverlay;
window.switchPortfolioTab = switchPortfolioTab;
window.openVaultInteraction = openVaultInteraction;
window.tradeShares = tradeShares;
window.openMutationHistoryOverlay = openMutationHistoryOverlay;
window.adjustMutationVector = adjustMutationVector;
window.submitVectorRealignment = submitVectorRealignment;
window.submitMoodRecalibration = submitMoodRecalibration;
window.submitMutationLoyaltySynthesis = submitMutationLoyaltySynthesis;
window.submitDistrictTax = submitDistrictTax;
window.openBlackMarket = openBlackMarket;
window.buyBlackMarketItem = buyBlackMarketItem;
window.openArtGalleryOverlay = openArtGalleryOverlay;
window.openConsignmentOverlay = openConsignmentOverlay;
window.selectConsignmentItem = selectConsignmentItem;
window.submitConsignment = submitConsignment;
window.promptBid = promptBid;
window.submitBid = submitBid;
window.addXChainWallet = addXChainWallet;
window.submitLinkWallet = submitLinkWallet;
window.openClubLeaseBoard = openClubLeaseBoard;
window.openCreateLeaseOverlay = openCreateLeaseOverlay;
window.submitCreateLease = submitCreateLease;
window.switchFoundryTab = switchFoundryTab; // PILLAR 4: Expose for inline HTML calls
window.takeLease = takeLease;
window.openCourthouse = openCourthouse;
window.submitCourthouseFine = submitCourthouseFine;
window.initiateBail = initiateBail;
window.openSecuritySentry = openSecuritySentry;
window.deployTrap = deployTrap;
window.openBountyBoard = openBountyBoard;
window.openRumorMill = openRumorMill;
window.spreadRumor = spreadRumor;
window.openHeistPlanningOverlay = openHeistPlanningOverlay;
window.updateHeistRiskAssessment = updateHeistRiskAssessment;
window.executeHeistStrike = executeHeistStrike;
window.sendHeistRequest = sendHeistRequest;
window.openKidnapSelectionOverlay = openKidnapSelectionOverlay;
window.executeKidnap = executeKidnap;
window.payRansom = payRansom;
window.releaseHostage = releaseHostage; // Existing
window.sendAllianceInvite = sendAllianceInvite;
window.acceptAlliance = acceptAlliance;
window.dissolveAlliance = dissolveAlliance;
window.openTrophyView = openTrophyView;
window.triggerGlobalKidnapEffect = triggerGlobalKidnapEffect;
window.triggerMutationScarEffect = triggerMutationScarEffect;
window.shareTournamentVictory = shareTournamentVictory;
window.adminSeasonRollover = adminSeasonRollover;
window.initiateRegionalSabotage = initiateRegionalSabotage; // New: Regional Warfare
window.adminExportAuditLog = adminExportAuditLog;
window.adminRestockDLC = adminRestockDLC;
window.adminSimulateMutationSuccess = adminSimulateMutationSuccess;
window.adminSimulateMutationFailure = adminSimulateMutationFailure;
window.adminSimulateTournament = adminSimulateTournament;
window.adminCommissionAudit = adminCommissionAudit;
window.adminTaxAudit = adminTaxAudit;
window.adminDistrictTaxAudit = adminDistrictTaxAudit;
window.adminSimulateMojoDecay = adminSimulateMojoDecay;
window.adminCyberSecurityAudit = adminCyberSecurityAudit;
window.adminForcePayout = adminForcePayout;
window.displayCyberAuditReportInChat = displayCyberAuditReportInChat; // New: Expose for network.js to call
window.adminAssetForfeiture = adminAssetForfeiture;
window.adminToggleDevMode = adminToggleDevMode;
window.adminToggleMaintenance = adminToggleMaintenance;
window.adminUpdateRules = adminUpdateRules;
window.adminBroadcast = adminBroadcast;
window.adminAddReward = adminAddReward;
window.adminRemoveReward = adminRemoveReward;
window.updateAdminRewardRegistry = updateAdminRewardRegistry;
window.adminSetActiveNetwork = adminSetActiveNetwork;
window.adminAddNetwork = adminAddNetwork; // Network registry upsert: sets the indexer/node base(s) the server reads through
window.onAdminNetworkSelectChange = onAdminNetworkSelectChange;
window.selectCard = selectCard;
window.showMutationStabilityTooltip = showMutationStabilityTooltip;
window.hidePowerTooltip = hidePowerTooltip;
window.hideAllOverlays = hideAllOverlays;
window.toggleActionDropdown = function () {
    const dd = document.getElementById('action-dropdown');
    if (dd) dd.classList.toggle('open');
};
window.switchShopCategory = switchShopCategory;
window.clickGrid = clickGrid;
window.executeQuickCast = executeQuickCast;

// --- Menu & Controller Bridge ---
window.ControllerNav = {
    init: initControllerNav,
    STATE: ControllerState,
    CONTROLLER_TYPES,
    handleAction: handleControllerAction,
    setFocus: setControllerFocus,
    showFocusRing,
    hideFocusRing,
    hapticFeedback,
    injectConsoleHints,
    getConsoleHint,
    enable: enableController,
    disable: disableController,
    isActive: isControllerActive,
    getType: getControllerType,
    getFocusIndex,
    getActiveGamepad,
    detectControllerType,
};
window.MenuCustomization = {
    SHAPES,
    SIZES,
    GRIDS,
    RADIAL_GRIDS,
    FLOW_GRIDS,
    PATHWAY_SHAPES,
    SYSTEM_SHAPES,
    CONTROLLER_MAP,
    getShapeForSystem,
    getPathwayShape,
    getSize,
    getGrid,
    computePositions,
    getClipPath,
    getControllerNav,
    getAllShapeIds,
    getAllGridIds,
    getAllSizeIds,
    isRadialGrid,
    isPositionedGrid,
};
window.PathwayAvenues = {
    PATHWAYS,
    calculateTier,
    getTierProgress,
    getCSSVariables,
    injectAnimationStyles,
    getPathway,
    getAllPathways,
};
// --- World Dashboard Tab UI Bridge ---
window.initCareerTree = initCareerTree;
window.initClubFoundry = initClubFoundry;
window.initTournamentBrackets = initTournamentBrackets;
window.initJusticeDashboard = initJusticeDashboard;
window.initLoanTerms = initLoanTerms;
window.initAmmChart = initAmmChart;
window.initPetBreeder = initPetBreeder;
window.initChurchStorefront = initChurchStorefront;
window.initRivalryChallenge = initRivalryChallenge;
window.initAchievementProgress = initAchievementProgress;
window.initReportHistory = initReportHistory;
window.initGovernanceChambers = initGovernanceChambers;
window.initTreasureMap = initTreasureMap;
window.initSeasonCountdown = initSeasonCountdown;
window.initReplayViewer = initReplayViewer;
window.initIdentityEditor = initIdentityEditor;
window.initDividendYield = initDividendYield;
window.initUnderworldContracts = initUnderworldContracts;
window.initBlackMarket = initBlackMarket;
window.initItemsEquip = initItemsEquip;
window.initEventsArena = initEventsArena;
window.initTerritoryMap = initTerritoryMap;
window.initRewardsCenter = initRewardsCenter;
window.initMatchArena = window.openMatchArena || function () {};
window.initCounterfeitScanner = initCounterfeitScanner;
window.initCreatorStudio = initCreatorStudio;
window.initFaithWarGambit = initFaithWarGambit;
window.initIndustrialFlow = initIndustrialFlow;
window.initMatchCreator = initMatchCreator;
window.initCardAnimations = initCardAnimations;
window.initCampaignMode = initCampaignMode;
window.initLaunchpad = initLaunchpad;
window.initDeckManager = initDeckManager;
window.initEquipmentSystem = initEquipmentSystem;
window.initCardProgression = initCardProgression;
window.initGameBoard = initGameBoard;
window.initAds = initAds;
window.initVehicles = initVehicles;
window.initStatsOverlay = initStatsOverlay;
window.initWorldContent = initWorldContent;
window.initGamingOS = initGamingOS;
window.initLocalModel = initLocalModel;
window.initGovernor = initGovernor;
window.initLeaderboard = initLeaderboard;
window.initCompliance = initCompliance;
window.initMaintenance = initMaintenance;
window.initSystemMsg = initSystemMsg;
window.initOrphans = initOrphans;
window.initRegions = initRegions;
window.UserPreferences = {
    get: getUserPrefs,
    onChange: onUserPrefsChange,
    isStarred,
    toggleStar,
    getStarred,
    getStarredByTab,
    reorderStarred,
    unstar,
    bindAsset,
    getBoundAsset,
    getAllBoundAssets,
    getLayout,
    setLayout,
    getMainMenu,
    setMainMenu,
    getMenuLayout,
    setMenuLayout,
    getButtonOverride,
    setButtonOverride,
    removeButtonOverride,
    getDefaultShape,
    getDefaultSize,
    setDefaultShape,
    setDefaultSize,
    getGridType,
    setGridType,
    exportPrefs,
    importPrefs,
    resetAll,
    syncFromWasm,
    syncToWasm,
};
window.openMenuCustomization = openMenuCustomization;
window.closeMenuCustomization = closeMenuCustomization;
window.switchCustomTab = switchCustomTab;
window.selectShape = selectShape;
window.selectSize = selectSize;
window.onGridChange = onGridChange;
window.setControllerScheme = setControllerScheme;
window.testController = testController;
window.selectPerButton = selectPerButton;
window.onPerButtonShapeChange = onPerButtonShapeChange;
window.onPerButtonSizeChange = onPerButtonSizeChange;
window.removePerButtonOverride = removePerButtonOverride;
window.saveMenuCustomization = saveMenuCustomization;
window.resetMenuCustomization = resetMenuCustomization;
window.exportMenuCustomization = exportMenuCustomization;
window.importMenuCustomization = importMenuCustomization;
window.renderMenuPreview = renderMenuPreview;
window.renderPerButtonList = renderPerButtonList;
window.renderControllerTab = renderControllerTab;
window.renderGridDiagram = renderGridDiagram;
window.renderStarredItems = renderStarredItems;
window.openQuickPlay = openQuickPlay;
window.closeQuickPlay = closeQuickPlay;
window.acceptMatch = acceptMatch;
window.openTutorial = openTutorial;
window.closeTutorial = closeTutorial;
window.playVideo = playVideo;
window.initMenuConstellation = initMenuConstellation;
window.onload = async () => {
    console.log("[ARENA] Initiating Neural Uplink...");
    
    const go = new Go();
    try {
        const result = await WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject);
        go.run(result.instance);
        console.log("[ARENA] WASM Engine ACTIVE.");
        renderGameScreen(); // Build the board/game screen DOM (replaces static index.html markup)
        // Bind the Admin Suite's controls. The shell markup declares WHAT each control does
        // (`data-admin-action`); this is the ONE place that says HOW, so no control depends on an
        // inline handler resolving a global.
        bindAdminSuiteControls();

        // §23.5.5: read the viewer's OWN card-display preference once the engine is up. It is a
        // no-op without a wallet (and the board/quick-play paths re-check lazily afterwards).
        if (window.CardViewSkins) window.CardViewSkins.refresh();

        // PILLAR 6: Client Beacon Recovery (Warm Start).
        // Immediately prime the engine with the last known "Push" state from the server.
        const cachedBeacon = localStorage.getItem("vbabes_state_beacon");
        if (cachedBeacon) {
            try {
                const beacon = JSON.parse(cachedBeacon);

                // PILLAR 4: Session Identity Restoration.
                // Must restore player index before syncing profile to ensure correct slot hydration.
                if (beacon.local_player_index !== undefined && window.SetLocalPlayerIndex) {
                    window.SetLocalPlayerIndex(beacon.local_player_index);
                }

                if (window.SyncFullProfile) window.SyncFullProfile(beacon.profile);
                // PILLAR 4: Sequence Restoration.
                // Restore the Replay Engine sequence count to enable seamless catch-up.
                if (beacon.profile.last_sequence_id && window.SetLastSequenceID) {
                    window.SetLastSequenceID(beacon.profile.last_sequence_id);
                }

                // PILLAR 4: Active State Restoration.
                // Re-prime the engine with the phase and board state from the beacon.
                if (beacon.profile.phase && window.SetPhase) window.SetPhase(beacon.profile.phase);
                if (beacon.profile.board && window.SetBoardState) window.SetBoardState(beacon.profile);

                // PILLAR 2: Integer Supremacy. Ensure vault_balance is passed as float for WASM.
                // The WASM engine will convert it to its internal float representation.
                // The server's authoritative micro-unit balance is used for calculations.
                if (window.SyncVaultBalance) {
                    // beacon.vault_balance is already float from app.js's syncUI
                    window.SyncVaultBalance(beacon.vault_balance);
                }
                if (window.SyncRewards) window.SyncRewards(beacon.rewards);
                if (window.SyncClubs) window.SyncClubs(beacon.clubs);
            } catch (e) { console.warn("[BOOT] Beacon corrupt or unavailable."); }
        }

        // 1. Initial configuration sync
        if (window.SetApiBase) window.SetApiBase(CONFIG.API_BASE);
        if (window.SetAssetBase) window.SetAssetBase(CONFIG.ASSET_URL);

        // 2. Establish Network Switchboard
        initWebSocket(handleServerMessage);

        // 3. Initialize Visuals
        initParticleSystem();
        buildEmptyBoard();

        // 4. Initial Heartbeat
        setInterval(sendPing, 30000);
        
        window.syncUI();

    // PILLAR 6: AudioContextManager initialization.
        // Initialize contextual audio after WASM is active to ensure it has the audio module reference.
        // We'll set it up once the WebSocket handshake completes and first syncUI fires.
        
    // PILLAR 4: Warm-Boot Restoration.
        // If the beacon restored an 'Active' state, trigger the catch-up protocol.
        const postSyncState = window.GetGameState("combat");
        if (postSyncState && postSyncState.phase === "Active") {
            rejoinActiveMatch();
        }
    } catch (err) {
        console.error("[BOOT ERROR] Engine initialization failed:", err);
        showToast("❌ Critical Error: Neural Uplink Failed. Please refresh.", "error", 0);
     }
 };

/**
 * handlePhaseMusicTransition - Bridges syncUI phase changes to AudioContextManager.
 * PILLAR 6: Phase-Based Atmosphere. Dispatches game_phase_change events for context-aware music.
 */
function handlePhaseMusicTransition(state) {
    const phaseToContext = {
        "DISCONNECTED": "menu",
        "Setup": "lobby",
        "Lobby": "lobby",
        "PreGame": "casual_2p",
        "Active_Casual": "casual_2p",
        "Active_Quick": "quick_play",
        "Active_Tournament": "tournament",
        "Active": "combat",
        "TournamentLobby": "tournament_lobby",
        "Finished": "finished"
    };
    
    const context = phaseToContext[state.phase] || null;
    if (context && context !== window._lastAudioContext) {
        window._lastAudioContext = context;
        window.dispatchEvent(new CustomEvent('game_phase_change', { detail: { context, reason: 'syncUI' } }));
    }
}

// --- UI Performance Layer ---
const UI_CACHE = new Map();
const getEl = (id) => {
    if (!UI_CACHE.has(id)) UI_CACHE.set(id, document.getElementById(id));
    return UI_CACHE.get(id);
};

let dashboardCache = { stateKey: "", lastBalance: -1, lastGhostActive: null };

/**
 * updateMojoDecayStatus displays the current Mojo decay rate if a stabilizer is active.
 * PILLAR 1: Infrastructure Prestige.
 */
function updateMojoDecayStatus(state) {
    const container = getEl("mojo-decay-status-hud");
    if (!container) return;

    const isStabilizerActive = state.is_mojo_stabilizer_active;
    const decayRate = state.mojo_decay_rate;

    if (!isStabilizerActive || decayRate === 0) {
        container.classList.add("hidden");
        return;
    }

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-neon-cyan flex-row align-center gap-5 accelerated" 
             style="background: rgba(0, 242, 254, 0.1); height: 32px;"
             title="MOJO DECAY MITIGATION ACTIVE">
            <span class="text-neon-cyan font-bold font-size-0-7em letter-spacing-1">📉 DECAY:</span>
            <b class="text-white font-mono font-size-0-8em">${(decayRate * 100).toFixed(1)}%</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateSolvencyDashboard renders a reactive indicator of the Arena's coverage ratio.
 * PILLAR 2: Ledger Integrity. Supports high-fidelity administrative overrides.
 */
function updateSolvencyDashboard(state, adminData = null) {
    const container = getEl("solvency-hud-container");
    if (!container) return;

    // PILLAR 5: Data Authority. Prioritize backend admin audit data over local WASM sync.
    const physical = adminData ? adminData.physical_vault : (state.faucet_micro || 0);
    const virtual = adminData ? adminData.virtual_liabilities : (state.total_virtual_liability || 0);
    const healthy = adminData ? adminData.kernel_healthy : true;
    const report = adminData ? adminData.audit_report : "Real-time ledger synchronization active.";

    // PILLAR 2: Coverage Calculation.
    // Default to 100% if no liabilities exist (perfectly solvent).
    const ratio = virtual > 0 ? (physical / virtual) : 1.0;
    const percent = (ratio * 100).toFixed(1);
    
    // PILLAR 2: Solvency Tiers.
    let status = ratio >= 1.0 ? "HEALTHY" : "CRITICAL";
    if (!healthy) status = "DEGRADED"; // Structural drift detected by kernel

    let color = ratio >= 1.0 ? "var(--neon-green)" : "#ff4b4b";
    if (!healthy) color = "var(--warning-orange)"; // Integrity warning color (Drift)

    container.innerHTML = `
        <div class="glass-panel p-5-10 flex-row align-center gap-5 accelerated" 
             style="background: rgba(63, 185, 80, 0.05); height: 32px; border-color: ${color}; cursor: ${adminData ? 'help' : 'default'};"
             title="${report}">
            <span style="color: ${color}; font-weight: bold; font-size: 0.7em; letter-spacing: 1px;">⚖️ COV:</span>
            <b class="text-white font-mono font-size-0-8em">${percent}%</b>
            <span class="font-xs opacity-5 ml-5" style="color: ${color}; font-weight: bold;">${status}</span>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateVolumeSlidersUI synchronizes the range inputs with persisted values.
 * PILLAR 4: Persistence Hardening.
 */
function updateVolumeSlidersUI() {
    const mv = getEl("master-volume");
    const mu = getEl("music-volume");
    const sf = getEl("sfx-volume");
    
    // Values are retrieved from localStorage via audio.js logic
    if (mv) mv.value = localStorage.getItem('masterVolume') || 0.5;
    if (mu) mu.value = localStorage.getItem('musicVolume') || 0.5;
    if (sf) sf.value = localStorage.getItem('sfxVolume') || 0.5;
    
    // Update contextual ambients toggle status
    updateAmbientsToggleUI();
}

/**
 * updateAmbientsToggleUI updates the contextual ambients button/icon based on persisted state.
 * PILLAR 6: Audio Context Manager.
 */
function updateAmbientsToggleUI() {
    const statusEl = getEl("ambients-status");
    const btnEl = getEl("ambients-toggle-btn");
    
    if (!statusEl || !btnEl) return;
    
    const isEnabled = localStorage.getItem('contextualAmbients') === 'true';
    
    statusEl.innerText = isEnabled ? "ON" : "OFF";
    statusEl.style.color = isEnabled ? "var(--neon-green)" : "var(--opacity-6, rgba(255,255,255,0.6))";
    btnEl.innerText = isEnabled ? "🎙️" : "⏺️"; // Active mic vs inactive
    
    // Add subtle glow if enabled
    if (isEnabled) {
        btnEl.style.boxShadow = "0 0 8px var(--neon-green)";
        btnEl.style.borderColor = "var(--neon-green)";
    } else {
        btnEl.style.boxShadow = "";
        btnEl.style.borderColor = "";
    }
}

/**
 * toggleContextualAmbients toggles the contextual ambients feature on/off.
 * PILLAR 6: Audio Context Manager.
 */
function toggleContextualAmbients() {
    const audioCtx = getAudioContextManager();
    if (!audioCtx) {
        showToast("Audio context not yet initialized. Try again in a moment.", "warning");
        return;
    }
    
    const isEnabled = localStorage.getItem('contextualAmbients') === 'true';
    const newState = !isEnabled;
    
    localStorage.setItem('contextualAmbients', String(newState));
    
    // Apply to audio context manager
    audioCtx.setAmbientEnabled(newState);
    
    // Update UI immediately
    updateAmbientsToggleUI();
    
    // Log the change for audit purposes
    console.log(`[Audio Context] Contextual ambients ${newState ? 'enabled' : 'disabled'}`);
}

/**
 * displayCyberAuditReportInChat processes and displays a Cyber-Audit report in the chat.
 * This function is called by network.js when a Cyber-Audit admin_notification is received.
 * PILLAR 3: Intelligence Display.
 */
export function displayCyberAuditReportInChat(reportText) {
    // Directly render the detailed report in the chat using the game.js helper
    import('./js/game.js').then(m => m.renderChatMessage("SYSTEM", reportText));
}
window.displayCyberAuditReportInChat = displayCyberAuditReportInChat;

/**
 * triggerFoundryFusion - Special FX for Club actions.
 */
window.triggerFoundryFusion = (type) => {
    import('./js/particles.js').then(m => m.triggerFoundryFusion(type));
};

/**
 * updateDistrictStabilizerHUD renders a status widget for the active mojo stabilizer.
 * PILLAR 1: Infrastructure Prestige.
 */
function updateDistrictStabilizerHUD(state) {
    const container = getEl("district-stabilizer-hud");
    if (!container) return;

    const myClub = globalClubs[state.employer_id];
    const expiry = myClub?.buff_expirations?.["MOJO_STABILIZER"];
    const isStabilizerActive = expiry && new Date(expiry) > Date.now();

    if (!isStabilizerActive) {
        container.classList.add("hidden");
        return;
    }

    const remaining = Math.max(0, new Date(expiry) - Date.now());
    const totalHours = Math.floor(remaining / 3600000);
    const mins = Math.ceil((remaining % 3600000) / 60000);

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-neon-cyan flex-row align-center gap-5 accelerated" 
             style="background: rgba(0, 242, 254, 0.1); height: 32px;"
             title="MOJO STABILIZER FIELD ACTIVE">
            <span class="text-neon-cyan font-bold font-size-0-7em letter-spacing-1">📡 STABILIZER:</span>
            <b class="text-white font-mono font-size-0-8em">${totalHours}h ${mins}m</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateSabotageHUD renders a persistent countdown for owners during blackouts.
 * PILLAR 1: Regional Warfare Intelligence.
 */
function updateSabotageHUD(state) {
    const container = getEl("sabotage-hud-container");
    if (!container) return;

    const myClub = globalClubs[state.employer_id];
    const isOwner = myClub && myClub.owner_wallet && userAddress && myClub.owner_wallet.toLowerCase() === userAddress.toLowerCase();

    if (!isOwner || !myClub.buff_expirations) {
        container.classList.add("hidden");
        return;
    }

    const disruptions = Object.entries(myClub.buff_expirations)
        .filter(([key, expiry]) => key.startsWith("DISRUPTION_") && new Date(expiry) > Date.now());

    if (disruptions.length === 0) {
        container.classList.add("hidden");
        return;
    }

    // Display only the most urgent disruption (shortest time remaining)
    const mostUrgent = disruptions.sort((a, b) => new Date(a[1]) - new Date(b[1]))[0];
    const remaining = Math.max(0, new Date(mostUrgent[1]) - Date.now());
    const mins = Math.ceil(remaining / 60000);

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-error animate-pulse flex-row align-center gap-5" style="background: rgba(255, 0, 255, 0.15); height: 32px;">
            <span class="text-error font-bold font-size-0-7em letter-spacing-1">📡 BLACKOUT:</span>
            <b class="text-warning font-mono font-size-0-8em">${mins}m</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateMutationStabilityHUD renders the real-time success chance for lab personnel.
 * PILLAR 6: Specialized Gene-Editing.
 */
function updateMutationStabilityHUD(state) {
    const container = getEl("mutation-stability-hud");
    if (!container) return;

    const myClub = globalClubs[state.employer_id];
    if (!myClub || (myClub.type !== "Vitality" && myClub.type !== "Elemental")) {
        container.classList.add("hidden");
        return;
    }

    const mojo = myClub.club_mojo || 0;
    const staffCount = Object.keys(myClub.staff || {}).length;
    const hasInsurance = state.has_mutation_insurance;
    const isGovernor = (myClub.territories?.length || 0) + (myClub.allied_club_id ? (globalClubs[myClub.allied_club_id]?.territories?.length || 0) : 0) >= 2;
    const isSabotaged = myClub.buff_expirations?.["SABOTAGE"] && new Date(myClub.buff_expirations["SABOTAGE"]) > Date.now();
    const isTrainingActive = myClub.buff_expirations?.["STAFF_TRAINING"] && new Date(myClub.buff_expirations["STAFF_TRAINING"]) > Date.now();

    let chance = 0.70;
    if (hasInsurance) {
        chance = 1.0;
    } else {
        let mojoBonus = mojo / 5000.0;
        if (mojoBonus > 0.20) mojoBonus = 0.20;
        chance += mojoBonus;
        let staffBonus = staffCount * 0.02;
        if (staffBonus > 0.10) staffBonus = 0.10;
        chance += staffBonus;
        if (isSabotaged) chance -= 0.15;
        if (isGovernor) chance += 0.05;
        if (isTrainingActive) chance += 0.05;
        if (chance > 0.98) chance = 0.98;
        if (chance < 0.50) chance = 0.50;
    }

    const percent = Math.floor(chance * 100);
    const statusClass = percent >= 90 ? 'text-neon-green' : percent >= 70 ? 'text-neon-cyan' : 'text-warning';

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-neon-purple flex-row align-center gap-5 accelerated" 
             style="background: rgba(180, 0, 255, 0.1); height: 32px; cursor: help;"
             onmouseenter="window.showMutationStabilityTooltip(event, ${mojo}, ${staffCount}, ${hasInsurance}, ${isSabotaged}, ${isGovernor}, ${isTrainingActive})"
             onmouseleave="window.hidePowerTooltip()">
            <span class="text-neon-purple font-bold font-size-0-7em letter-spacing-1">🧬 STABILITY:</span>
            <b class="${statusClass} font-mono font-size-0-8em">${percent}%</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateBountyWarningHUD displays a high-priority alarm for outlaws with active bounties.
 * PILLAR 3: Criminality & Intelligence.
 */
function updateBountyWarningHUD(state) {
    const container = getEl("active-bounty-warning");
    if (!container) return;

    const wanted = state.wanted_level || 0;
    const isGhostActive = state.ghost_protocol_expires_at && new Date(state.ghost_protocol_expires_at) > Date.now();

    // PILLAR 2: Cloak Failure Trigger.
    // Detect the transition from Active to Expired while under high infamy.
    if (dashboardCache.lastGhostActive === true && !isGhostActive && wanted > 10) {
        if (window.triggerCloakFailureParticles) window.triggerCloakFailureParticles();
        if (window.playCloakFailureSFX) window.playCloakFailureSFX();
        showToast("⚠️ <b>CLOAK FAILURE:</b> Your signal is now visible on the Bounty Board!", "critical");
    }
    
    dashboardCache.lastGhostActive = isGhostActive;

    // Trigger alarm if Wanted Level is 10 or higher and signals are not scrambled.
    if (wanted > 10 && !isGhostActive) {
        container.innerHTML = `
            <div class="glass-panel p-5-10 border-error animate-pulse flex-row align-center gap-5 accelerated" 
                 style="background: rgba(255, 75, 75, 0.2); height: 32px; box-shadow: 0 0 10px rgba(255, 75, 75, 0.3);">
                <span class="text-error font-bold font-size-0-7em letter-spacing-1">🎯 BOUNTY ACTIVE:</span>
                <b class="text-white font-mono font-size-0-8em">${wanted * 50} $VBV</b>
            </div>`;
        container.classList.remove("hidden");
    } else {
        container.classList.add("hidden");
    }
}

/**
 * updateBountyHunterHUD renders tactical tracking intel for clean players.
 * PILLAR 3: Criminality & Intelligence.
 */
function updateBountyHunterHUD(state) {
    const container = getEl("bounty-hunter-hud");
    if (!container) return;

    const myWanted = state.wanted_level || 0;
    // Bounty Hunters must maintain a clean record (Wanted Level <= 2).
    if (myWanted > 2) {
        container.classList.add("hidden");
        return;
    }

    // Find high-priority targets (Wanted >= 10) who aren't under Ghost Protocol.
    const outlaws = lastLobbyPlayers.filter(p => {
        const isGhost = p.ghost_protocol_expires_at && new Date(p.ghost_protocol_expires_at) > Date.now();
        return (p.wanted_level || 0) >= 10 && !isGhost && p.id !== myClientId;
    });

    if (outlaws.length > 0) {
        const target = outlaws.sort((a, b) => b.wanted_level - a.wanted_level)[0];
        const name = getCachedEnvoiName(target.wallet);
        const district = (target.last_seen_district || "Sector Unknown").replace(/_/g, ' ').toUpperCase();

        container.innerHTML = `
            <div class="glass-panel p-5-10 border-neon-cyan flex-row align-center gap-5 accelerated" 
                 style="background: rgba(0, 242, 254, 0.1); height: 32px; cursor: pointer; border-style: dashed;"
                 title="CLICK TO ENGAGE OUTLAW"
                 onclick="window.sendChallenge('${target.id}')">
                <span class="text-neon-cyan font-bold font-size-0-7em letter-spacing-1">📡 TRACKING:</span>
                <b class="text-white font-mono font-size-0-8em">${name}</b>
                <span class="text-neon-purple font-mono font-size-0-7em ml-5">[${district}]</span>
            </div>`;
        container.classList.remove("hidden");
    } else {
        container.classList.add("hidden");
    }
}

/**
 * updateDistrictStabilizerVisuals triggers the shimmering grid effect if the buff is active.
 * PILLAR 1: Infrastructure Prestige.
 */
function updateDistrictStabilizerVisuals(state) {
    const myClub = globalClubs[state.employer_id];
    // Check for MOJO_STABILIZER buff expiration
    const isStabilizerActive = myClub?.buff_expirations?.["MOJO_STABILIZER"] && new Date(myClub.buff_expirations["MOJO_STABILIZER"]) > Date.now();
    
    // Trigger or remove the grid effect
    triggerDistrictStabilizerEffect(isStabilizerActive);

    // PILLAR 1: Infrastructure Audio.
    if (isStabilizerActive) { if (window.playDistrictStabilizerThrum) window.playDistrictStabilizerThrum(); }
    else { if (window.stopDistrictStabilizerThrum) window.stopDistrictStabilizerThrum(); }
}

/**
 * updateCommissionSummaryHUD renders the total alliance dividends for Regional Governors.
 * PILLAR 1: Industrial Loop.
 */
function updateCommissionSummaryHUD(state) {
    const container = getEl("commission-summary-hud");
    if (!container) return;

    const myClubID = state.employer_id;
    const myClub = globalClubs[myClubID];
    
    // Verification: Must be the owner to see the organization's war chest summary.
    const isOwner = myClub && myClub.owner_wallet && userAddress && myClub.owner_wallet.toLowerCase() === userAddress.toLowerCase();
    
    if (!isOwner || !myClub.commission_history || myClub.commission_history.length === 0) {
        container.classList.add("hidden");
        return;
    }

    const totalEarned = myClub.commission_history.reduce((sum, event) => sum + (event.amount || 0), 0);

    if (totalEarned <= 0) {
        container.classList.add("hidden");
        return;
    }

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-neon-green flex-row align-center gap-5 accelerated" 
             style="background: rgba(63, 185, 80, 0.1); height: 32px; cursor: help;"
             title="Rolling Dividend Total (Alliance Procedures)">
            <span class="text-neon-green font-bold font-size-0-7em letter-spacing-1">💰 DIVIDENDS:</span>
            <b class="text-white font-mono font-size-0-8em">${totalEarned.toFixed(2)} $VBV</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateBountyTallyHUD calculates and displays the total $VBV value of all active bounties.
 * PILLAR 3: Criminality & Intelligence.
 */
function updateBountyTallyHUD() {
    const container = getEl("bounty-tally-hud");
    if (!container) return;

    let totalBounty = 0;
    lastLobbyPlayers.forEach(p => {
        const isGhost = p.ghost_protocol_expires_at && new Date(p.ghost_protocol_expires_at) > Date.now();
        if ((p.wanted_level || 0) >= 10 && !isGhost) {
            totalBounty += (p.wanted_level * 50);
        }
    });

    if (totalBounty > 0) {
        container.innerHTML = `
            <div class="glass-panel p-5-10 border-gold flex-row align-center gap-5 accelerated" 
                 style="background: rgba(255, 215, 0, 0.1); height: 32px;"
                 title="TOTAL ACTIVE BOUNTIES IN SECTOR">
                <span class="text-gold font-bold font-size-0-7em letter-spacing-1">💰 SECTOR BOUNTY:</span>
                <b class="text-white font-mono font-size-0-8em">${totalBounty} $VBV</b>
            </div>`;
        container.classList.remove("hidden");
    } else {
        container.classList.add("hidden");
    }
}

/**
 * window.syncUI - The heart of the client orchestrator.
 * Reads authoritative state from the WASM engine and updates the DOM.
 */
window.syncUI = (scope = "all", overrideData = null) => {
    const state = window.GetGameState(scope);
    if (!state) return;

    // PERFORMANCE GUARD: Detect state changes to prevent redundant re-renders
    // PILLAR 5: Precise Synchronization. Include tournament and player count in state key.
    const scannerActive = state.district_scanner_expires_at && (new Date(state.district_scanner_expires_at) > Date.now());
    const cyberJammerActive = state.has_cyber_jammer;
    const isGhostActive = state.ghost_protocol_expires_at && new Date(state.ghost_protocol_expires_at) > Date.now();
    const isQueued = state.in_matchmaking_queue;
    const isMaint = state.maintenance ? "ON" : "OFF";
    const maintPrio = state.maintenance_priority || "info";
    const currentStateKey = `${state.phase}-${state.turn}-${state.wanted_level}-${state.reputation}-${state.tournament?.matches?.length || 0}-${state.tournament?.participants?.length || 0}-${scannerActive}-${cyberJammerActive}-${isGhostActive}-${isQueued}-${isMaint}-${maintPrio}`;

    if (scope !== "combat" && scope !== "solvency_override" && currentStateKey === dashboardCache.stateKey && state.faucet === dashboardCache.lastBalance) {
        // Only proceed if scope is combat or state has evolved
        if (scope !== "all") return;
    }
    dashboardCache.stateKey = currentStateKey;
    dashboardCache.lastBalance = state.faucet;

    // --- Board visibility: the game board must ONLY show during an Active match,
    //     a live spectate session, or replay catch-up. In the lobby/menu it is hidden
    //     and a lobby placeholder takes its place (operator mandate). ---
    const isSpectating = spectatorMatchState !== null;
    const isReplay = !!(state.replay_state && state.replay_state !== "SYNCHRONIZED");
    const boardVisible = state.phase === "Active" || isSpectating || isReplay;
    const boardEl = document.getElementById("board-container");
    const p2El = document.getElementById("p2-info");
    const p1El = document.getElementById("p1-info");
    const thinkEl = document.getElementById("ai-thinking-indicator");
    const lobbyEl = document.getElementById("lobby-placeholder");
    if (boardEl) boardEl.classList.toggle("hidden", !boardVisible);
    if (p2El) p2El.classList.toggle("hidden", !boardVisible);
    if (p1El) p1El.classList.toggle("hidden", !boardVisible);
    if (thinkEl) thinkEl.classList.toggle("hidden", !boardVisible);
    if (lobbyEl) lobbyEl.classList.toggle("hidden", boardVisible);
    document.body.classList.toggle("lobby-mode", !boardVisible);
    // Constellation hub is the main screen in the lobby; reveal the (app.js-rendered)
    // game screen only when an Active match / spectate / replay is showing.
    if (boardVisible) document.body.classList.remove("nexus-lobby-active");

    // PILLAR 6: Audio Context Transitions.
    if (scope === "all" || scope === "meta") {
        handlePhaseMusicTransition(state);
        
        // Dispatch game phase change event for AudioContextManager
        initAudioContextManager(window.AudioModuleInstance);
        const audioCtx = getAudioContextManager();
        if (audioCtx) {
            const contextMap = {
                'Lobby': 'lobby',
                'Active': 'combat',
                'TournamentLobby': 'tournament',
                'PreGame': 'casual_2p'
            };
            const contextKey = contextMap[state.phase] || 'menu';
            audioCtx.transitionToContext(contextKey, 'syncUI');
        }
    }

    // PILLAR 4: Critical Alerts.
    // The native VOI 'gas_warning' toast is handled by the 'admin_notification'
    // system (network.js -> ui.js:showToast), not directly by syncUI.
    // syncUI primarily updates persistent UI elements based on GetGameState.
    // PILLAR 2: Usable Total.
    // Combine physical blockchain balance ($VBV) with virtual salary/heist rewards.
    // PILLAR 5: Delta-Safe Sync. Only update liquidity if values are present in scope.
    // PILLAR 2: Integer Supremacy alignment. state.faucet is the Arena's physical balance.
    const physicalVBV = state.faucet;
    const virtualVBV = state.virtual_balance;
    const totalLiquid = (physicalVBV !== undefined && virtualVBV !== undefined) ? (physicalVBV + virtualVBV) : null;

    const liquidEl = getEl("total-liquid-balance");
    if (liquidEl && totalLiquid !== null) liquidEl.innerText = totalLiquid.toFixed(2);

    updateVolumeSlidersUI(); // Ensure volume sliders reflect persisted values

    // --- Domain Orchestration ---
    if (scope === "combat" || scope === "all") {
        updateDynamicArenaFloor(state);
        syncBoardParticles(state);
        updateSpectatorHUD(state);

        // PILLAR 4: Session Identity styling.
        updateAvatarIdentityStyle(state);
        updateStaffTrainingVisuals(state);
        updateMoodCatalystVisuals(state);

        // PILLAR 3: Interaction Lock Reset.
        // authoritative update received; clear the quick-cast lock.
        setPendingQuickCastId(null);

        // PILLAR 5: Granular Node-Diffing (Combat Grid).
        // Update individual grid slots to preserve the .onclick listeners 
        // established in game.js:buildEmptyBoard.
        const slots = document.querySelectorAll(".grid-slot");
        if (state.board && slots.length === 9) {
            state.board.forEach((card, i) => {
                const slot = slots[i];
                if (!card) {
                    if (slot.innerHTML !== "") slot.innerHTML = "";
                    slot.classList.remove("owner-0", "owner-1", "common", "rare", "epic", "legendary");
                    return;
                }
                
                const cardHTML = renderCardHTML(card);
                if (slot.innerHTML !== cardHTML) {
                    slot.innerHTML = cardHTML;

                    // PILLAR 5: Visual Authority. 
                    // Apply owner and rarity classes to the slot for CSS targeting.
                    slot.classList.remove("owner-0", "owner-1", "common", "rare", "epic", "legendary");
                    slot.classList.add(`owner-${card.owner}`);
                    
                    const r = card.rarity || 1.0;
                    if (r >= 2.0) slot.classList.add("legendary");
                    else if (r >= 1.5) slot.classList.add("epic");
                    else if (r > 1.0) slot.classList.add("rare");
                    else slot.classList.add("common");
                    
                    // PILLAR 4: Reactive Captured Animation.
                    if (card.is_combo) {
                        slot.classList.remove("flip-capture");
                        void slot.offsetWidth; // Force Reflow to re-trigger animation
                        slot.classList.add("flip-capture");
                    } else {
                        slot.classList.remove("flip-capture");
                    }
                }
            });
        }
        
        // Dynamic taunts from NPCs based on observed playstyle
        if (state.multiplayer === false && state.phase === "Active") {
            // Heuristic to ensure taunts don't spam every frame
            const turnKey = `${state.turn}`;
            if (window.lastTauntTurn !== turnKey) {
                window.lastTauntTurn = turnKey;
                import('./collective-intelligence.js').then(m => {
                    const taunt = m.collectiveIntelligence.generatePlaystyleTaunt(state.p2_name, state.playstyle);
                    if (taunt) renderChatMessage("SYSTEM", taunt);
                });
            }
        }
        updateSabotageHUD(state);
        updateMutationStabilityHUD(state);
        updateDistrictStabilizerHUD(state);
    }

    // PILLAR 5: Matchmaking Interface Sync.
    // Ensure the button state and status text reflect the authoritative WASM engine state.
    const matchmakingBtn = getEl("btn-matchmaking");
    const queueStatus = getEl("queue-status");
    if (matchmakingBtn && queueStatus && state.in_matchmaking_queue !== undefined) {
        matchmakingBtn.disabled = false; // PILLAR 5: Re-enable once authoritative state is synced
        if (state.in_matchmaking_queue) {
            matchmakingBtn.innerText = "Leave Queue";
            matchmakingBtn.style.background = "var(--neon-purple)";
            queueStatus.innerHTML = `<span class="status-active">SEARCHING FOR OPPONENT...</span>`;
        } else {
            matchmakingBtn.innerText = "Join Matchmaking Pool";
            matchmakingBtn.style.background = "";
            queueStatus.innerText = "Ready for automatic pairing?";
        }
    }

    if (scope === "meta" || scope === "all") {
        handleTournamentUI(state.tournament);
        if (state.tournament && state.tournament.active) {
            renderTournamentBracket(state.tournament);
        }
        updateMapStatusIndicators();
        updateSolvencyDashboard(state, overrideData);
    }

    if (scope === "solvency_override" && overrideData) {
        updateSolvencyDashboard(state, overrideData);
    }

    if (scope === "inventory" || scope === "all") {
        renderDeckManager(state);
    }

    // --- Phase Transitions ---
    const lobby = getEl("lobby-container");
    const combat = getEl("combat-container");
    const tourney = getEl("tournament-lobby-container");

    if (lobby && combat && tourney) {
        lobby.classList.add("hidden");
        combat.classList.add("hidden");
        tourney.classList.add("hidden");

        switch(state.phase) {
            case "Active":
                combat.classList.remove("hidden");
                break;
            case "TournamentLobby":
                tourney.classList.remove("hidden");
                break;
            default:
                lobby.classList.remove("hidden");
        }
    }

    // Initial wallet detection
    if (!userAddress) {
        const overlay = getEl("wallet-selector-overlay");
        if (overlay) overlay.classList.remove("hidden");
    }

    // PILLAR 3: Cyber-Jammer UI State.
    const cyberJammerEl = getEl("cyber-jammer-status");
    if (cyberJammerEl) {
        if (state.has_cyber_jammer) {
            cyberJammerEl.classList.remove("hidden");
        } else {
            cyberJammerEl.classList.add("hidden");
        }
    }

    // PILLAR 3: Heist Saboteur Progress.
    const jammerCount = state.heist_alarms_jammer_count || 0;
    const heistProgressEl = getEl("heist-saboteur-progress");
    const isHeistSaboteur = state.achievements && state.achievements.includes("HEIST_SABOTEUR");

    if (heistProgressEl) {
        if (jammerCount > 0 && !isHeistSaboteur) {
            heistProgressEl.classList.remove("hidden");
            const percent = Math.min(100, (jammerCount / 3) * 100);
            heistProgressEl.innerHTML = `
                <div class="font-size-0-7em text-warning mb-2 uppercase letter-spacing-1">SABOTEUR: ${jammerCount}/3</div>
                <div class="progress-bar" style="width: 80px; height: 3px;">
                    <div class="progress-fill" style="width: ${percent}%"></div>
                </div>`;
        } else {
            heistProgressEl.classList.add("hidden");
        }
    }

    // PILLAR 6: Push-Authority Beacon.
    // Cache the authoritative push to localStorage to support warm-boot restoration.
    if (scope === "all" && (state.phase === "Lobby" || state.phase === "Active") && userAddress) {
        const beaconData = {
            profile: state,
            local_player_index: state.local_player_index, // PILLAR 4: Maintain identity across refreshes
            vault_balance: state.faucet, // PILLAR 2: Synchronize with WASM export key
            maintenance_priority: state.maintenance_priority, // PILLAR 4: Critical Alert state preservation
            match_id: state.match_id, // PILLAR 3: Standardized identification persistence
            rewards: state.rewards,
            clubs: state.clubs,
            ts: Date.now()
        };
        localStorage.setItem("vbabes_state_beacon", JSON.stringify(beaconData));
    }
};

// ── Dev diagnostics: auto-report client errors to the server (no console paste needed) ──
(function installClientErrorReporter() {
    // Parse "file:line:col" out of a stack string; returns {src,line,col} or null.
    function parseStackFrame(stack) {
        if (!stack) return null;
        // Match "something.js:LINE:COL". \S+ may grab a leading "(" from "at (http://...)"
        // — trim it so the src is a clean URL/path.
        const m = String(stack).match(/\(?(\S+\.js):(\d+):(\d+)\)?/);
        if (!m) return null;
        return { src: m[1].replace(/^\(/, ""), line: parseInt(m[2], 10) || 0, col: parseInt(m[3], 10) || 0 };
    }
    function report(msg, src, line, col, stack) {
        try {
            // Fallback: if the browser gave us no line (lineno 0 / empty src),
            // extract the real file:line:col from the stack trace.
            if ((!line || line === 0 || !src) && stack) {
                const fr = parseStackFrame(stack);
                if (fr) {
                    if (!src) src = fr.src;
                    if (!line || line === 0) line = fr.line;
                    if (!col || col === 0) col = fr.col;
                }
            }
            const wallet = (typeof userAddress !== "undefined" && userAddress) ? String(userAddress) : "";
            fetch("/api/client-error", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ msg: String(msg || ""), src: src || "", line: line || 0, col: col || 0, stack: stack || "", wallet: wallet }),
                keepalive: true
            }).catch(() => {});
        } catch (_) {}
    }
    window.addEventListener("error", function (e) {
        const er = e.error || {};
        report(e.message, e.filename, e.lineno, e.colno, er.stack || (e.message + ""));
    });
    window.addEventListener("unhandledrejection", function (e) {
        const r = e.reason || {};
        const stack = r.stack || (typeof r === "string" ? r : "");
        // Parse the real source/line from the rejection stack instead of hardcoding 0.
        const fr = parseStackFrame(stack);
        const src = (fr && fr.src) ? fr.src : ((stack || "").split("\n")[0]) || "unhandledrejection";
        const line = fr ? fr.line : 0;
        const col = fr ? fr.col : 0;
        report(r.message || r, src, line, col, stack);
    });
})();
