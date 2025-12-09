																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																																			package handlers

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"iptv-panel/database"
	"iptv-panel/models"
	"iptv-panel/parser"
	"iptv-panel/streaming"
	"log"
	"net/http"
	"os"
	"strconv"
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
	json.NewEncoder(w).Encode(playlists)
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
		"success":     true,
		"playlist_id": playlistID,
		"channels":    len(channels),
	})
}

// GetChannels returns channels for a playlist
func GetChannels(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	rows, err := database.DB.Query("SELECT id, playlist_id, name, url, logo, group_name, active, created_at FROM channels WHERE playlist_id = ?", playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		var c models.Channel
		if err := rows.Scan(&c.ID, &c.PlaylistID, &c.Name, &c.URL, &c.Logo, &c.Group, &c.Active, &c.CreatedAt); err != nil {
			continue
		}
		channels = append(channels, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
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
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
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
	json.NewEncoder(w).Encode(relays)
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

	// Use FFmpeg manager for better compatibility and transcoding
	ffmpegManager := streaming.GetFFmpegManager()
	session := ffmpegManager.GetOrCreateFFmpegSession(path, urls, "mpegts")

	// Generate unique client ID
	clientID := fmt.Sprintf("%x", md5.Sum([]byte(r.RemoteAddr+r.UserAgent())))

	// Add client and get data channel
	dataChan := session.AddClient(clientID, r.RemoteAddr)
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

	// Get host for proxy URLs
	scheme := "http"
	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}

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
		proxyURL := fmt.Sprintf("%s://%s/api/proxy/channel/%d", scheme, host, channelID)

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
		"success": true,
		"active":  newActive == 1,
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
		"success": true,
		"message": "Channel deleted successfully",
	})
}

// BatchDeleteChannels deletes multiple channels by category
func BatchDeleteChannels(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Category   string `json:"category"`
		PlaylistID int    `json:"playlist_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Category == "" {
		http.Error(w, "Category is required", http.StatusBadRequest)
		return
	}

	// Delete all channels in the category
	query := "DELETE FROM channels WHERE group_name = ?"
	args := []interface{}{request.Category}

	if request.PlaylistID > 0 {
		query += " AND playlist_id = ?"
		args = append(args, request.PlaylistID)
	}

	result, err := database.DB.Exec(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"deleted": rowsAffected,
		"message": fmt.Sprintf("Deleted %d channels from category '%s'", rowsAffected, request.Category),
	})
}

// SearchChannels searches channels by name
func SearchChannels(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	
	var rows *sql.Rows
	var err error
	
	if query == "" {
		// If no query, return all active channels
		rows, err = database.DB.Query("SELECT id, playlist_id, name, url, logo, group_name, active, created_at FROM channels WHERE active = 1 ORDER BY name LIMIT 5000")
	} else {
		// If query provided, search by name
		rows, err = database.DB.Query("SELECT id, playlist_id, name, url, logo, group_name, active, created_at FROM channels WHERE name LIKE ? AND active = 1 ORDER BY name LIMIT 5000",
			"%"+query+"%")
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		var c models.Channel
		if err := rows.Scan(&c.ID, &c.PlaylistID, &c.Name, &c.URL, &c.Logo, &c.Group, &c.Active, &c.CreatedAt); err != nil {
			continue
		}
		channels = append(channels, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
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
	database.DB.QueryRow("SELECT COUNT(*) FROM channels WHERE active = 1").Scan(&stats.ActiveChannels)
	database.DB.QueryRow("SELECT COUNT(*) FROM relays").Scan(&stats.TotalRelays)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
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
	dataChan := session.AddClient(clientID, r.RemoteAddr)
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
	dataChan := session.AddClient(clientID, r.RemoteAddr)
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
	for _, session := range sessions {
		stats := session.GetStats()
		// Only include active sessions with clients
		if session.IsActive() && session.GetClientCount() > 0 {
			status = append(status, stats)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_streams": len(status),
		"streams":       status,
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
	dataChan := session.AddClient(clientID, r.RemoteAddr)
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
	playlistDir := "./static/playlists"
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
	url := fmt.Sprintf("/playlists/%s", req.Filename)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":     url,
		"success": true,
	})
}
