package streaming

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	expiredStreamMux     sync.Mutex
	expiredStreamClients = make(map[string]chan []byte)
	expiredStreamActive  = false
	expiredStreamCancel  context.CancelFunc
)

// StreamExpiredVideo streams the expired notification video in infinite loop
func StreamExpiredVideo(w http.ResponseWriter, r *http.Request) {
	videoPath := "./static/expired-notification.mp4"
	
	// Check if video exists
	if _, err := os.Stat(videoPath); err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("SUBSCRIPTION EXPIRED\n\nLangganan Anda Telah Berakhir\nYour Subscription Has Expired\n\nHubungi Admin / Contact Admin"))
		return
	}
	
	// Generate client ID
	clientID := fmt.Sprintf("%x", md5.Sum([]byte(r.RemoteAddr+r.UserAgent()+fmt.Sprintf("%d", time.Now().UnixNano()))))
	
	// Create data channel for this client
	dataChan := make(chan []byte, 2000)
	
	expiredStreamMux.Lock()
	expiredStreamClients[clientID] = dataChan
	
	// Start FFmpeg stream if not active
	if !expiredStreamActive {
		expiredStreamActive = true
		go startExpiredStream(videoPath)
	}
	expiredStreamMux.Unlock()
	
	// Cleanup on disconnect
	defer func() {
		expiredStreamMux.Lock()
		delete(expiredStreamClients, clientID)
		close(dataChan)
		log.Printf("👋 Expired stream client disconnected: %s (remaining: %d)", clientID, len(expiredStreamClients))
		expiredStreamMux.Unlock()
	}()
	
	// Set streaming headers
	w.Header().Set("Content-Type", "video/MP2T")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	
	log.Printf("👤 Expired stream client connected: %s", clientID)
	
	// Stream data to client
	for {
		select {
		case data, ok := <-dataChan:
			if !ok {
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func startExpiredStream(videoPath string) {
	defer func() {
		expiredStreamMux.Lock()
		expiredStreamActive = false
		expiredStreamMux.Unlock()
	}()
	
	ctx, cancel := context.WithCancel(context.Background())
	expiredStreamCancel = cancel
	defer cancel()
	
	log.Printf("🎬 Starting expired notification stream (infinite loop)")
	
	// FFmpeg command with infinite loop
	args := []string{
		"-stream_loop", "-1",          // Infinite loop
		"-re",                          // Read input at native framerate
		"-i", videoPath,                // Input video file
		"-c", "copy",                   // Copy without re-encoding
		"-f", "mpegts",                 // MPEG-TS format
		"-avoid_negative_ts", "make_zero",
		"-max_muxing_queue_size", "9999",
		"pipe:1",                       // Output to stdout
	}
	
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("❌ Failed to create pipe for expired stream: %v", err)
		return
	}
	
	if err := cmd.Start(); err != nil {
		log.Printf("❌ Failed to start FFmpeg for expired stream: %v", err)
		return
	}
	
	// Read from FFmpeg and broadcast to all clients
	buffer := make([]byte, 188*7) // MPEG-TS packet size (188 bytes) * 7
	for {
		n, err := stdout.Read(buffer)
		if err != nil {
			log.Printf("⚠️ Expired stream ended: %v", err)
			break
		}
		
		if n > 0 {
			data := make([]byte, n)
			copy(data, buffer[:n])
			
			// Broadcast to all connected clients
			expiredStreamMux.Lock()
			for clientID, ch := range expiredStreamClients {
				select {
				case ch <- data:
				default:
					log.Printf("⚠️ Client %s buffer full, skipping packet", clientID)
				}
			}
			expiredStreamMux.Unlock()
		}
	}
	
	cmd.Wait()
	log.Printf("🛑 Expired notification stream stopped")
}
