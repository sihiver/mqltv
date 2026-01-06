package handlers

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"iptv-panel/database"
	"iptv-panel/models"
	"iptv-panel/parser"
	"iptv-panel/streaming"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// GetPlaylists returns all playlists
func GetPlaylists(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
		SELECT p.id, p.name, p.url, p.type, p.created_at, p.updated_at, 
		       COUNT(c.id) as channel_count
		FROM playlists p
		LEFT JOIN channels c ON p.id = c.playlist_id
		GROUP BY p.id, p.name, p.url, p.type, p.created_at, p.updated_at
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var playlists []models.Playlist
	for rows.Next() {
		var p models.Playlist
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.Type, &p.CreatedAt, &p.UpdatedAt, &p.ChannelCount); err != nil {
			continue
		}
		playlists = append(playlists, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": playlists,
	})
}

// ImportPlaylist imports M3U playlist
func ImportPlaylist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse M3U
	channels, err := parser.ParseM3UURL(req.URL)
	if err != nil {
		http.Error(w, "Failed to parse M3U: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Insert playlist
	result, err := tx.Exec("INSERT INTO playlists (name, url, type) VALUES (?, ?, ?)", req.Name, req.URL, "m3u")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	playlistID, _ := result.LastInsertId()

	// Insert channels
	for _, ch := range channels {
		_, err := tx.Exec("INSERT INTO channels (playlist_id, name, url, logo, group_name) VALUES (?, ?, ?, ?, ?)",
			playlistID, ch.Name, ch.URL, ch.Logo, ch.Group)
		if err != nil {
			log.Printf("Failed to insert channel: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"playlist_id": playlistID,
			"channels":    len(channels),
		},
		"message": "Playlist imported successfully",
	})
}

// CreatePlaylist creates a new manual playlist (without M3U import)
func CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Type string `json:"type"` // "manual", "m3u", or "relay"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Playlist name is required", http.StatusBadRequest)
		return
	}

	// Default type to "manual" if not specified
	if req.Type == "" {
		req.Type = "manual"
	}

	// Validate type
	if req.Type != "manual" && req.Type != "m3u" && req.Type != "relay" {
		http.Error(w, "Invalid playlist type. Must be 'manual', 'm3u', or 'relay'", http.StatusBadRequest)
		return
	}

	// Insert playlist
	result, err := database.DB.Exec(
		"INSERT INTO playlists (name, url, type, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		req.Name, "", req.Type, time.Now(), time.Now())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to create playlist: " + err.Error(),
		})
		return
	}

	playlistID, _ := result.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"id":   playlistID,
			"name": req.Name,
			"type": req.Type,
		},
		"message": "Playlist created successfully",
	})
}

// GetChannels returns channels for a playlist
func GetChannels(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	rows, err := database.DB.Query(`
		SELECT id, playlist_id, name, url, logo, group_name, active, created_at 
		FROM channels 
		WHERE playlist_id = ?
	`, playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		var c models.Channel
		if err := rows.Scan(&c.ID, &c.PlaylistID, &c.Name, &c.URL, &c.Logo, &c.Group,
			&c.Active, &c.CreatedAt); err != nil {
			continue
		}
		channels = append(channels, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": channels,
	})
}

// RefreshPlaylist re-imports playlist from the same URL
func RefreshPlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	// Get playlist URL
	var playlistURL string
	err := database.DB.QueryRow("SELECT url FROM playlists WHERE id = ?", playlistID).Scan(&playlistURL)
	if err != nil {
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}

	// Parse M3U from URL
	channels, err := parser.ParseM3UURL(playlistURL)
	if err != nil {
		http.Error(w, "Failed to parse M3U: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete old channels
	_, err = tx.Exec("DELETE FROM channels WHERE playlist_id = ?", playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Insert new channels
	channelCount := 0
	for _, ch := range channels {
		_, err := tx.Exec("INSERT INTO channels (playlist_id, name, url, logo, group_name) VALUES (?, ?, ?, ?, ?)",
			playlistID, ch.Name, ch.URL, ch.Logo, ch.Group)
		if err != nil {
			log.Printf("Failed to insert channel: %v", err)
		} else {
			channelCount++
		}
	}

	// Update playlist timestamp
	_, err = tx.Exec("UPDATE playlists SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":           0,
		"message":        "Playlist refreshed successfully",
		"channels_count": channelCount,
	})
}

// UpdatePlaylist updates playlist name and URL
func UpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	_, err := database.DB.Exec("UPDATE playlists SET name = ?, url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		req.Name, req.URL, playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"message": "Playlist updated successfully",
	})
}

