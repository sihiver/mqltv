package models

import (
	"time"
)

type Playlist struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Type         string    `json:"type"` // "m3u" or "relay"
	ChannelCount int       `json:"channel_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Channel struct {
	ID         int       `json:"id"`
	PlaylistID int       `json:"playlist_id"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Logo       string    `json:"logo"`
	Group      string    `json:"group"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
}

type Relay struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	SourceURLs  string    `json:"source_urls"` // JSON array of URLs
	OutputPath  string    `json:"output_path"` // path untuk akses relay
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type User struct {
	ID             int        `json:"id"`
	Username       string     `json:"username"`
	Password       string     `json:"-"` // Never expose in JSON
	FullName       string     `json:"full_name"`
	Email          string     `json:"email"`
	MaxConnections int        `json:"max_connections"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	ActivatedAt    *time.Time `json:"activated_at"`
	ExpiresAt      *time.Time `json:"expires_at"`
	LastLogin      *time.Time `json:"last_login"`
	Notes          string     `json:"notes"`
	DaysRemaining  int        `json:"days_remaining"`
	IsExpired      bool       `json:"is_expired"`
}

type UserSession struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Token     string    `json:"token"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
