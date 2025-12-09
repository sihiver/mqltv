package handlers

import (
	"iptv-panel/streaming"
	"net/http"
)

// ServeExpiredImage serves the expired notification as continuous looping video stream
func ServeExpiredImage(w http.ResponseWriter, r *http.Request) {
	// Stream expired video using FFmpeg with infinite loop
	streaming.StreamExpiredVideo(w, r)
}