// DeletePlaylist deletes a playlist and its channels
func DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	// Delete channels first
	_, err := database.DB.Exec("DELETE FROM channels WHERE playlist_id = ?", playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Then delete the playlist
	_, err = database.DB.Exec("DELETE FROM playlists WHERE id = ?", playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"data":    map[string]bool{"success": true},
		"message": "Playlist deleted successfully",
	})
}

// GetRelays returns all relays
func GetRelays(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query("SELECT id, name, source_urls, output_path, active, created_at, updated_at FROM relays")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var relays []models.Relay
	for rows.Next() {
		var rel models.Relay
		if err := rows.Scan(&rel.ID, &rel.Name, &rel.SourceURLs, &rel.OutputPath, &rel.Active, &rel.CreatedAt, &rel.UpdatedAt); err != nil {
			continue
		}
		relays = append(relays, rel)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": relays,
	})
}

// CreateRelay creates a new relay configuration
func CreateRelay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string   `json:"name"`
		SourceURLs []string `json:"source_urls"`
		OutputPath string   `json:"output_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sourceURLsJSON, _ := json.Marshal(req.SourceURLs)

	result, err := database.DB.Exec("INSERT INTO relays (name, source_urls, output_path) VALUES (?, ?, ?)",
		req.Name, string(sourceURLsJSON), req.OutputPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	relayID, _ := result.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      relayID,
	})
}

// DeleteRelay deletes a relay
func DeleteRelay(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	relayID := vars["id"]

	_, err := database.DB.Exec("DELETE FROM relays WHERE id = ?", relayID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// StreamRelay handles relay streaming with FFmpeg (on-demand multi-client)
func StreamRelay(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]

	// Authenticate user
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	if username == "" || password == "" {
		http.Error(w, "Authentication required: username and password parameters missing", http.StatusUnauthorized)
		return
	}

	// Verify user credentials and check if active
	// Password from M3U URL is already MD5 hashed
	var userID int
	var isActive bool
	var expiresAt sql.NullTime

	err := database.DB.QueryRow(`
		SELECT id, is_active, expires_at 
		FROM users 
		WHERE username = ? AND password = ?
	`, username, password).Scan(&userID, &isActive, &expiresAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !isActive {
		ServeExpiredImage(w, r)
		return
	}

	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		ServeExpiredImage(w, r)
		return
	}

	var sourceURLs string
	err = database.DB.QueryRow("SELECT source_urls FROM relays WHERE output_path = ? AND active = 1", path).Scan(&sourceURLs)
	if err == sql.ErrNoRows {
		http.Error(w, "Relay not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var urls []string
	json.Unmarshal([]byte(sourceURLs), &urls)

	// Get channel ID from relay path (format: channel-{id})
	var channelID sql.NullInt64
	if strings.HasPrefix(path, "channel-") {
		if id, err := strconv.Atoi(strings.TrimPrefix(path, "channel-")); err == nil {
			channelID = sql.NullInt64{Int64: int64(id), Valid: true}
		}
	}

	// Track user connection
	var connectionID int64
	if channelID.Valid {
		result, err := database.DB.Exec(`
			INSERT INTO user_connections (user_id, channel_id, ip_address, connected_at) 
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		`, userID, channelID.Int64, r.RemoteAddr)
		if err == nil {
			connectionID, _ = result.LastInsertId()
		}
	}

	// Defer closing connection
	defer func() {
		if connectionID > 0 {
			database.DB.Exec("UPDATE user_connections SET disconnected_at = CURRENT_TIMESTAMP WHERE id = ?", connectionID)
		}
	}()

	// Use FFmpeg manager for better compatibility and transcoding
	ffmpegManager := streaming.GetFFmpegManager()
	session := ffmpegManager.GetOrCreateFFmpegSession(path, urls, "mpegts")

	// Generate unique client ID
	clientID := fmt.Sprintf("%x", md5.Sum([]byte(r.RemoteAddr+r.UserAgent())))

	// Add client and get data channel
	dataChan, err := session.AddClient(clientID, r.RemoteAddr)
	if err != nil {
		http.Error(w, "Channel temporarily unavailable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer session.RemoveClient(clientID)

	// Set headers
	w.Header().Set("Content-Type", "video/MP2T")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Stream to this client
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send data to client
	for {
		select {
		case data, ok := <-dataChan:
			if !ok {
				// Channel closed
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// ExportM3U generates M3U playlist from database with panel proxy URLs
func ExportM3U(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	rows, err := database.DB.Query("SELECT id, name, url, logo, group_name FROM channels WHERE playlist_id = ? AND active = 1", playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/x-mpegurl")
	w.Header().Set("Content-Disposition", "attachment; filename=playlist.m3u")

	w.Write([]byte("#EXTM3U\n"))

	baseURL := publicBaseURL(r)

	for rows.Next() {
		var channelID int
		var name, url, logo, group string
		if err := rows.Scan(&channelID, &name, &url, &logo, &group); err != nil {
			continue
		}

		info := "#EXTINF:-1"
		if logo != "" {
			info += " tvg-logo=\"" + logo + "\""
		}
		if group != "" {
			info += " group-title=\"" + group + "\""
		}
		info += "," + name + "\n"

		// Export URL proxy panel via FFmpeg, bukan URL asli provider
		proxyURL := fmt.Sprintf("%s/api/proxy/channel/%d", baseURL, channelID)

		w.Write([]byte(info))
		w.Write([]byte(proxyURL + "\n"))
	}
}

// UpdateChannelStatus toggles channel active status
func UpdateChannelStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["id"]

	var active int
	err := database.DB.QueryRow("SELECT active FROM channels WHERE id = ?", channelID).Scan(&active)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newActive := 1
	if active == 1 {
		newActive = 0
	}

	_, err = database.DB.Exec("UPDATE channels SET active = ? WHERE id = ?", newActive, channelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"message": "Channel status updated",
		"data": map[string]interface{}{
			"success": true,
			"active":  newActive == 1,
		},
	})
}

// DeleteChannel deletes a channel by ID
func DeleteChannel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["id"]

	result, err := database.DB.Exec("DELETE FROM channels WHERE id = ?", channelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"data":    map[string]bool{"success": true},
		"message": "Channel deleted successfully",
	})
}

// BatchDeleteChannels deletes multiple channels by IDs
func BatchDeleteChannels(w http.ResponseWriter, r *http.Request) {
	var request struct {
		IDs        []int  `json:"ids"`
		Category   string `json:"category"`
		PlaylistID int    `json:"playlist_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Invalid request body",
		})
		return
	}

	var query string
	var args []interface{}
	var rowsAffected int64

	// Delete by IDs (new method)
	if len(request.IDs) > 0 {
		placeholders := make([]string, len(request.IDs))
		for i, id := range request.IDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query = fmt.Sprintf("DELETE FROM channels WHERE id IN (%s)", strings.Join(placeholders, ","))
	} else if request.Category != "" {
		// Delete by category (legacy method)
		query = "DELETE FROM channels WHERE group_name = ?"
		args = []interface{}{request.Category}

		if request.PlaylistID > 0 {
			query += " AND playlist_id = ?"
			args = append(args, request.PlaylistID)
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Either IDs or category is required",
		})
		return
	}

	result, err := database.DB.Exec(query, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to delete channels",
		})
		return
	}

	rowsAffected, _ = result.RowsAffected()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"success": true,
			"deleted": rowsAffected,
		},
		"message": fmt.Sprintf("Deleted %d channels", rowsAffected),
	})
}

