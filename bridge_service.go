//go:build js && wasm

package main

import (
	"syscall/js"
)

// bridge_service.go acts as the explicit IPC (Inter-Process Communication) 
// translation protocol between the browser's JavaScript environment and 
// the running WASM memory instance.
// PILLAR 4: Deterministic Sync.

func registerWasmHooks() {
	// 1. Identity & Wallet
	js.Global().Set("connectWallet", js.FuncOf(connectWallet))
	js.Global().Set("disconnectWallet", js.FuncOf(disconnectWallet))
	js.Global().Set("toggleNetwork", js.FuncOf(toggleNetwork))
	js.Global().Set("SetAvatar", js.FuncOf(SetAvatar))
	js.Global().Set("SendReward", js.FuncOf(SendReward))
	js.Global().Set("SyncFullProfile", js.FuncOf(SyncFullProfile))
	js.Global().Set("SyncPortfolio", js.FuncOf(SyncPortfolio))
	js.Global().Set("TriggerManualSync", js.FuncOf(TriggerManualSync))
	js.Global().Set("SetInMatchmakingQueue", js.FuncOf(SetInMatchmakingQueue))
	js.Global().Set("SyncSolvency", js.FuncOf(SyncSolvency))
	
	// 2. Battle & Mechanics
	js.Global().Set("StartMatch", js.FuncOf(StartMatch))
	js.Global().Set("PlaceCard", js.FuncOf(PlaceCard))
	js.Global().Set("SyncMove", js.FuncOf(SyncMove))
	js.Global().Set("SyncMatchMetadata", js.FuncOf(SyncMatchMetadata))
	js.Global().Set("SyncOpponentDeck", js.FuncOf(SyncOpponentDeck))
	js.Global().Set("SyncOpponentProfile", js.FuncOf(SyncOpponentProfile))
	js.Global().Set("SyncOpponentWanted", js.FuncOf(SyncOpponentWanted))
	js.Global().Set("SetLocalPlayerIndex", js.FuncOf(SetLocalPlayerIndex))
	js.Global().Set("initiateSuddenDeath", js.FuncOf(initiateSuddenDeath))
	js.Global().Set("ApplyArtifactToBoard", js.FuncOf(ApplyArtifactToBoard))
	
	// 3. State Exporter (Camera)
	js.Global().Set("GetGameState", js.FuncOf(GetGameState))
	js.Global().Set("SetPhase", js.FuncOf(SetPhase))
	js.Global().Set("ResetGame", js.FuncOf(ResetGame))
	js.Global().Set("SyncRules", js.FuncOf(SyncRules))
	js.Global().Set("SyncRewards", js.FuncOf(SyncRewards))
	js.Global().Set("SyncVaultBalance", js.FuncOf(SyncVaultBalance))
	js.Global().Set("SyncClubs", js.FuncOf(SyncClubs))
	js.Global().Set("ToggleRule", js.FuncOf(ToggleRule))
	
	// 4. Intelligence, HUD & Meta
	js.Global().Set("SyncLatency", js.FuncOf(SyncLatency))
	js.Global().Set("PushReplayFrame", js.FuncOf(PushReplayFrame))
	js.Global().Set("CompleteRecovery", js.FuncOf(CompleteRecovery))
	js.Global().Set("SyncServerLoad", js.FuncOf(SyncServerLoad))
	js.Global().Set("SyncPlayerStats", js.FuncOf(SyncPlayerStats))
	js.Global().Set("SyncPlaystyle", js.FuncOf(SyncPlaystyle))
	js.Global().Set("GetLevelLabelForDisplay", js.FuncOf(GetLevelLabelForDisplay))
	js.Global().Set("GetServerLoadColor", js.FuncOf(GetServerLoadColor))
	js.Global().Set("GetTierInfo", js.FuncOf(GetTierInfo))
	js.Global().Set("ToggleLeaderboard", js.FuncOf(ToggleLeaderboard))
	js.Global().Set("GetTournamentArchiveBadge", js.FuncOf(GetTournamentArchiveBadge))
	js.Global().Set("SyncTournament", js.FuncOf(SyncTournament))

	// 5. Inventory & Assets
	js.Global().Set("AddToDeck", js.FuncOf(AddToDeck))
	js.Global().Set("RemoveFromDeck", js.FuncOf(RemoveFromDeck))
	js.Global().Set("AutoBuildDeck", js.FuncOf(AutoBuildDeck))
	js.Global().Set("SelectDeck", js.FuncOf(SelectDeck))
	js.Global().Set("ImportARC72Card", js.FuncOf(ImportARC72Card))
	js.Global().Set("SyncCardMetadata", js.FuncOf(SyncCardMetadata))

	// 6. Audio & Visual System Settings
	js.Global().Set("SetMasterVolume", js.FuncOf(SetMasterVolume))
	js.Global().Set("SetMusicVolume", js.FuncOf(SetMusicVolume))
	js.Global().Set("SetSfxVolume", js.FuncOf(SetSfxVolume))
	js.Global().Set("PlaySelectSound", js.FuncOf(PlaySelectSound))
	js.Global().Set("SetAssetBase", js.FuncOf(SetAssetBase))
	js.Global().Set("SetApiBase", js.FuncOf(SetApiBase))
	
	// 7. System Overrides
	js.Global().Set("SetAdminState", js.FuncOf(SetAdminState))
	js.Global().Set("SetMaintenanceState", js.FuncOf(SetMaintenanceState))
	js.Global().Set("SetTestingMode", js.FuncOf(SetTestingMode))
	js.Global().Set("SetHardMode", js.FuncOf(SetHardMode))
	js.Global().Set("ForceActive", js.FuncOf(ForceActive))
	js.Global().Set("SetPlayerReady", js.FuncOf(SetPlayerReady))
}

// handleJSCallback wraps the execution of Go logic triggered by a browser event.
func handleJSCallback(this js.Value, args []js.Value, logic func([]js.Value) interface{}) interface{} {
	// Optimization: This allows us to yield to the JS event loop to prevent UI freezing
	// during intensive isomorphic calculations.
	return logic(args)
}
