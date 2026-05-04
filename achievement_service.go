package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// unlockAchievement checks if a player already has an achievement; if not, it unlocks it and notifies them.
func (l *Lobby) unlockAchievement(wallet, id string) {
	l.mutex.Lock()
	wallet = strings.ToLower(wallet)
	stats, exists := l.leaderboard[wallet]
	if !exists {
		l.mutex.Unlock()
		return
	}

	// Check for existing
	for _, a := range stats.Achievements {
		if a == id {
			l.mutex.Unlock()
			return
		}
	}

	// Unlock trophy
	stats.Achievements = append(stats.Achievements, id)
	l.leaderboard[wallet] = stats
	l.mutex.Unlock()

	l.logAdminAudit("ACHIEVEMENT_UNLOCKED", wallet, id)

	// Notify all client sessions associated with this wallet
	msg, _ := json.Marshal(map[string]string{
		"text": fmt.Sprintf("🏆 <b>TROPHY UNLOCKED:</b> %s", strings.ReplaceAll(id, "_", " ")),
	})

	l.mutex.RLock()
	for cid, w := range l.wallets {
		if strings.ToLower(w) == wallet {
			l.sendToClient(cid, Envelope{
				Type:    "admin_notification",
				FromID:  "SERVER",
				Payload: msg,
			})
		}
	}
	l.mutex.RUnlock()

	// Broadcast lobby update to show new trophy counts/badges
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}
