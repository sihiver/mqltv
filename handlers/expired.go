package handlers

import (
	"iptv-panel/streaming"
	"net/http"
	"os"
	"strings"
)

// ServeExpiredImage serves the expired notification as continuous looping video stream
func ServeExpiredImage(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html") {
		if _, err := os.Stat("./static/expired.html"); err == nil {
			http.ServeFile(w, r, "./static/expired.html")
			return
		}
	}
	// Stream expired video using FFmpeg with infinite loop
	streaming.StreamExpiredVideo(w, r)
}

// ServeOutageNotification serves the signal disruption ("gangguan") page to browsers
// or streams the outage notification screen to IPTV players.
func ServeOutageNotification(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html") {
		if _, err := os.Stat("./static/gangguan.html"); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			http.ServeFile(w, r, "./static/gangguan.html")
			return
		}
	}
	streaming.StreamOutageVideo(w, r)
}
