package handlers

// user_auth_compat.go — Android-app compatible endpoints
//
// The MQLTV Android app (LoginActivity.java) calls:
//   POST /api/auth/login  { "email": "...", "password": "..." }
//   expects: { "token": "...", "user": { "name": "...", "expiresAt": "..." } }
//
//   GET /api/auth/me  (Authorization: Bearer <token>)
//   expects: { "user": { "expiresAt": "...", ... } }
//
// This file bridges those calls to the MQLTV user database.

import (
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"iptv-panel/database"
	"net/http"
	"strings"
	"time"
)

// generateToken creates a random 32-byte hex token.
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// lookupUserByToken finds user info from the user_sessions table.
// Returns (userID, username, isActive, expiresAt, err).
func lookupUserByToken(token string) (int, string, bool, sql.NullTime, error) {
	var (
		userID    int
		username  string
		isActive  bool
		expiresAt sql.NullTime
		sessExp   sql.NullTime
	)
	err := database.DB.QueryRow(`
		SELECT u.id, u.username, u.is_active, u.expires_at, s.expires_at
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ?
	`, token).Scan(&userID, &username, &isActive, &expiresAt, &sessExp)
	if err != nil {
		return 0, "", false, sql.NullTime{}, err
	}
	// Check if session itself is expired
	if sessExp.Valid && sessExp.Time.Before(time.Now()) {
		return 0, "", false, sql.NullTime{}, sql.ErrNoRows
	}
	return userID, username, isActive, expiresAt, nil
}

// UserAuthLogin is the Android-app compatible login endpoint.
// POST /api/auth/login  { "email"/"username": "...", "password": "..." }
// Returns: { "token": "...", "user": { "name":"...", "expiresAt":"..." } }
func UserAuthLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Email    string `json:"email"`    // Android sends "email" field
		Username string `json:"username"` // fallback
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Invalid request body"})
		return
	}

	// Normalize: Android sends "email" but it's actually username
	username := strings.TrimSpace(req.Email)
	if username == "" {
		username = strings.TrimSpace(req.Username)
	}
	password := strings.TrimSpace(req.Password)

	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Username and password are required"})
		return
	}

	passwordHash := fmt.Sprintf("%x", md5.Sum([]byte(password)))

	var (
		userID      int
		fullName    string
		isActive    bool
		expiresAt   sql.NullTime
	)
	err := database.DB.QueryRow(`
		SELECT id, COALESCE(full_name, username), is_active, expires_at
		FROM users
		WHERE username = ? AND password = ?
	`, username, passwordHash).Scan(&userID, &fullName, &isActive, &expiresAt)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Username atau password salah"})
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Server error"})
		return
	}
	if !isActive {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Akun tidak aktif"})
		return
	}
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Langganan sudah habis"})
		return
	}

	// Create session token (valid 30 days)
	token := generateToken()
	sessExpiry := time.Now().Add(30 * 24 * time.Hour)
	_, err = database.DB.Exec(`
		INSERT INTO user_sessions (user_id, token, ip_address, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, userID, token, r.RemoteAddr, time.Now(), sessExpiry)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Gagal membuat sesi"})
		return
	}

	// Update last login
	database.DB.Exec("UPDATE users SET last_login = ? WHERE id = ?", time.Now(), userID)

	// Build expiresAt in RFC3339 (expected by Android SubscriptionGuard)
	var expiresAtStr string
	if expiresAt.Valid {
		expiresAtStr = expiresAt.Time.UTC().Format(time.RFC3339)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":        userID,
			"username":  username,
			"name":      fullName,
			"expiresAt": expiresAtStr,
		},
	})
}

// UserAuthMe is the Android-app compatible profile endpoint.
// GET /api/auth/me  (Authorization: Bearer <token>)
// Returns current user status including expiresAt for SubscriptionGuard.
func UserAuthMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Unauthorized"})
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	token = strings.TrimSpace(token)

	userID, username, isActive, expiresAt, err := lookupUserByToken(token)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Token tidak valid atau sudah kadaluarsa"})
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Server error"})
		return
	}
	if !isActive {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Akun tidak aktif"})
		return
	}
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Langganan sudah habis"})
		return
	}

	var expiresAtStr string
	if expiresAt.Valid {
		expiresAtStr = expiresAt.Time.UTC().Format(time.RFC3339)
	}

	// Fetch additional info
	var fullName, email string
	var daysRemaining int
	database.DB.QueryRow(`SELECT COALESCE(full_name,''), COALESCE(email,'') FROM users WHERE id = ?`, userID).
		Scan(&fullName, &email)
	if expiresAt.Valid {
		daysRemaining = int(time.Until(expiresAt.Time).Hours() / 24)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"id":            userID,
			"username":      username,
			"name":          fullName,
			"displayName":   fullName,
			"email":         email,
			"expiresAt":     expiresAtStr,
			"daysRemaining": daysRemaining,
			"isActive":      isActive,
		},
	})
}
