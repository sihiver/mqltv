package main

import (
	"iptv-panel/database"
	"iptv-panel/handlers"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./iptv.db"
	}

	if err := database.InitDB(dbPath); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	// Setup router
	r := mux.NewRouter()

	// Auth routes (public)
	r.HandleFunc("/api/auth/login", handlers.Login).Methods("POST")
	r.HandleFunc("/api/auth/logout", handlers.Logout).Methods("POST")
	r.HandleFunc("/api/auth/check", handlers.CheckAuth).Methods("GET")
	r.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/login.html")
	}).Methods("GET")

	// Proxy channel stream (public with user auth - must be before api subrouter)
	r.HandleFunc("/api/proxy/channel/{id}", handlers.ProxyChannel).Methods("GET")
	r.HandleFunc("/api/proxy/channel/{id}/hls", handlers.ProxyChannelHLS).Methods("GET")

	// API routes (protected)
	api := r.PathPrefix("/api").Subrouter()
	api.Use(func(next http.Handler) http.Handler {
		return handlers.AuthMiddleware(next)
	})
	
	// Auth (protected)
	api.HandleFunc("/auth/change-password", handlers.ChangePassword).Methods("POST")
	
	// Stats
	api.HandleFunc("/stats", handlers.GetStats).Methods("GET")
	api.HandleFunc("/recently-watched", handlers.GetRecentlyWatchedChannels).Methods("GET")
	
	// Playlists
	api.HandleFunc("/playlists", handlers.GetPlaylists).Methods("GET")
	api.HandleFunc("/playlists/import", handlers.ImportPlaylist).Methods("POST")
	api.HandleFunc("/playlists/{id}", handlers.UpdatePlaylist).Methods("PUT")
	api.HandleFunc("/playlists/{id}", handlers.DeletePlaylist).Methods("DELETE")
	api.HandleFunc("/playlists/{id}/refresh", handlers.RefreshPlaylist).Methods("POST")
	api.HandleFunc("/playlists/{id}/channels", handlers.GetChannels).Methods("GET")
	api.HandleFunc("/playlists/{id}/export", handlers.ExportM3U).Methods("GET")
	
	// Channels
	api.HandleFunc("/channels", handlers.SearchChannels).Methods("GET")
	api.HandleFunc("/channels/search", handlers.SearchChannels).Methods("GET")
	api.HandleFunc("/channels/{id}/toggle", handlers.UpdateChannelStatus).Methods("POST")
	api.HandleFunc("/channels/{id}", handlers.DeleteChannel).Methods("DELETE")
	api.HandleFunc("/channels/batch-delete", handlers.BatchDeleteChannels).Methods("POST")
	
	// Relays
	api.HandleFunc("/relays", handlers.GetRelays).Methods("GET")
	api.HandleFunc("/relays", handlers.CreateRelay).Methods("POST")
	api.HandleFunc("/relays/{id}", handlers.DeleteRelay).Methods("DELETE")
	
	// Stream status
	api.HandleFunc("/streams/status", handlers.GetStreamStatus).Methods("GET")
	api.HandleFunc("/streams/{id}/status", handlers.GetStreamStatusByID).Methods("GET")
	
	// Users
	api.HandleFunc("/users", handlers.GetUsers).Methods("GET")
	api.HandleFunc("/users", handlers.CreateUser).Methods("POST")
	api.HandleFunc("/users/{id}", handlers.GetUserDetail).Methods("GET")
	api.HandleFunc("/users/{id}", handlers.UpdateUser).Methods("PUT")
	api.HandleFunc("/users/{id}", handlers.DeleteUser).Methods("DELETE")
	api.HandleFunc("/users/{id}/reset-password", handlers.ResetUserPassword).Methods("POST")
	api.HandleFunc("/users/{id}/connections", handlers.GetUserConnections).Methods("GET")
	api.HandleFunc("/users/{id}/set-expired", handlers.SetUserExpired).Methods("POST")
	api.HandleFunc("/users/{id}/extend", handlers.ExtendSubscription).Methods("POST")
	
	// Generate playlist
	api.HandleFunc("/generate-playlist", handlers.GenerateUserPlaylist).Methods("POST")
	
	// Generated Playlists
	api.HandleFunc("/generated-playlists", handlers.SaveGeneratedPlaylist).Methods("POST")
	
	// Stream relay endpoints (on-demand, multi-client)
	r.HandleFunc("/stream/{path:.+}", handlers.StreamRelay).Methods("GET")
	r.HandleFunc("/stream/{path:.+}/hls", handlers.StreamRelayHLS).Methods("GET")
	r.HandleFunc("/stream/{path:.+}/hls/{segment}", handlers.StreamRelayHLSSegment).Methods("GET")

	// Serve user playlists with short URL: /mql/{user}.m3u
	r.HandleFunc("/mql/{user:[a-zA-Z0-9_-]+}.m3u", handlers.ServeUserPlaylist).Methods("GET")
	
	// Serve generated playlists (legacy support)
	r.PathPrefix("/generated_playlists/").Handler(http.StripPrefix("/generated_playlists/", http.FileServer(http.Dir("./generated_playlists"))))
	
	// Serve static files with auth middleware
	r.PathPrefix("/").Handler(handlers.StaticAuthMiddleware(http.FileServer(http.Dir("./static"))))

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🎬 IPTV Panel server started on http://localhost:%s", port)
	log.Printf("📺 Open your browser and navigate to http://localhost:%s", port)
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server failed:", err)
	}
}
