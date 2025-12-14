package handlers

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"iptv-panel/database"
	"iptv-panel/models"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// GetUsers returns all users
func GetUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`
		SELECT id, username, full_name, email, max_connections, is_active, 
		       created_at, activated_at, expires_at, last_login, notes
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Username, &user.FullName, &user.Email,
			&user.MaxConnections, &user.IsActive, &user.CreatedAt,
			&user.ActivatedAt, &user.ExpiresAt, &user.LastLogin, &user.Notes)
		if err != nil {
			continue
		}

		// Calculate days remaining and expired status
		if user.ExpiresAt != nil {
			remaining := time.Until(*user.ExpiresAt)
			user.DaysRemaining = int(remaining.Hours() / 24)
			user.IsExpired = remaining < 0
		}

		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": users,
	})
}

// CreateUser creates a new user
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username       string `json:"username"`
		Password       string `json:"password"`
		FullName       string `json:"full_name"`
		Email          string `json:"email"`
		MaxConnections int    `json:"max_connections"`
		DurationDays   int    `json:"duration_days"`
		Notes          string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	if req.MaxConnections == 0 {
		req.MaxConnections = 1
	}

	// Hash password
	passwordHash := fmt.Sprintf("%x", md5.Sum([]byte(req.Password)))

	// Calculate expiry date
	now := time.Now()
	var expiresAt *time.Time
	if req.DurationDays > 0 {
		expiry := now.AddDate(0, 0, req.DurationDays)
		expiresAt = &expiry
	}

	result, err := database.DB.Exec(`
		INSERT INTO users (username, password, full_name, email, max_connections, 
		                   is_active, activated_at, expires_at, notes)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?)
	`, req.Username, passwordHash, req.FullName, req.Email, req.MaxConnections, now, expiresAt, req.Notes)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to create user: " + err.Error(),
		})
		return
	}

	id, _ := result.LastInsertId()

	// Calculate days remaining
	var daysRemaining int
	if expiresAt != nil {
		remaining := time.Until(*expiresAt)
		daysRemaining = int(remaining.Hours() / 24)
	}

	// Return created user data
	user := map[string]interface{}{
		"id":              id,
		"username":        req.Username,
		"full_name":       req.FullName,
		"email":           req.Email,
		"max_connections": req.MaxConnections,
		"is_active":       true,
		"created_at":      now,
		"activated_at":    now,
		"expires_at":      expiresAt,
		"notes":           req.Notes,
		"days_remaining":  daysRemaining,
		"is_expired":      false,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"data":    user,
		"message": "User created successfully",
	})
}

// UpdateUser updates user information
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var req struct {
		FullName       string `json:"full_name"`
		Email          string `json:"email"`
		MaxConnections int    `json:"max_connections"`
		IsActive       bool   `json:"is_active"`
		ExtendDays     int    `json:"extend_days"`
		Notes          string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current user data
	var currentExpiresAt sql.NullTime
	err := database.DB.QueryRow("SELECT expires_at FROM users WHERE id = ?", userID).Scan(&currentExpiresAt)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Calculate new expiry if extending
	var newExpiresAt *time.Time
	if req.ExtendDays > 0 {
		baseTime := time.Now()
		if currentExpiresAt.Valid && currentExpiresAt.Time.After(time.Now()) {
			baseTime = currentExpiresAt.Time
		}
		expiry := baseTime.AddDate(0, 0, req.ExtendDays)
		newExpiresAt = &expiry
	}

	query := `UPDATE users SET full_name = ?, email = ?, max_connections = ?, is_active = ?, notes = ?`
	args := []interface{}{req.FullName, req.Email, req.MaxConnections, req.IsActive, req.Notes}

	if newExpiresAt != nil {
		query += `, expires_at = ?`
		args = append(args, newExpiresAt)
	}

	query += ` WHERE id = ?`
	args = append(args, userID)

	_, err = database.DB.Exec(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User updated successfully",
	})
}

// DeleteUser deletes a user
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	result, err := database.DB.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to delete user: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "User not found",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"success": true,
		},
		"message": "User deleted successfully",
	})
}

// ResetUserPassword resets user password
func ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var req struct {
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.NewPassword == "" {
		http.Error(w, "New password is required", http.StatusBadRequest)
		return
	}

	passwordHash := fmt.Sprintf("%x", md5.Sum([]byte(req.NewPassword)))

	_, err := database.DB.Exec("UPDATE users SET password = ? WHERE id = ?", passwordHash, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Password reset successfully",
	})
}

// GetUserConnections returns active connections for a user
func GetUserConnections(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	rows, err := database.DB.Query(`
		SELECT uc.id, uc.user_id, uc.channel_id, uc.ip_address, uc.connected_at, c.name as channel_name
		FROM user_connections uc
		LEFT JOIN channels c ON uc.channel_id = c.id
		WHERE uc.user_id = ? AND uc.disconnected_at IS NULL
		ORDER BY uc.connected_at DESC
	`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	connections := []map[string]interface{}{}
	for rows.Next() {
		var id, userID, channelID sql.NullInt64
		var ipAddress, channelName sql.NullString
		var connectedAt time.Time

		rows.Scan(&id, &userID, &channelID, &ipAddress, &connectedAt, &channelName)

		conn := map[string]interface{}{
			"id":           id.Int64,
			"channel_id":   channelID.Int64,
			"channel_name": channelName.String,
			"ip_address":   ipAddress.String,
			"connected_at": connectedAt,
			"duration":     time.Since(connectedAt).Minutes(),
		}
		connections = append(connections, conn)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": connections,
	})
}

// SetUserExpired sets user expiration date (for testing)
func SetUserExpired(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var req struct {
		Days int `json:"days"` // negative = expired, positive = extend
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	expiresAt := time.Now().AddDate(0, 0, req.Days)

	_, err := database.DB.Exec("UPDATE users SET expires_at = ? WHERE id = ?", expiresAt, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "User expiration updated",
		"expires_at": expiresAt,
	})
}

// ExtendSubscription extends user subscription by adding days to current expiration
func ExtendSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var req struct {
		Days int `json:"days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Days <= 0 {
		http.Error(w, "Days must be greater than 0", http.StatusBadRequest)
		return
	}

	// Get current user data
	var currentExpiresAt sql.NullTime
	err := database.DB.QueryRow("SELECT expires_at FROM users WHERE id = ?", userID).Scan(&currentExpiresAt)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate new expiration date
	var newExpiresAt time.Time
	if currentExpiresAt.Valid && currentExpiresAt.Time.After(time.Now()) {
		// If user has valid future expiration, extend from that date
		newExpiresAt = currentExpiresAt.Time.AddDate(0, 0, req.Days)
	} else {
		// If user is expired or has no expiration, extend from now
		newExpiresAt = time.Now().AddDate(0, 0, req.Days)
	}

	// Update database
	_, err = database.DB.Exec("UPDATE users SET expires_at = ?, is_active = 1 WHERE id = ?", newExpiresAt, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate days remaining
	daysRemaining := int(time.Until(newExpiresAt).Hours() / 24)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"message":        "Subscription extended successfully",
		"expires_at":     newExpiresAt,
		"days_extended":  req.Days,
		"days_remaining": daysRemaining,
	})
}

