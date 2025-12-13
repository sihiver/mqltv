package handlers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"iptv-panel/database"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
)

var store = sessions.NewCookieStore([]byte("iptv-panel-secret-key-change-in-production"))

func init() {
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// Login handles admin login
func Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword := fmt.Sprintf("%x", md5.Sum([]byte(req.Password)))

	// Check admin credentials in database
	var adminID int
	var username string
	err := database.DB.QueryRow(
		"SELECT id, username FROM admins WHERE username = ? AND password = ?",
		req.Username, hashedPassword,
	).Scan(&adminID, &username)

	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Create session
	session, _ := store.Get(r, "admin-session")
	session.Values["admin_id"] = adminID
	session.Values["username"] = username
	session.Values["logged_in"] = true
	session.Save(r, w)

	// Update last login
	database.DB.Exec("UPDATE admins SET last_login = ? WHERE id = ?", time.Now(), adminID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"username": username,
			"role":     "admin",
			"roleId":   "1",
		},
		"message": "Login successful",
	})
}

// Logout handles admin logout
func Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "admin-session")
	session.Values["logged_in"] = false
	session.Options.MaxAge = -1
	session.Save(r, w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"message": "Logout successful",
	})
}

// CheckAuth checks if user is authenticated
func CheckAuth(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "admin-session")
	loggedIn, ok := session.Values["logged_in"].(bool)
	username, _ := session.Values["username"].(string)

	if ok && loggedIn {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"username":      username,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": false,
	})
}

// ChangePassword handles admin password change
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "admin-session")
	adminID, ok := session.Values["admin_id"].(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate new password length
	if len(req.NewPassword) < 6 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Password baru minimal 6 karakter",
		})
		return
	}

	// Hash old password and verify
	hashedOldPassword := fmt.Sprintf("%x", md5.Sum([]byte(req.OldPassword)))
	
	var currentPassword string
	err := database.DB.QueryRow(
		"SELECT password FROM admins WHERE id = ?",
		adminID,
	).Scan(&currentPassword)

	if err != nil {
		http.Error(w, "Admin not found", http.StatusNotFound)
		return
	}

	if currentPassword != hashedOldPassword {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Password lama tidak sesuai",
		})
		return
	}

	// Hash new password and update
	hashedNewPassword := fmt.Sprintf("%x", md5.Sum([]byte(req.NewPassword)))
	
	_, err = database.DB.Exec(
		"UPDATE admins SET password = ? WHERE id = ?",
		hashedNewPassword, adminID,
	)

	if err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"message": "Password berhasil diubah",
	})
}

// GetProfile returns current admin profile
func GetProfile(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "admin-session")
	adminID, ok := session.Values["admin_id"].(int)

	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Not authenticated",
		})
		return
	}

	var admin struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}

	err := database.DB.QueryRow(
		"SELECT id, username, COALESCE(full_name, ''), COALESCE(email, '') FROM admins WHERE id = ?",
		adminID,
	).Scan(&admin.ID, &admin.Username, &admin.FullName, &admin.Email)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Failed to load profile",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": admin,
	})
}

// UpdateProfile updates admin profile (full_name, email)
func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "admin-session")
	adminID, ok := session.Values["admin_id"].(int)

	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Not authenticated",
		})
		return
	}

	var req struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Invalid request",
		})
		return
	}

	_, err := database.DB.Exec(
		"UPDATE admins SET full_name = ?, email = ? WHERE id = ?",
		req.FullName, req.Email, adminID,
	)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    1,
			"message": "Failed to update profile",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"message": "Profile updated successfully",
	})
}

// AuthMiddleware protects routes
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "admin-session")
		loggedIn, ok := session.Values["logged_in"].(bool)

		if !ok || !loggedIn {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// StaticAuthMiddleware for Vue SPA - serves index.html for all routes
func StaticAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		// Allow all static assets (js, css, images, fonts)
		if len(path) > 1 && (
			// Asset directories
			len(path) > 7 && path[:7] == "/assets" ||
			// Static files
			path == "/favicon.ico" ||
			path == "/logo.png" ||
			path == "/clear-storage.html" ||
			// Legacy support for old panel
			path == "/expired.html") {
			next.ServeHTTP(w, r)
			return
		}
		
		// For root path and all other routes, serve index.html (Vue SPA handles routing)
		// Vue router will handle /login, /dashboard, etc.
		next.ServeHTTP(w, r)
	})
}
