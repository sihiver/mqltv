package handlers

import (
	"encoding/json"
	"iptv-panel/database"
	"iptv-panel/streaming"
	"log"
	"net/http"
	"os/exec"
	"strconv"
)

// GetSettings returns all settings grouped by category
func GetSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query("SELECT key, value, category FROM settings")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Failed to load settings: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	settings := map[string]map[string]interface{}{
		"system": {},
		"ffmpeg": {},
		"stream": {},
	}

	for rows.Next() {
		var key, value, category string
		if err := rows.Scan(&key, &value, &category); err != nil {
			continue
		}

		// Convert string values to appropriate types
		var convertedValue interface{} = value
		if value == "true" || value == "false" {
			convertedValue = value == "true"
		} else if num, err := strconv.Atoi(value); err == nil {
			convertedValue = num
		}

		if _, ok := settings[category]; ok {
			settings[category][key] = convertedValue
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": settings,
	})
}

// UpdateSettings updates settings by category
func UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Category string                 `json:"category"`
		Settings map[string]interface{} `json:"settings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	// Update each setting
	for key, value := range req.Settings {
		var strValue string
		switch v := value.(type) {
		case bool:
			strValue = strconv.FormatBool(v)
		case float64:
			strValue = strconv.Itoa(int(v))
		case string:
			strValue = v
		default:
			strValue = ""
		}

		_, err := database.DB.Exec(
			"UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ? AND category = ?",
			strValue, key, req.Category,
		)
		if err != nil {
			log.Printf("Failed to update setting %s: %v", key, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"message": "Settings updated successfully",
	})
}

// TestFFmpeg tests if FFmpeg is installed and working
func TestFFmpeg(w http.ResponseWriter, r *http.Request) {
	// Get FFmpeg path from settings
	var ffmpegPath string
	err := database.DB.QueryRow("SELECT value FROM settings WHERE key = 'ffmpeg_path'").Scan(&ffmpegPath)
	if err != nil {
		ffmpegPath = "/usr/bin/ffmpeg"
	}

	// Test FFmpeg
	cmd := exec.Command(ffmpegPath, "-version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "FFmpeg test failed: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"data":    string(output),
		"message": "FFmpeg is working correctly",
	})
}

// ClearHLSCache clears all HLS cache files
func ClearHLSCache(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("sh", "-c", "rm -rf hls_cache/*")
	err := cmd.Run()

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Failed to clear cache: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"message": "HLS cache cleared successfully",
	})
}

// RestartStreams stops all current FFmpeg sessions and recreates those that were running.
// Useful after changing ffmpeg_path or other runtime streaming settings.
func RestartStreams(w http.ResponseWriter, r *http.Request) {
	manager := streaming.GetFFmpegManager()
	sessions := manager.GetAllSessions()

	type sessionSnapshot struct {
		ID          string
		SourceURLs  []string
		Format      string
		OnDemand    bool
		WasActive   bool
		ClientCount int
	}

	snapshots := make([]sessionSnapshot, 0, len(sessions))
	for _, s := range sessions {
		snapshots = append(snapshots, sessionSnapshot{
			ID:          s.ID,
			SourceURLs:  append([]string(nil), s.SourceURLs...),
			Format:      s.OutputFormat,
			OnDemand:    s.IsOnDemand(),
			WasActive:   s.IsActive(),
			ClientCount: s.GetClientCount(),
		})
	}

	stopped := 0
	for _, snap := range snapshots {
		manager.DeleteSession(snap.ID)
		stopped++
	}

	restarted := 0
	for _, snap := range snapshots {
		// Only restart streams that were effectively in-use:
		// - had clients, or
		// - on-demand disabled (meant to keep running), or
		// - already active.
		if snap.ClientCount == 0 && snap.OnDemand && !snap.WasActive {
			continue
		}

		newSession := manager.GetOrCreateFFmpegSession(snap.ID, snap.SourceURLs, snap.Format)
		newSession.SetOnDemand(snap.OnDemand)
		go newSession.Start()
		restarted++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"stopped":   stopped,
			"restarted": restarted,
		},
		"message": "Streams restart triggered",
	})
}

// ResetTotalBandwidth resets the persisted total upload/download counters.
// Note: if streams are currently running, totals may start increasing again immediately.
func ResetTotalBandwidth(w http.ResponseWriter, r *http.Request) {
	manager := streaming.GetFFmpegManager()
	manager.ResetPersistedTotals()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"message": "Total bandwidth counters reset",
	})
}
