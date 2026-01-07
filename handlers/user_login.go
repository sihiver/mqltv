package handlers

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"iptv-panel/database"
	"net/http"
	"time"
)

// UserLogin validates user credentials for client apps (e.g., Android) and
// returns account status including expiry information.
//
// Public endpoint (no admin session).
func UserLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Invalid request body",
		})
		return
	}

	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Username and password are required",
		})
		return
	}

	passwordHash := fmt.Sprintf("%x", md5.Sum([]byte(req.Password)))

	// Load user with credential check
	var (
		userID         int
		username       string
		fullName       string
		email          string
		maxConnections int
		isActive       bool
		createdAt      time.Time
		activatedAt    sql.NullTime
		expiresAt      sql.NullTime
		lastLogin      sql.NullTime
		notes          string
	)

	err := database.DB.QueryRow(`
		SELECT id, username, COALESCE(full_name, ''), COALESCE(email, ''), max_connections,
		       is_active, created_at, activated_at, expires_at, last_login, COALESCE(notes, '')
		FROM users
		WHERE username = ? AND password = ?
	`, req.Username, passwordHash).Scan(
		&userID,
		&username,
		&fullName,
		&email,
		&maxConnections,
		&isActive,
		&createdAt,
		&activatedAt,
		&expiresAt,
		&lastLogin,
		&notes,
	)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 1,
			"data": map[string]interface{}{
				"username":          req.Username,
				"valid_credentials": false,
			},
			"message": "Invalid credentials",
		})
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    nil,
			"message": "Failed to login",
		})
		return
	}

	// Calculate expiry
	isExpired := false
	daysRemaining := 0
	var expiresAtPtr interface{} = nil
	if expiresAt.Valid {
		expiresAtPtr = expiresAt.Time
		remaining := time.Until(expiresAt.Time)
		daysRemaining = int(remaining.Hours() / 24)
		isExpired = remaining < 0
	}

	// Active connections count
	var activeConnections int
	database.DB.QueryRow(`
		SELECT COUNT(*) FROM user_connections
		WHERE user_id = ? AND disconnected_at IS NULL
	`, userID).Scan(&activeConnections)

	var activatedAtPtr interface{} = nil
	if activatedAt.Valid {
		activatedAtPtr = activatedAt.Time
	}
	var lastLoginPtr interface{} = nil
	if lastLogin.Valid {
		lastLoginPtr = lastLogin.Time
	}

	data := map[string]interface{}{
		"id":                 userID,
		"username":           username,
		"full_name":          fullName,
		"email":              email,
		"max_connections":    maxConnections,
		"active_connections": activeConnections,
		"is_active":          isActive,
		"is_expired":         isExpired,
		"days_remaining":     daysRemaining,
		"created_at":         createdAt,
		"activated_at":       activatedAtPtr,
		"expires_at":         expiresAtPtr,
		"last_login":         lastLoginPtr,
		"notes":              notes,
		"valid_credentials":  true,
		"playlist_url":       fmt.Sprintf("/mql/%s.m3u", username),
	}

	if !isActive {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    data,
			"message": "User account is inactive",
		})
		return
	}

	if isExpired {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"data":    data,
			"message": "User subscription has expired",
		})
		return
	}

	// Update last login timestamp (best-effort) only on successful login
	database.DB.Exec("UPDATE users SET last_login = ? WHERE id = ?", time.Now(), userID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"data":    data,
		"message": "Login successful",
	})
}
