package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(filepath string) error {
	var err error
	DB, err = sql.Open("sqlite3", filepath)
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	log.Println("Database connected successfully")
	return createTables()
}

func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS playlists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			playlist_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			logo TEXT,
			group_name TEXT,
			active INTEGER DEFAULT 1,
			on_demand INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS relays (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			source_urls TEXT NOT NULL,
			output_path TEXT NOT NULL UNIQUE,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			full_name TEXT,
			email TEXT,
			max_connections INTEGER DEFAULT 1,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			activated_at DATETIME,
			expires_at DATETIME,
			last_login DATETIME,
			notes TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS user_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT NOT NULL UNIQUE,
			ip_address TEXT,
			user_agent TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS user_connections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			channel_id INTEGER,
			ip_address TEXT,
			connected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			disconnected_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			full_name TEXT,
			email TEXT,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_login DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			value TEXT NOT NULL,
			category TEXT DEFAULT 'system',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			return err
		}
	}

	// Create default admin if not exists
	createDefaultAdmin()
	
	// Create default settings
	createDefaultSettings()

	// Run migrations
	runMigrations()

	log.Println("Database tables created successfully")
	return nil
}

func createDefaultAdmin() {
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	
	if count == 0 {
		// Default password: admin123 (MD5 hashed)
		defaultPassword := "0192023a7bbd73250516f069df18b500" // MD5 of "admin123"
		_, err := DB.Exec(
			"INSERT INTO admins (username, password, full_name, is_active) VALUES (?, ?, ?, ?)",
			"admin", defaultPassword, "Administrator", 1,
		)
		if err == nil {
			log.Println("✅ Default admin created - Username: admin, Password: admin123")
		}
	}
}

func createDefaultSettings() {
	defaultSettings := map[string]map[string]string{
		"system": {
			"server_name":                "IPTV Panel",
			"server_url":                 "http://localhost:8080",
			"max_connections_per_user":   "3",
			"session_timeout":            "3600",
			"enable_user_registration":   "false",
			"enable_relay_mode":          "true",
		},
		"ffmpeg": {
			"ffmpeg_path":            "/usr/bin/ffmpeg",
			"buffer_size":            "2048",
			"idle_timeout":           "60",
			"max_streams":            "100",
			"enable_hls":             "true",
			"hls_segment_duration":   "6",
		},
		"stream": {
			"auto_start":         "true",
			"auto_stop":          "true",
			"max_bitrate":        "8000",
			"enable_transcode":   "false",
			"default_format":     "mpegts",
		},
	}

	for category, settings := range defaultSettings {
		for key, value := range settings {
			var count int
			DB.QueryRow("SELECT COUNT(*) FROM settings WHERE key = ?", key).Scan(&count)
			if count == 0 {
				DB.Exec("INSERT INTO settings (key, value, category) VALUES (?, ?, ?)", key, value, category)
			}
		}
	}
}

func runMigrations() {
	// Migration: Add on_demand column to channels table if not exists
	var columnExists int
	err := DB.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('channels') WHERE name='on_demand'
	`).Scan(&columnExists)
	
	if err == nil && columnExists == 0 {
		_, err = DB.Exec("ALTER TABLE channels ADD COLUMN on_demand INTEGER DEFAULT 1")
		if err == nil {
			log.Println("✅ Migration: Added on_demand column to channels table")
		} else {
			log.Printf("⚠️  Migration failed: %v", err)
		}
	}
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
