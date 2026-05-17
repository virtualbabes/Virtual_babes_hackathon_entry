//go:build !js || !wasm

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// unlockAchievement checks if a player already has an achievement; if not, it unlocks it and notifies them.
func (l *Lobby) unlockAchievement(wallet, id string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.unlockAchievementLocked(wallet, id)
}

// unlockAchievementLocked checks if a player already has an achievement, assuming the lock is held.
func (l *Lobby) unlockAchievementLocked(wallet, id string) {
	targetWallet := strings.ToLower(wallet)
	l.ensurePlayerStatsMapsInitialized(targetWallet)
	stats, exists := l.leaderboard[targetWallet]
	if !exists {
		return
	}

	// Prevention: Ensure achievement is only granted once
	for _, a := range stats.Achievements {
		if a == id {
			return
		}
	}

	// Unlock trophy
	stats.Achievements = append(stats.Achievements, id)
	// Update reputation to reflect the social impact of the new achievement
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[targetWallet] = stats

	l.logAdminAuditLocked("ACHIEVEMENT_UNLOCKED", targetWallet, id)

	// Notify all client sessions associated with this wallet
	msg, _ := json.Marshal(map[string]string{
		"text": fmt.Sprintf("🏆 <b>TROPHY UNLOCKED:</b> %s", strings.ReplaceAll(id, "_", " ")),
	})

	for cid, w := range l.wallets {
		if strings.ToLower(w) == wallet {
			l.sendToClientLocked(cid, Envelope{
				Type:    "admin_notification",
				FromID:  "SERVER",
				Payload: msg,
			})
		}
	}

	// Broadcast lobby update using the Locked snapshot pattern
	msg = l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()
}
