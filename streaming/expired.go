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

var (
	outageStreamMux     sync.Mutex
	outageStreamClients = make(map[string]chan []byte)
	outageStreamActive  = false
)

// StreamOutageVideo streams the outage/disruption ("gangguan sinyal") screen in an infinite loop
func StreamOutageVideo(w http.ResponseWriter, r *http.Request) {
	videoPath := "./static/gangguan-notification.mp4"
	imagePath := "./static/gangguan-notification.png"
	htmlPath := "./static/gangguan.html"

	// Auto-generate MP4 from PNG once if MP4 does not exist yet
	if _, err := os.Stat(videoPath); err != nil {
		if _, imgErr := os.Stat(imagePath); imgErr == nil {
			ffmpegExe := getFFmpegExecutable()
			genCmd := exec.Command(
				ffmpegExe,
				"-y",
				"-loop", "1",
				"-framerate", "25",
				"-i", imagePath,
				"-f", "lavfi",
				"-i", "anullsrc=r=44100:cl=stereo",
				"-t", "4",
				"-c:v", "libx264",
				"-preset", "fast",
				"-tune", "stillimage",
				"-pix_fmt", "yuv420p",
				"-g", "12",
				"-keyint_min", "12",
				"-sc_threshold", "0",
				"-bf", "0",
				"-c:a", "aac",
				"-b:a", "64k",
				"-shortest",
				videoPath,
			)
			_ = genCmd.Run()
		}
	}

	// If MP4 still doesn't exist, fall back to HTML page
	if _, err := os.Stat(videoPath); err != nil {
		if _, htmlErr := os.Stat(htmlPath); htmlErr == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			http.ServeFile(w, r, htmlPath)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("SEDANG TERJADI GANGGUAN SINYAL\n\nMohon maaf, saluran sedang mengalami gangguan sementara.\nSilakan mencoba kembali beberapa saat lagi."))
		return
	}

	clientID := fmt.Sprintf("%x", md5.Sum([]byte(r.RemoteAddr+r.UserAgent()+fmt.Sprintf("%d", time.Now().UnixNano()))))
	dataChan := make(chan []byte, 2000)

	outageStreamMux.Lock()
	outageStreamClients[clientID] = dataChan
	if !outageStreamActive {
		outageStreamActive = true
		go startOutageStream(videoPath)
	}
	outageStreamMux.Unlock()

	defer func() {
		outageStreamMux.Lock()
		delete(outageStreamClients, clientID)
		close(dataChan)
		outageStreamMux.Unlock()
	}()

	// Only set headers if not already written
	w.Header().Set("Content-Type", "video/MP2T")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

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

func startOutageStream(videoPath string) {
	defer func() {
		outageStreamMux.Lock()
		outageStreamActive = false
		outageStreamMux.Unlock()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ffmpegExe := getFFmpegExecutable()
	args := []string{
		"-stream_loop", "-1",
		"-re",
		"-i", videoPath,
		"-c", "copy",
		"-f", "mpegts",
		"-avoid_negative_ts", "make_zero",
		"-mpegts_flags", "+resend_headers+initial_discontinuity",
		"-max_muxing_queue_size", "9999",
		"-flush_packets", "1",
		"pipe:1",
	}

	cmd := exec.CommandContext(ctx, ffmpegExe, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}

	buffer := make([]byte, 188*7)
	for {
		outageStreamMux.Lock()
		clientCount := len(outageStreamClients)
		outageStreamMux.Unlock()
		if clientCount == 0 {
			_ = cmd.Process.Kill()
			break
		}

		n, err := stdout.Read(buffer)
		if err != nil {
			break
		}
		if n > 0 {
			data := make([]byte, n)
			copy(data, buffer[:n])
			outageStreamMux.Lock()
			for _, ch := range outageStreamClients {
				select {
				case ch <- data:
				default:
				}
			}
			outageStreamMux.Unlock()
		}
	}

	_ = cmd.Wait()
}
