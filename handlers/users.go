package handlers

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"iptv-panel/database"
	"iptv-panel/models"
	"net/http"
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
	json.NewEncoder(w).Encode(users)
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
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      id,
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
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
	json.NewEncoder(w).Encode(connections)
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