// GetUserDetail returns user details including generated playlist info and channels
func GetUserDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	// Get user info including password
	var user models.User
	var password string
	err := database.DB.QueryRow(`
		SELECT id, username, password, full_name, email, max_connections, is_active, 
		       created_at, activated_at, expires_at, last_login, notes
		FROM users WHERE id = ?
	`, userID).Scan(&user.ID, &user.Username, &password, &user.FullName, &user.Email,
		&user.MaxConnections, &user.IsActive, &user.CreatedAt,
		&user.ActivatedAt, &user.ExpiresAt, &user.LastLogin, &user.Notes)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Calculate days remaining
	if user.ExpiresAt != nil {
		remaining := time.Until(*user.ExpiresAt)
		user.DaysRemaining = int(remaining.Hours() / 24)
		user.IsExpired = remaining < 0
	}

	// Get playlist info and count channels in generated playlist
	playlistInfo := map[string]interface{}{
		"generated": false,
		"url":       "",
		"filename":  "",
	}

	var totalChannels int
	var userChannelIDs []int
	playlistPath := fmt.Sprintf("./generated_playlists/playlist-%s.m3u", user.Username)
	if fileInfo, err := os.Stat(playlistPath); err == nil {
		playlistInfo["generated"] = true
		playlistInfo["url"] = fmt.Sprintf("/mql/%s.m3u", user.Username)
		playlistInfo["filename"] = fmt.Sprintf("playlist-%s.m3u", user.Username)
		playlistInfo["size"] = fileInfo.Size()
		playlistInfo["generated_at"] = fileInfo.ModTime()

		// Parse M3U file to get channel IDs and count
		if content, err := os.ReadFile(playlistPath); err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "#EXTINF") {
					totalChannels++
					// Extract tvg-id from #EXTINF line
					// Format: #EXTINF:-1 tvg-id="123" tvg-name="..." ...
					if idx := strings.Index(line, `tvg-id="`); idx != -1 {
						idStr := line[idx+8:]
						if endIdx := strings.Index(idStr, `"`); endIdx != -1 {
							idStr = idStr[:endIdx]
							if id, err := strconv.Atoi(idStr); err == nil {
								userChannelIDs = append(userChannelIDs, id)
							}
						}
					}
				}
			}
		}
	}

	// Get channels grouped by playlist
	rows, err := database.DB.Query(`
		SELECT p.id, p.name, COUNT(c.id) as channel_count
		FROM playlists p
		LEFT JOIN channels c ON c.playlist_id = p.id AND c.active = 1
		GROUP BY p.id, p.name
		ORDER BY p.name
	`)
	if err == nil {
		defer rows.Close()
		playlists := []map[string]interface{}{}
		for rows.Next() {
			var playlistID int
			var playlistName string
			var channelCount int
			if err := rows.Scan(&playlistID, &playlistName, &channelCount); err == nil {
				playlists = append(playlists, map[string]interface{}{
					"id":            playlistID,
					"name":          playlistName,
					"channel_count": channelCount,
				})
			}
		}
		playlistInfo["available_playlists"] = playlists
	}

	// Get user's channels details (only channels in generated playlist)
	var userChannels []map[string]interface{}
	if len(userChannelIDs) > 0 {
		placeholders := make([]string, len(userChannelIDs))
		args := make([]interface{}, len(userChannelIDs))
		for i, id := range userChannelIDs {
			placeholders[i] = "?"
			args[i] = id
		}

		query := fmt.Sprintf(`
			SELECT c.id, c.name, c.logo, c.group_name, c.active, p.name as playlist_name
			FROM channels c
			LEFT JOIN playlists p ON c.playlist_id = p.id
			WHERE c.id IN (%s)
			ORDER BY c.group_name, c.name
		`, strings.Join(placeholders, ","))

		channelRows, err := database.DB.Query(query, args...)
		if err == nil {
			defer channelRows.Close()
			for channelRows.Next() {
				var id, active int
				var name, logo, group, playlistName sql.NullString
				if err := channelRows.Scan(&id, &name, &logo, &group, &active, &playlistName); err == nil {
					userChannels = append(userChannels, map[string]interface{}{
						"id":            id,
						"name":          name.String,
						"logo":          logo.String,
						"category":      group.String,
						"enabled":       active == 1,
						"playlist_name": playlistName.String,
					})
				}
			}
		}
	}

	// Create response with password included
	userResponse := map[string]interface{}{
		"id":              user.ID,
		"username":        user.Username,
		"password":        password,
		"full_name":       user.FullName,
		"email":           user.Email,
		"max_connections": user.MaxConnections,
		"is_active":       user.IsActive,
		"created_at":      user.CreatedAt,
		"activated_at":    user.ActivatedAt,
		"expires_at":      user.ExpiresAt,
		"last_login":      user.LastLogin,
		"notes":           user.Notes,
		"days_remaining":  user.DaysRemaining,
		"is_expired":      user.IsExpired,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"user":           userResponse,
			"playlist":       playlistInfo,
			"total_channels": totalChannels,
			"channels":       userChannels,
		},
	})
}

