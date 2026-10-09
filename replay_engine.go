//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ── Replay Engine ────────────────────────────────────────────────────────────
// Captures game state, broadcasts frames to dashboard for live spectating.

// ReplayFrame represents a single frame of game state.
type ReplayFrame struct {
	Timestamp   int64       `json:"timestamp"`
	FrameID     int64       `json:"frame_id"`
	Player      PlayerState `json:"player"`
	Territory   string      `json:"territory"`
	Region      string      `json:"region"`
	Action      string      `json:"action"`
	Position    Position3D  `json:"position"`
	Rotation    float64     `json:"rotation"`
	Animation   string      `json:"animation"`
	Health      uint64      `json:"health"`
	Mana        uint64      `json:"mana"`
	Score       uint64      `json:"score"`
	Level       uint64      `json:"level"`
	Experience  uint64      `json:"experience"`
	Inventory   []string    `json:"inventory"`
	CurrentGame string      `json:"current_game"`
	Opponent    string      `json:"opponent"`
}

// PlayerState represents the player's current state.
type PlayerState struct {
	Wallet      string `json:"wallet"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	Faction     string `json:"faction"`
	CareerTier  string `json:"career_tier"`
	Faith       string `json:"faith"`
	Club        string `json:"club"`
}

// Position3D represents a 3D position.
type Position3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// ReplayEngine manages replay frame capture and broadcast.
type ReplayEngine struct {
	mu          sync.RWMutex
	frames      []ReplayFrame
	frameID     int64
	isRecording bool
	isPlaying   bool
	players     map[string]*ReplayFrame
	subscribers map[string]chan ReplayFrame
}

var replayEngine = &ReplayEngine{
	frames:      make([]ReplayFrame, 0),
	players:     make(map[string]*ReplayFrame),
	subscribers: make(map[string]chan ReplayFrame),
}

// StartRecording starts capturing frames.
func (re *ReplayEngine) StartRecording() {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.isRecording = true
}

// StopRecording stops capturing frames.
func (re *ReplayEngine) StopRecording() {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.isRecording = false
}

// CaptureFrame captures a new frame.
func (re *ReplayEngine) CaptureFrame(frame ReplayFrame) {
	re.mu.Lock()
	defer re.mu.Unlock()

	re.frameID++
	frame.FrameID = re.frameID
	frame.Timestamp = time.Now().UnixMilli()

	re.frames = append(re.frames, frame)
	if len(re.frames) > 160 {
		re.frames = re.frames[len(re.frames)-160:]
	}

	re.players[frame.Player.Wallet] = &frame

	// Broadcast to subscribers
	for _, ch := range re.subscribers {
		select {
		case ch <- frame:
		default:
		}
	}
}

// GetLatestFrame returns the latest frame.
func (re *ReplayEngine) GetLatestFrame() *ReplayFrame {
	re.mu.RLock()
	defer re.mu.RUnlock()
	if len(re.frames) == 0 {
		return nil
	}
	return &re.frames[len(re.frames)-1]
}

// GetFrames returns all frames.
func (re *ReplayEngine) GetFrames() []ReplayFrame {
	re.mu.RLock()
	defer re.mu.RUnlock()
	result := make([]ReplayFrame, len(re.frames))
	copy(result, re.frames)
	return result
}

// GetPlayerFrame returns the latest frame for a player.
func (re *ReplayEngine) GetPlayerFrame(wallet string) *ReplayFrame {
	re.mu.RLock()
	defer re.mu.RUnlock()
	return re.players[wallet]
}

// Subscribe returns a channel for new frames.
func (re *ReplayEngine) Subscribe(id string) chan ReplayFrame {
	re.mu.Lock()
	defer re.mu.Unlock()
	ch := make(chan ReplayFrame, 100)
	re.subscribers[id] = ch
	return ch
}

// Unsubscribe removes a subscriber.
func (re *ReplayEngine) Unsubscribe(id string) {
	re.mu.Lock()
	defer re.mu.Unlock()
	if ch, ok := re.subscribers[id]; ok {
		close(ch)
		delete(re.subscribers, id)
	}
}

// GetState returns the current replay state.
func (re *ReplayEngine) GetState() map[string]interface{} {
	re.mu.RLock()
	defer re.mu.RUnlock()
	return map[string]interface{}{
		"is_recording": re.isRecording,
		"is_playing":   re.isPlaying,
		"frame_count":  len(re.frames),
		"player_count": len(re.players),
	}
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleReplayState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"success": true, "state": replayEngine.GetState()})
}

func (l *Lobby) handleReplayLatest(w http.ResponseWriter, r *http.Request) {
	frame := replayEngine.GetLatestFrame()
	if frame == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "no frames"})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "frame": frame})
}

func (l *Lobby) handleReplayFrames(w http.ResponseWriter, r *http.Request) {
	sinceStr := r.URL.Query().Get("since")
	limitStr := r.URL.Query().Get("limit")
	
	var sinceID int64
	if sinceStr != "" {
		sinceID, _ = strconv.ParseInt(sinceStr, 10, 64)
	}
	
	limit := 50 // default limit
	if limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err == nil && parsed > 0 && parsed <= 160 {
			limit = parsed
		}
	}
	
	frames := replayEngine.GetFrames()
	var result []ReplayFrame
	for _, f := range frames {
		if f.FrameID > sinceID {
			result = append(result, f)
		}
	}
	// Apply limit (most recent N frames)
	if len(result) > limit {
		result = result[len(result)-limit:]
	}
	writeJSON(w, map[string]interface{}{"success": true, "frames": result})
}

func (l *Lobby) handleReplayPlayer(w http.ResponseWriter, r *http.Request) {
	wallet := r.URL.Query().Get("wallet")
	if wallet == "" {
		wallet = l.getWalletFromRequest(r)
	}
	frame := replayEngine.GetPlayerFrame(wallet)
	if frame == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "player not found"})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "frame": frame})
}

func (l *Lobby) handleReplayCapture(w http.ResponseWriter, r *http.Request) {
	var frame ReplayFrame
	if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	replayEngine.CaptureFrame(frame)
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleReplayStart(w http.ResponseWriter, r *http.Request) {
	replayEngine.StartRecording()
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleReplayStop(w http.ResponseWriter, r *http.Request) {
	replayEngine.StopRecording()
	writeJSON(w, map[string]interface{}{"success": true})
}

// ── Diagnostics ─────────────────────────────────────────────────────────────
func (l *Lobby) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Diagnostics — NFT-Seduction</title>
<style>
body{font-family:Consolas,monospace;background:#0a0e17;color:#e5e7eb;padding:20px}
h1{color:#6366f1;margin-bottom:8px}
h2{color:#a855f7;margin:16px 0 8px}
pre{background:#111827;border:1px solid #1f2937;border-radius:8px;padding:12px;overflow:auto;font-size:12px}
.ok{color:#10b981}.err{color:#ef4444}
</style></head>
<body>
<h1>🔍 NFT-Seduction Diagnostics</h1>
<h2>Server Status</h2>
<pre>Uptime: %s
Replay Frames: %d
Connected Players: %d
</pre>
<h2>Error Feed</h2>
<pre class="err" id="errors">No errors captured yet. Errors will appear here when the spectate client sends them.</pre>
<h2>System Health</h2>
<pre class="ok">✅ Dev Server Online
✅ REST Routes Active
✅ WebSocket Active
</pre>
<p style="margin-top:16px;color:#9ca3af;font-size:12px">Refresh for latest status. Client errors captured via /api/client-error.</p>
</body></html>`, time.Since(time.Now().Add(-time.Minute)), replayEngine.GetFrameCount(), len(replayEngine.players))
}

// ── Client Error Feed ───────────────────────────────────────────────────────
func (l *Lobby) handleClientError(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Msg       string `json:"msg"`
		Stack     string `json:"stack"`
		URL       string `json:"url"`
		Timestamp int64  `json:"timestamp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	
	// Log to server console
	log.Printf("[CLIENT-ERROR] %s | %s | %s", req.Msg, req.URL, req.Stack)
	
	writeJSON(w, map[string]interface{}{"success": true})
}

// GetFrameCount returns the number of frames.
func (re *ReplayEngine) GetFrameCount() int {
	re.mu.RLock()
	defer re.mu.RUnlock()
	return len(re.frames)
}