// RenameChannelCategory renames a category (group_name) for all matching channels.
func RenameChannelCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldName    string `json:"old_name"`
		NewName    string `json:"new_name"`
		PlaylistID int    `json:"playlist_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Invalid request body",
		})
		return
	}

	oldName := strings.TrimSpace(req.OldName)
	newName := strings.TrimSpace(req.NewName)

	if oldName == "" || newName == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "old_name and new_name are required",
		})
		return
	}

	if oldName == newName {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"success": true,
				"updated": 0,
			},
			"message": "No changes",
		})
		return
	}

	query := "UPDATE channels SET group_name = ? WHERE group_name = ?"
	args := []interface{}{newName, oldName}
	if req.PlaylistID > 0 {
		query += " AND playlist_id = ?"
		args = append(args, req.PlaylistID)
	}

	result, err := database.DB.Exec(query, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to rename category",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"success":  true,
			"updated":  rowsAffected,
			"old_name": oldName,
			"new_name": newName,
		},
		"message": fmt.Sprintf("Renamed %d channels", rowsAffected),
	})
}

// SearchChannels searches channels by name
func SearchChannels(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	var rows *sql.Rows
	var err error

	if query == "" {
		// If no query, return all active channels with playlist info
		rows, err = database.DB.Query(`
			SELECT c.id, c.playlist_id, c.name, c.url, c.logo, c.group_name, c.active, c.on_demand, c.created_at, p.name as playlist_name
			FROM channels c
			LEFT JOIN playlists p ON c.playlist_id = p.id
			WHERE c.active = 1 
			ORDER BY c.created_at DESC 
			LIMIT 5000
		`)
	} else {
		// If query provided, search by name
		rows, err = database.DB.Query(`
			SELECT c.id, c.playlist_id, c.name, c.url, c.logo, c.group_name, c.active, c.on_demand, c.created_at, p.name as playlist_name
			FROM channels c
			LEFT JOIN playlists p ON c.playlist_id = p.id
			WHERE c.name LIKE ? AND c.active = 1 
			ORDER BY c.created_at DESC 
			LIMIT 5000
		`, "%"+query+"%")
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var channels []map[string]interface{}
	for rows.Next() {
		var c models.Channel
		var playlistName sql.NullString
		if err := rows.Scan(&c.ID, &c.PlaylistID, &c.Name, &c.URL, &c.Logo, &c.Group, &c.Active, &c.OnDemand, &c.CreatedAt, &playlistName); err != nil {
			continue
		}

		channel := map[string]interface{}{
			"id":            c.ID,
			"playlist_id":   c.PlaylistID,
			"name":          c.Name,
			"url":           c.URL,
			"logo":          c.Logo,
			"category":      c.Group,
			"group_name":    c.Group,
			"enabled":       c.Active,
			"active":        c.Active,
			"on_demand":     c.OnDemand,
			"created_at":    c.CreatedAt,
			"playlist_name": "",
		}

		if playlistName.Valid {
			channel["playlist_name"] = playlistName.String
		}

		channels = append(channels, channel)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": channels,
	})
}

// CreateChannel creates a new channel
func CreateChannel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlaylistID int    `json:"playlist_id"`
		Name       string `json:"name"`
		URL        string `json:"url"`
		Logo       string `json:"logo"`
		GroupName  string `json:"group_name"`
		OnDemand   *bool  `json:"on_demand"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	if req.Name == "" || req.URL == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Name and URL are required",
		})
		return
	}

	// Default on_demand to true if not specified
	onDemand := 1
	if req.OnDemand != nil && !*req.OnDemand {
		onDemand = 0
	}

	result, err := database.DB.Exec(
		"INSERT INTO channels (playlist_id, name, url, logo, group_name, active, on_demand) VALUES (?, ?, ?, ?, ?, 1, ?)",
		req.PlaylistID, req.Name, req.URL, req.Logo, req.GroupName, onDemand,
	)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Failed to create channel: " + err.Error(),
		})
		return
	}

	channelID, _ := result.LastInsertId()

	// Get the created channel with playlist info
	var c models.Channel
	var playlistName sql.NullString
	err = database.DB.QueryRow(`
		SELECT c.id, c.playlist_id, c.name, c.url, c.logo, c.group_name, c.active, c.on_demand, c.created_at, p.name as playlist_name
		FROM channels c
		LEFT JOIN playlists p ON c.playlist_id = p.id
		WHERE c.id = ?
	`, channelID).Scan(&c.ID, &c.PlaylistID, &c.Name, &c.URL, &c.Logo, &c.Group, &c.Active, &c.OnDemand, &c.CreatedAt, &playlistName)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Channel created but failed to retrieve: " + err.Error(),
		})
		return
	}

	channel := map[string]interface{}{
		"id":            c.ID,
		"playlist_id":   c.PlaylistID,
		"name":          c.Name,
		"url":           c.URL,
		"logo":          c.Logo,
		"category":      c.Group,
		"group_name":    c.Group,
		"enabled":       c.Active,
		"active":        c.Active,
		"on_demand":     c.OnDemand,
		"created_at":    c.CreatedAt,
		"playlist_name": "",
	}

	if playlistName.Valid {
		channel["playlist_name"] = playlistName.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"data":    channel,
		"message": "Channel created successfully",
	})
}

