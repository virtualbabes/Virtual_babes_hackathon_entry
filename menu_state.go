//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"
)

// GetMenuState returns the authoritative menu/controller state as JSON.
func GetMenuState(this js.Value, args []js.Value) interface{} {
	Game.mutex.RLock()
	defer Game.mutex.RUnlock()

	data, err := json.Marshal(Game.MenuState)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// SyncMenuState ingests full menu state from JS (e.g., on beacon recovery).
func SyncMenuState(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	raw := args[0].String()
	var ms MenuState
	if err := json.Unmarshal([]byte(raw), &ms); err != nil {
		return false
	}
	Game.MenuState = ms
	return true
}

// SetMenuLayout updates grid type, shape, size, and grid config.
func SetMenuLayout(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	data := args[0].String()
	var layout struct {
		GridType     string `json:"grid_type"`
		DefaultShape string `json:"default_shape"`
		DefaultSize  string `json:"default_size"`
		GridCols     int    `json:"grid_cols"`
		GridGap      string `json:"grid_gap"`
		GridRadius   string `json:"grid_radius"`
		GridRotation string `json:"grid_rotation"`
	}
	if err := json.Unmarshal([]byte(data), &layout); err != nil {
		return false
	}

	if layout.GridType != "" {
		Game.MenuState.GridType = layout.GridType
	}
	if layout.DefaultShape != "" {
		Game.MenuState.DefaultShape = layout.DefaultShape
	}
	if layout.DefaultSize != "" {
		Game.MenuState.DefaultSize = layout.DefaultSize
	}
	if layout.GridCols > 0 {
		Game.MenuState.GridCols = layout.GridCols
	}
	if layout.GridGap != "" {
		Game.MenuState.GridGap = layout.GridGap
	}
	if layout.GridRadius != "" {
		Game.MenuState.GridRadius = layout.GridRadius
	}
	if layout.GridRotation != "" {
		Game.MenuState.GridRotation = layout.GridRotation
	}
	return true
}

// SetControllerType sets the active controller type (xbox/playstation/nintendo/keyboard/touch_tilt).
func SetControllerType(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	Game.MenuState.ControllerType = args[0].String()
	return true
}

// SetControllerFocus sets the current focus index for controller navigation.
func SetControllerFocus(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	Game.MenuState.FocusIndex = args[0].Int()
	return true
}

// HandleControllerInput processes a controller action (up/down/left/right/select/back).
// Returns the new focus index.
func HandleControllerInput(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return -1
	}

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	action := args[0].String()
	count := len(Game.MenuState.StarredItems)
	if count == 0 {
		return Game.MenuState.FocusIndex
	}

	switch action {
	case "right":
		if Game.MenuState.GridType == "circle" || Game.MenuState.GridType == "linear-h" || Game.MenuState.GridType == "arc" {
			Game.MenuState.FocusIndex = (Game.MenuState.FocusIndex + 1) % count
		} else {
			cols := Game.MenuState.GridCols
			if cols < 1 {
				cols = 1
			}
			row := Game.MenuState.FocusIndex / cols
			col := Game.MenuState.FocusIndex % cols
			if col < cols-1 && row*cols+col+1 < count {
				Game.MenuState.FocusIndex = row*cols + col + 1
			}
		}
	case "left":
		if Game.MenuState.GridType == "circle" || Game.MenuState.GridType == "linear-h" || Game.MenuState.GridType == "arc" {
			Game.MenuState.FocusIndex = (Game.MenuState.FocusIndex - 1 + count) % count
		} else {
			cols := Game.MenuState.GridCols
			if cols < 1 {
				cols = 1
			}
			row := Game.MenuState.FocusIndex / cols
			col := Game.MenuState.FocusIndex % cols
			if col > 0 {
				Game.MenuState.FocusIndex = row*cols + col - 1
			}
		}
	case "down":
		if Game.MenuState.GridType == "linear-v" || Game.MenuState.GridType == "triangle" {
			if Game.MenuState.FocusIndex < count-1 {
				Game.MenuState.FocusIndex++
			}
		} else {
			cols := Game.MenuState.GridCols
			if cols < 1 {
				cols = 1
			}
			newIdx := Game.MenuState.FocusIndex + cols
			if newIdx < count {
				Game.MenuState.FocusIndex = newIdx
			}
		}
	case "up":
		if Game.MenuState.GridType == "linear-v" || Game.MenuState.GridType == "triangle" {
			if Game.MenuState.FocusIndex > 0 {
				Game.MenuState.FocusIndex--
			}
		} else {
			cols := Game.MenuState.GridCols
			if cols < 1 {
				cols = 1
			}
			newIdx := Game.MenuState.FocusIndex - cols
			if newIdx >= 0 {
				Game.MenuState.FocusIndex = newIdx
			}
		}
	case "select":
		// Focus index stays; UI handles activation
	case "back":
		// Focus index stays; UI handles back
	}

	return Game.MenuState.FocusIndex
}

// ToggleStarItem adds or removes a starred item.
func ToggleStarItem(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	data := args[0].String()
	var item StarredItem
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return false
	}

	// Check if already starred
	for i, s := range Game.MenuState.StarredItems {
		if s.WdTab == item.WdTab && s.WdSub == item.WdSub {
			// Remove
			Game.MenuState.StarredItems = append(Game.MenuState.StarredItems[:i], Game.MenuState.StarredItems[i+1:]...)
			return true
		}
	}

	// Add
	Game.MenuState.StarredItems = append(Game.MenuState.StarredItems, item)
	return true
}

// SetButtonOverrideWASM sets a per-button shape/size override.
func SetButtonOverrideWASM(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	buttonID := args[0].String()
	override := args[1].String()

	var bo ButtonOverride
	if err := json.Unmarshal([]byte(override), &bo); err != nil {
		return false
	}

	if Game.MenuState.ButtonOverrides == nil {
		Game.MenuState.ButtonOverrides = make(map[string]ButtonOverride)
	}
	if bo.Shape == "" && bo.Size == "" {
		delete(Game.MenuState.ButtonOverrides, buttonID)
	} else {
		Game.MenuState.ButtonOverrides[buttonID] = bo
	}
	return true
}