// GenerateUserPlaylist generates a custom M3U playlist for a user with selected channels
func GenerateUserPlaylist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID     int   `json:"user_id"`
		ChannelIDs []int `json:"channel_ids"`
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

	if req.UserID == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "User ID is required",
		})
		return
	}

	if len(req.ChannelIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "At least one channel is required",
		})
		return
	}

	// Get user details
	var user models.User
	err := database.DB.QueryRow("SELECT id, username, password FROM users WHERE id = ?", req.UserID).
		Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "User not found",
		})
		return
	}

	// Get channel details
	placeholders := make([]string, len(req.ChannelIDs))
	args := make([]interface{}, len(req.ChannelIDs))
	for i, id := range req.ChannelIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT c.id, c.name, c.logo, c.group_name, c.url 
		FROM channels c
		WHERE c.id IN (%s) AND c.active = 1
		ORDER BY c.group_name, c.name
	`, strings.Join(placeholders, ","))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to fetch channels",
		})
		return
	}
	defer rows.Close()

	// Collect channel data first (avoid database locked)
	type channelData struct {
		ID        int
		Name      string
		Logo      string
		Group     string
		SourceURL string
	}
	var channelsData []channelData

	for rows.Next() {
		var ch channelData
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Logo, &ch.Group, &ch.SourceURL); err != nil {
			continue
		}
		channelsData = append(channelsData, ch)
	}
	rows.Close() // Close rows before creating relays

	// Delete old relays for this user (cleanup before regenerating)
	// Get old relay paths for this user's channels
	oldRelayPaths := []string{}
	oldRelayRows, err := database.DB.Query(`
		SELECT DISTINCT output_path FROM relays 
		WHERE output_path LIKE 'channel-%' 
		AND output_path IN (
			SELECT 'channel-' || c.id 
			FROM channels c 
			WHERE c.id IN (` + strings.Join(placeholders, ",") + `)
		)
	`, args...)
	if err == nil {
		for oldRelayRows.Next() {
			var path string
			if err := oldRelayRows.Scan(&path); err == nil {
				oldRelayPaths = append(oldRelayPaths, path)
			}
		}
		oldRelayRows.Close()
		
		// Delete old relays
		for _, path := range oldRelayPaths {
			database.DB.Exec("DELETE FROM relays WHERE output_path = ?", path)
		}
	}

	// Build M3U content
	m3uContent := "#EXTM3U\n"
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost:8080"
	}

	channelCount := 0
	for _, ch := range channelsData {
		// Create or get relay for this channel
		relayPath := fmt.Sprintf("channel-%d", ch.ID)
		
		// Check if relay exists, if not create it
		var relayID int
		err := database.DB.QueryRow("SELECT id FROM relays WHERE output_path = ?", relayPath).Scan(&relayID)
		if err == sql.ErrNoRows {
			// Create new relay
			sourceURLsJSON := fmt.Sprintf("[\"%s\"]", ch.SourceURL)
			_, err = database.DB.Exec(
				"INSERT INTO relays (name, source_urls, output_path, active) VALUES (?, ?, ?, 1)",
				ch.Name, sourceURLsJSON, relayPath,
			)
			if err != nil {
				log.Printf("Failed to create relay for channel %d: %v", ch.ID, err)
				continue
			}
		}

		m3uContent += fmt.Sprintf("#EXTINF:-1 tvg-id=\"%d\" tvg-name=\"%s\" tvg-logo=\"%s\" group-title=\"%s\",%s\n",
			ch.ID, ch.Name, ch.Logo, ch.Group, ch.Name)
		m3uContent += fmt.Sprintf("http://%s/stream/%s?username=%s&password=%s\n",
			host, relayPath, user.Username, user.Password)
		channelCount++
	}

	// Create directory if not exists
	playlistDir := "./generated_playlists"
	if err := os.MkdirAll(playlistDir, 0755); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to create playlist directory",
		})
		return
	}

	// Save playlist file
	filename := fmt.Sprintf("playlist-%s.m3u", user.Username)
	filePath := fmt.Sprintf("%s/%s", playlistDir, filename)
	if err := os.WriteFile(filePath, []byte(m3uContent), 0644); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to save playlist file",
		})
		return
	}

	playlistURL := fmt.Sprintf("/mql/%s.m3u", user.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"url":           playlistURL,
			"filename":      filename,
			"channel_count": channelCount,
		},
		"message": fmt.Sprintf("Playlist generated successfully with %d channels", channelCount),
	})
}