// UpdateChannel updates an existing channel
func UpdateChannel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["id"]

	var req struct {
		Name      string `json:"name"`
		URL       string `json:"url"`
		Logo      string `json:"logo"`
		GroupName string `json:"group_name"`
		OnDemand  *bool  `json:"on_demand"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	if req.Name == "" || req.URL == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Name and URL are required",
		})
		return
	}

	// Build update query
	if req.OnDemand != nil {
		onDemand := 0
		if *req.OnDemand {
			onDemand = 1
		}
		_, err := database.DB.Exec(
			"UPDATE channels SET name = ?, url = ?, logo = ?, group_name = ?, on_demand = ? WHERE id = ?",
			req.Name, req.URL, req.Logo, req.GroupName, onDemand, channelID,
		)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    1,
				"message": "Failed to update channel: " + err.Error(),
			})
			return
		}
	} else {
		_, err := database.DB.Exec(
			"UPDATE channels SET name = ?, url = ?, logo = ?, group_name = ? WHERE id = ?",
			req.Name, req.URL, req.Logo, req.GroupName, channelID,
		)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    1,
				"message": "Failed to update channel: " + err.Error(),
			})
			return
		}
	}

	// Get the updated channel with playlist info
	var c models.Channel
	var playlistName sql.NullString
	err := database.DB.QueryRow(`
		SELECT c.id, c.playlist_id, c.name, c.url, c.logo, c.group_name, c.active, c.on_demand, c.created_at, p.name as playlist_name
		FROM channels c
		LEFT JOIN playlists p ON c.playlist_id = p.id
		WHERE c.id = ?
	`, channelID).Scan(&c.ID, &c.PlaylistID, &c.Name, &c.URL, &c.Logo, &c.Group, &c.Active, &c.OnDemand, &c.CreatedAt, &playlistName)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Channel updated but failed to retrieve: " + err.Error(),
		})
		return
	}

	channel := map[string]interface{}{
		"id":            c.ID,
		"playlist_id":   c.PlaylistID,
		"name":          c.Name,
		"url":           c.URL,
		"logo":          c.Logo,
		"category":      c.Group,
		"group_name":    c.Group,
		"enabled":       c.Active,
		"active":        c.Active,
		"on_demand":     c.OnDemand,
		"created_at":    c.CreatedAt,
		"playlist_name": "",
	}

	if playlistName.Valid {
		channel["playlist_name"] = playlistName.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"data":    channel,
		"message": "Channel updated successfully",
	})
}

// GetStats returns dashboard statistics
func GetStats(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		TotalPlaylists int `json:"total_playlists"`
		TotalChannels  int `json:"total_channels"`
		ActiveChannels int `json:"active_channels"`
		TotalRelays    int `json:"total_relays"`
	}

	database.DB.QueryRow("SELECT COUNT(*) FROM playlists").Scan(&stats.TotalPlaylists)
	database.DB.QueryRow("SELECT COUNT(*) FROM channels").Scan(&stats.TotalChannels)
	// Count channels currently being watched (disconnected_at IS NULL)
	database.DB.QueryRow(`
		SELECT COUNT(DISTINCT channel_id) 
		FROM user_connections 
		WHERE channel_id IS NOT NULL 
		AND disconnected_at IS NULL
	`).Scan(&stats.ActiveChannels)
	database.DB.QueryRow("SELECT COUNT(*) FROM relays").Scan(&stats.TotalRelays)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": stats,
	})
}

// GetRecentlyWatchedChannels returns channels that were recently watched by users
func GetRecentlyWatchedChannels(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
		SELECT c.id, c.playlist_id, c.name, c.url, c.logo, c.group_name, c.active, c.created_at, 
		       p.name as playlist_name, MAX(uc.connected_at) as last_watched
		FROM user_connections uc
		INNER JOIN channels c ON uc.channel_id = c.id
		LEFT JOIN playlists p ON c.playlist_id = p.id
		WHERE uc.channel_id IS NOT NULL
		GROUP BY c.id, c.playlist_id, c.name, c.url, c.logo, c.group_name, c.active, c.created_at, p.name
		ORDER BY last_watched DESC
		LIMIT 10
	`)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to load recently watched: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var channels []map[string]interface{}
	for rows.Next() {
		var id, playlistID, active int
		var name, url, logo, groupName string
		var createdAt time.Time
		var playlistName sql.NullString
		var lastWatchedStr string

		if err := rows.Scan(&id, &playlistID, &name, &url, &logo, &groupName, &active, &createdAt, &playlistName, &lastWatchedStr); err != nil {
			log.Printf("Error scanning recently watched row: %v", err)
			continue
		}

		channel := map[string]interface{}{
			"id":            id,
			"playlist_id":   playlistID,
			"name":          name,
			"url":           url,
			"logo":          logo,
			"category":      groupName,
			"group_name":    groupName,
			"enabled":       active == 1,
			"active":        active == 1,
			"created_at":    createdAt,
			"playlist_name": "",
			"last_watched":  lastWatchedStr,
		}

		if playlistName.Valid {
			channel["playlist_name"] = playlistName.String
		}

		channels = append(channels, channel)
	}

	// Return empty array instead of nil
	if channels == nil {
		channels = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": channels,
	})
}

// GetActiveChannelsWithViewers returns currently active channels with viewer counts
func GetActiveChannelsWithViewers(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
		SELECT c.id, c.playlist_id, c.name, c.url, c.logo, c.group_name, c.active, c.created_at,
		       p.name as playlist_name, COUNT(uc.user_id) as viewer_count
		FROM user_connections uc
		INNER JOIN channels c ON uc.channel_id = c.id
		LEFT JOIN playlists p ON c.playlist_id = p.id
		WHERE uc.channel_id IS NOT NULL 
		  AND uc.disconnected_at IS NULL
		GROUP BY c.id, c.playlist_id, c.name, c.url, c.logo, c.group_name, c.active, c.created_at, p.name
		ORDER BY viewer_count DESC
	`)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to load active channels: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var channels []map[string]interface{}
	for rows.Next() {
		var id, playlistID, active, viewerCount int
		var name, url, logo, groupName string
		var createdAt time.Time
		var playlistName sql.NullString

		if err := rows.Scan(&id, &playlistID, &name, &url, &logo, &groupName, &active, &createdAt, &playlistName, &viewerCount); err != nil {
			log.Printf("Error scanning active channel row: %v", err)
			continue
		}

		channel := map[string]interface{}{
			"id":            id,
			"playlist_id":   playlistID,
			"name":          name,
			"url":           url,
			"logo":          logo,
			"category":      groupName,
			"group_name":    groupName,
			"enabled":       active == 1,
			"active":        active == 1,
			"created_at":    createdAt,
			"playlist_name": "",
			"viewer_count":  viewerCount,
		}

		if playlistName.Valid {
			channel["playlist_name"] = playlistName.String
		}

		channels = append(channels, channel)
	}

	// Return empty array instead of nil
	if channels == nil {
		channels = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": channels,
	})
}

// ProxyChannel proxies a specific channel stream via FFmpeg
func ProxyChannel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelIDStr := vars["id"]
	channelID, err := strconv.Atoi(channelIDStr)
	if err != nil {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		return
	}

	// Authenticate user
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	if username == "" || password == "" {
		http.Error(w, "Authentication required: username and password parameters missing", http.StatusUnauthorized)
		return
	}

	// Verify user credentials and check if active
	passwordHash := fmt.Sprintf("%x", md5.Sum([]byte(password)))
	var userID int
	var isActive bool
	var expiresAt sql.NullTime

	err = database.DB.QueryRow(`
		SELECT id, is_active, expires_at 
		FROM users 
		WHERE username = ? AND password = ?
	`, username, passwordHash).Scan(&userID, &isActive, &expiresAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !isActive {
		ServeExpiredImage(w, r)
		return
	}

	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		ServeExpiredImage(w, r)
		return
	}

	var url string
	var active int
	err = database.DB.QueryRow("SELECT url, active FROM channels WHERE id = ?", channelID).Scan(&url, &active)
	if err == sql.ErrNoRows {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if active == 0 {
		http.Error(w, "Channel is disabled", http.StatusForbidden)
		return
	}

	// Use FFmpeg manager for consistent proxying
	ffmpegManager := streaming.GetFFmpegManager()
	sessionID := fmt.Sprintf("channel_%d", channelID)
	session := ffmpegManager.GetOrCreateFFmpegSession(sessionID, []string{url}, "mpegts")

	clientID := fmt.Sprintf("%x", md5.Sum([]byte(r.RemoteAddr+r.UserAgent())))
	dataChan, err := session.AddClient(clientID, r.RemoteAddr)
	if err != nil {
		http.Error(w, "Channel temporarily unavailable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer session.RemoveClient(clientID)

	w.Header().Set("Content-Type", "video/MP2T")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case data, ok := <-dataChan:
			if !ok {
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// StreamRelayHLS serves HLS stream for relay via FFmpeg transcoding
func StreamRelayHLS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]

	// Authenticate user
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	if username == "" || password == "" {
		http.Error(w, "Authentication required: username and password parameters missing", http.StatusUnauthorized)
		return
	}

	// Verify user credentials and check if active
	passwordHash := fmt.Sprintf("%x", md5.Sum([]byte(password)))
	var userID int
	var isActive bool
	var expiresAt sql.NullTime

	err := database.DB.QueryRow(`
		SELECT id, is_active, expires_at 
		FROM users 
		WHERE username = ? AND password = ?
	`, username, passwordHash).Scan(&userID, &isActive, &expiresAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !isActive {
		ServeExpiredImage(w, r)
		return
	}

	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		ServeExpiredImage(w, r)
		return
	}

	var sourceURLs string
	err = database.DB.QueryRow("SELECT source_urls FROM relays WHERE output_path = ? AND active = 1", path).Scan(&sourceURLs)
	if err == sql.ErrNoRows {
		http.Error(w, "Relay not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var urls []string
	json.Unmarshal([]byte(sourceURLs), &urls)

	if len(urls) == 0 {
		http.Error(w, "No source URLs configured", http.StatusInternalServerError)
		return
	}

	// Use FFmpeg to transcode to HLS format
	ffmpegManager := streaming.GetFFmpegManager()
	sessionID := path + "_hls"
	session := ffmpegManager.GetOrCreateFFmpegSession(sessionID, urls, "hls")

	clientID := fmt.Sprintf("%x", md5.Sum([]byte(r.RemoteAddr+r.UserAgent())))
	dataChan, err := session.AddClient(clientID, r.RemoteAddr)
	if err != nil {
		http.Error(w, "Channel temporarily unavailable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer session.RemoveClient(clientID)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case data, ok := <-dataChan:
			if !ok {
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// StreamRelayHLSSegment serves HLS segments (currently redirects to source)
func StreamRelayHLSSegment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]
	segment := vars["segment"]

	// Get relay source URLs
	var sourceURLs string
	err := database.DB.QueryRow("SELECT source_urls FROM relays WHERE output_path = ? AND active = 1", path).Scan(&sourceURLs)
	if err != nil {
		http.Error(w, "Relay not found", http.StatusNotFound)
		return
	}

	var urls []string
	json.Unmarshal([]byte(sourceURLs), &urls)

	if len(urls) > 0 {
		// Try to construct segment URL from base URL
		baseURL := urls[0]
		lastSlash := len(baseURL) - 1
		for i := len(baseURL) - 1; i >= 0; i-- {
			if baseURL[i] == '/' {
				lastSlash = i
				break
			}
		}
		segmentURL := baseURL[:lastSlash+1] + segment

		// Proxy the segment
		resp, err := http.Get(segmentURL)
		if err != nil {
			http.Error(w, "Failed to fetch segment", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "video/MP2T")
		w.Header().Set("Cache-Control", "max-age=10")
		w.WriteHeader(resp.StatusCode)

		buffer := make([]byte, 32*1024)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				w.Write(buffer[:n])
			}
			if err != nil {
				break
			}
		}
		return
	}

	http.Error(w, "Segment not found", http.StatusNotFound)
}

// GetStreamStatus returns status of all active streams (FFmpeg sessions)
func GetStreamStatus(w http.ResponseWriter, r *http.Request) {
	ffmpegManager := streaming.GetFFmpegManager()
	sessions := ffmpegManager.GetAllSessions()

	status := make([]map[string]interface{}, 0, len(sessions))
	var totalBytesRead uint64
	var totalBytesWritten uint64

	for _, session := range sessions {
		stats := session.GetStats()
		// Only include active sessions with clients
		if session.IsActive() && session.GetClientCount() > 0 {
			status = append(status, stats)
			if bytesRead, ok := stats["bytes_read"].(uint64); ok {
				totalBytesRead += bytesRead
			}
			if bytesWritten, ok := stats["bytes_written"].(uint64); ok {
				totalBytesWritten += bytesWritten
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"total_streams":     len(status),
			"streams":           status,
			"total_bytes_read":  totalBytesRead,
			"total_bytes_write": totalBytesWritten,
		},
	})
}

// GetStreamStatusByID returns status of a specific stream
func GetStreamStatusByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamID := vars["id"]

	ffmpegManager := streaming.GetFFmpegManager()
	session := ffmpegManager.GetSession(streamID)

	if session == nil {
		http.Error(w, "Stream not found or inactive", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session.GetStats())
}

// ProxyChannelHLS serves channel as HLS stream via FFmpeg
func ProxyChannelHLS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelIDStr := vars["id"]
	channelID, err := strconv.Atoi(channelIDStr)
	if err != nil {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		return
	}

	// Authenticate user
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	if username == "" || password == "" {
		http.Error(w, "Authentication required: username and password parameters missing", http.StatusUnauthorized)
		return
	}

	// Verify user credentials and check if active
	passwordHash := fmt.Sprintf("%x", md5.Sum([]byte(password)))
	var userID int
	var isActive bool
	var expiresAt sql.NullTime

	err = database.DB.QueryRow(`
		SELECT id, is_active, expires_at 
		FROM users 
		WHERE username = ? AND password = ?
	`, username, passwordHash).Scan(&userID, &isActive, &expiresAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !isActive {
		ServeExpiredImage(w, r)
		return
	}

	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		ServeExpiredImage(w, r)
		return
	}

	var url string
	var active int
	err = database.DB.QueryRow("SELECT url, active FROM channels WHERE id = ?", channelID).Scan(&url, &active)
	if err == sql.ErrNoRows {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if active == 0 {
		http.Error(w, "Channel is disabled", http.StatusForbidden)
		return
	}

	// Use FFmpeg to transcode to HLS format
	ffmpegManager := streaming.GetFFmpegManager()
	sessionID := fmt.Sprintf("channel_%d_hls", channelID)
	session := ffmpegManager.GetOrCreateFFmpegSession(sessionID, []string{url}, "hls")

	clientID := fmt.Sprintf("%x", md5.Sum([]byte(r.RemoteAddr+r.UserAgent())))
	dataChan, err := session.AddClient(clientID, r.RemoteAddr)
	if err != nil {
		http.Error(w, "Channel temporarily unavailable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer session.RemoveClient(clientID)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case data, ok := <-dataChan:
			if !ok {
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// SaveGeneratedPlaylist saves a generated M3U playlist to static/playlists directory
func SaveGeneratedPlaylist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create playlists directory if not exists
	playlistDir := "./generated_playlists"
	if err := os.MkdirAll(playlistDir, 0755); err != nil {
		http.Error(w, "Failed to create directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save file
	filePath := fmt.Sprintf("%s/%s", playlistDir, req.Filename)
	if err := os.WriteFile(filePath, []byte(req.Content), 0644); err != nil {
		http.Error(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return URL
	url := fmt.Sprintf("/generated_playlists/%s", req.Filename)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":     url,
		"success": true,
	})
}

// ServeUserPlaylist serves user playlist with short URL: /mql/{user}.m3u
func ServeUserPlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["user"]

	// Build file path
	filePath := fmt.Sprintf("./generated_playlists/playlist-%s.m3u", username)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Playlist not found. Please generate playlist first.", http.StatusNotFound)
		return
	}

	// Serve file
	w.Header().Set("Content-Type", "audio/x-mpegurl")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=playlist-%s.m3u", username))
	http.ServeFile(w, r, filePath)
}

// AdminPreviewChannel - Admin preview tanpa user authentication
func AdminPreviewChannel(w http.ResponseWriter, r *http.Request) {
	// Get channel ID
	vars := mux.Vars(r)
	channelIDStr := vars["id"]
	channelID, err := strconv.Atoi(channelIDStr)
	if err != nil {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		return
	}

	// Allow subsequent playlist/segment requests to be proxied via ?url=
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		// Get channel info
		err = database.DB.QueryRow("SELECT url FROM channels WHERE id = ?", channelID).Scan(&targetURL)
		if err != nil {
			http.Error(w, "Channel not found", http.StatusNotFound)
			return
		}
	}

	parsedTarget, err := url.Parse(targetURL)
	if err != nil || parsedTarget.Scheme == "" || parsedTarget.Host == "" {
		http.Error(w, "Invalid target URL", http.StatusBadRequest)
		return
	}
	if parsedTarget.Scheme != "http" && parsedTarget.Scheme != "https" {
		http.Error(w, "Unsupported URL scheme", http.StatusBadRequest)
		return
	}

	// Set CORS headers (dev + prod)
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Proxy the stream directly
	client := &http.Client{
		Timeout: 0,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers from original request
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch stream", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// If this is an HLS playlist, rewrite its URIs so the browser fetches everything through this endpoint.
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	looksLikePlaylist := strings.Contains(strings.ToLower(targetURL), ".m3u8") ||
		strings.Contains(contentType, "mpegurl") ||
		strings.Contains(contentType, "application/vnd.apple.mpegurl") ||
		strings.Contains(contentType, "application/x-mpegurl")

	if looksLikePlaylist {
		limited := io.LimitReader(resp.Body, 1024*1024) // 1MB max for playlist
		bodyBytes, err := io.ReadAll(limited)
		if err != nil {
			http.Error(w, "Failed to read playlist", http.StatusBadGateway)
			return
		}

		base := parsedTarget
		proxyBasePath := fmt.Sprintf("/api/channels/%d/preview", channelID)

		scanner := bufio.NewScanner(bytes.NewReader(bodyBytes))
		// Allow reasonably long URIs
		scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)

		var out strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				out.WriteString(line)
				out.WriteString("\n")
				continue
			}

			ref, err := url.Parse(trimmed)
			if err != nil {
				out.WriteString(line)
				out.WriteString("\n")
				continue
			}

			abs := base.ResolveReference(ref).String()
			out.WriteString(proxyBasePath)
			out.WriteString("?url=")
			out.WriteString(url.QueryEscape(abs))
			out.WriteString("\n")
		}
		if err := scanner.Err(); err != nil {
			http.Error(w, "Failed to parse playlist", http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.WriteHeader(resp.StatusCode)
		io.WriteString(w, out.String())
		return
	}

	// Copy response headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	// Stream the content
	io.Copy(w, resp.Body)
}

// AdminPreviewChannel allows admin to preview channel without user auth
