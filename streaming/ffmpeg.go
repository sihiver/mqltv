package streaming

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// FFmpegSession manages FFmpeg-based streaming
type FFmpegSession struct {
	ID            string
	SourceURLs    []string
	OutputFormat  string // "mpegts", "hls", "copy"
	onDemand      bool
	onDemandMux   sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	cmd           *exec.Cmd
	clients       map[string]*StreamClient
	clientsMux    sync.RWMutex
	isActive      bool
	activeMux     sync.RWMutex
	lastActivity  time.Time
	startTime     time.Time
	pipeWriter    *StreamPipe
	bytesRead     uint64 // Total bytes from source
	bytesWritten  uint64 // Total bytes to clients
	bytesMux      sync.RWMutex
	retryCount    int       // Number of consecutive failures
	lastFailTime  time.Time // Last time FFmpeg failed
	isBlacklisted bool      // If true, stop trying to restart
	
	// Real-time bandwidth tracking with sliding window
	lastBytesRead     uint64
	lastBytesWritten  uint64
	lastStatsTime     time.Time
	currentDownloadMbps float64
	currentUploadMbps   float64
	smoothingFactor     float64
	
	// Sliding window for stable average
	bytesHistory      []uint64  // History of bytes read
	bytesWriteHistory []uint64  // History of bytes written
	timeHistory       []time.Time
}

// StreamPipe handles FFmpeg output piping to multiple clients
type StreamPipe struct {
	readers    map[string]chan []byte
	readersMux sync.RWMutex
	buffer     *RingBuffer
}

// FFmpegManager manages all FFmpeg sessions
type FFmpegManager struct {
	sessions    map[string]*FFmpegSession
	sessionsMux sync.RWMutex
	idleTimeout time.Duration
}

var (
	ffmpegManager     *FFmpegManager
	ffmpegManagerOnce sync.Once
)

// GetFFmpegManager returns the global FFmpeg manager
func GetFFmpegManager() *FFmpegManager {
	ffmpegManagerOnce.Do(func() {
		ffmpegManager = &FFmpegManager{
			sessions:    make(map[string]*FFmpegSession),
			idleTimeout: 60 * time.Second,
		}
		go ffmpegManager.monitorSessions()
	})
	return ffmpegManager
}

// GetOrCreateFFmpegSession gets or creates FFmpeg session
func (m *FFmpegManager) GetOrCreateFFmpegSession(streamID string, sourceURLs []string, format string) *FFmpegSession {
	m.sessionsMux.Lock()
	defer m.sessionsMux.Unlock()

	if session, exists := m.sessions[streamID]; exists {
		session.lastActivity = time.Now()
		return session
	}

	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()
	session := &FFmpegSession{
		ID:           streamID,
		SourceURLs:   sourceURLs,
		OutputFormat: format,
		onDemand:     true,
		ctx:          ctx,
		cancel:       cancel,
		clients:      make(map[string]*StreamClient),
		lastActivity: now,
		startTime:    now,
		lastStatsTime: now,
		smoothingFactor: 0.6,
		bytesHistory:     make([]uint64, 0, 10),
		bytesWriteHistory: make([]uint64, 0, 10),
		timeHistory:      make([]time.Time, 0, 10),
		pipeWriter: &StreamPipe{
			readers: make(map[string]chan []byte),
			buffer:  NewRingBuffer(5 * 1024 * 1024), // 5MB buffer
		},
	}

	m.sessions[streamID] = session
	log.Printf("🎬 Created FFmpeg session: %s (format: %s)", streamID, format)

	return session
}

// SetOnDemand configures whether this session should auto-stop when idle.
// onDemand=true  => auto-start/auto-stop (default)
// onDemand=false => once started, keep running even with zero clients
func (s *FFmpegSession) SetOnDemand(onDemand bool) {
	s.onDemandMux.Lock()
	s.onDemand = onDemand
	s.onDemandMux.Unlock()
}

// IsOnDemand returns current on-demand mode.
func (s *FFmpegSession) IsOnDemand() bool {
	s.onDemandMux.RLock()
	defer s.onDemandMux.RUnlock()
	return s.onDemand
}

// AddClient adds a client to FFmpeg session
func (s *FFmpegSession) AddClient(clientID, remoteAddr string) (chan []byte, error) {
	// Check if blacklisted
	if s.isBlacklisted {
		return nil, fmt.Errorf("channel is offline or unavailable")
	}
	
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	dataChan := make(chan []byte, 2000) // Large buffer to prevent packet drops
	
	s.pipeWriter.readersMux.Lock()
	s.pipeWriter.readers[clientID] = dataChan
	s.pipeWriter.readersMux.Unlock()

	client := &StreamClient{
		ID:         clientID,
		Connected:  time.Now(),
		RemoteAddr: remoteAddr,
		Done:       make(chan bool, 1),
	}
	s.clients[clientID] = client
	s.lastActivity = time.Now()

	log.Printf("👤 Client connected to FFmpeg stream %s: %s (total: %d)", s.ID, clientID, len(s.clients))

	// Start FFmpeg if not active
	if !s.IsActive() {
		go s.Start()
	}

	return dataChan, nil
}

// RemoveClient removes client from FFmpeg session
func (s *FFmpegSession) RemoveClient(clientID string) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if _, exists := s.clients[clientID]; exists {
		delete(s.clients, clientID)
		
		s.pipeWriter.readersMux.Lock()
		if ch, ok := s.pipeWriter.readers[clientID]; ok {
			close(ch)
			delete(s.pipeWriter.readers, clientID)
		}
		s.pipeWriter.readersMux.Unlock()
		
		log.Printf("👋 Client disconnected from FFmpeg stream %s: %s (remaining: %d)", s.ID, clientID, len(s.clients))
	}

	s.lastActivity = time.Now()
}

// GetClientCount returns number of connected clients
func (s *FFmpegSession) GetClientCount() int {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()
	return len(s.clients)
}

// IsActive checks if FFmpeg is running
func (s *FFmpegSession) IsActive() bool {
	s.activeMux.RLock()
	defer s.activeMux.RUnlock()
	return s.isActive
}

// Start starts FFmpeg streaming
func (s *FFmpegSession) Start() {
	s.activeMux.Lock()
	if s.isActive {
		s.activeMux.Unlock()
		return
	}
	s.isActive = true
	s.startTime = time.Now() // Set start time here
	s.activeMux.Unlock()

	log.Printf("▶️  Starting FFmpeg stream: %s", s.ID)

	// Try each source until one works
	for _, url := range s.SourceURLs {
		if s.startFFmpeg(url) {
			return
		}
		log.Printf("⚠️  FFmpeg failed for source: %s, trying next...", url)
	}

	log.Printf("❌ All sources failed for FFmpeg stream: %s", s.ID)
	s.activeMux.Lock()
	s.isActive = false
	s.activeMux.Unlock()
}

// startFFmpeg starts FFmpeg process
func (s *FFmpegSession) startFFmpeg(sourceURL string) bool {
	// Build FFmpeg command optimized for multiple concurrent streams
	args := []string{
		"-threads", "1",               // Limit to 1 thread per stream
		"-reconnect", "1",             // Enable auto reconnect
		"-reconnect_streamed", "1",    // Reconnect for streamed protocols
		"-reconnect_delay_max", "5",   // Max 5 seconds between reconnects
		"-timeout", "10000000",        // 10 second timeout (in microseconds)
		"-fflags", "+genpts+discardcorrupt", // Generate PTS + discard corrupt packets
		"-flags", "low_delay",         // Low delay flag
		"-analyzeduration", "5000000", // 5 seconds analysis (detect all streams)
		"-probesize", "5000000",       // 5MB probe (ensure video detected)
		"-i", sourceURL,               // Input URL
		"-map", "0:v?",                // Map video stream (optional, won't fail if missing)
		"-map", "0:a?",                // Map audio stream (optional, won't fail if missing)
		"-c", "copy",                  // Copy codec (no transcoding)
		"-f", "mpegts",                // Output format MPEG-TS
		"-avoid_negative_ts", "make_zero", // Avoid timestamp issues
		"-max_muxing_queue_size", "9999", // Large muxing queue for stability
		"-bsf:v", "h264_mp4toannexb,dump_extra", // H264 conversion + dump extra data (SPS/PPS)
		"-async", "1",                 // Audio sync method (resample)
		"-vsync", "cfr",               // Video sync constant frame rate
		"-start_at_zero",              // Start timestamps at zero
		"-copytb", "1",                // Copy input timebase
		"pipe:1",                      // Output to stdout
	}

	// If HLS output is requested
	if s.OutputFormat == "hls" {
		args = []string{
			"-threads", "1",
			"-reconnect", "1",
			"-reconnect_streamed", "1",
			"-reconnect_delay_max", "5",
			"-timeout", "10000000",
			"-fflags", "+genpts+discardcorrupt",
			"-flags", "low_delay",
			"-analyzeduration", "5000000", // 5 seconds analysis
			"-probesize", "5000000",       // 5MB probe
			"-i", sourceURL,
			"-map", "0:v?",                // Map video (optional)
			"-map", "0:a?",                // Map audio (optional)
			"-c", "copy",
			"-f", "mpegts",
			"-avoid_negative_ts", "make_zero",
			"-max_muxing_queue_size", "9999",
			"-bsf:v", "h264_mp4toannexb,dump_extra",
			"-async", "1",
			"-vsync", "cfr",
			"-start_at_zero",
			"-copytb", "1",
			"pipe:1",
		}
	}

	s.cmd = exec.CommandContext(s.ctx, "ffmpeg", args...)
	
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		log.Printf("❌ Failed to create stdout pipe: %v", err)
		return false
	}

	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		log.Printf("❌ Failed to create stderr pipe: %v", err)
		return false
	}

	if err := s.cmd.Start(); err != nil {
		log.Printf("❌ Failed to start FFmpeg: %v", err)
		return false
	}

	log.Printf("✅ FFmpeg started for source: %s", sourceURL)

	// Read FFmpeg stderr in background (for logging errors)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				errMsg := string(buf[:n])
				// Log FFmpeg errors/warnings
				if strings.Contains(errMsg, "error") || strings.Contains(errMsg, "Error") || 
				   strings.Contains(errMsg, "Invalid") || strings.Contains(errMsg, "failed") {
					log.Printf("🔴 FFmpeg error for %s: %s", s.ID, errMsg)
				}
			}
			if err != nil {
				break
			}
		}
	}()

	// Read FFmpeg stdout and broadcast to all clients (non-blocking)
	go func() {
		defer func() {
			s.activeMux.Lock()
			s.isActive = false
			s.activeMux.Unlock()
			log.Printf("⏹️  FFmpeg reader stopped: %s", s.ID)
		}()

		buffer := make([]byte, 8192) // 8KB buffer - smaller chunks for better distribution
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				// Non-blocking read with SetReadDeadline would be ideal, but pipes don't support it
				// Instead, we read in goroutine and use select
				n, err := stdout.Read(buffer)
				if err != nil {
					if err.Error() != "EOF" {
						log.Printf("⚠️  FFmpeg read error for %s: %v", s.ID, err)
					}
					return
				}
				
				if n > 0 {
					data := make([]byte, n)
					copy(data, buffer[:n])

					// Track bytes read from source
					s.bytesMux.Lock()
					s.bytesRead += uint64(n)
					s.bytesMux.Unlock()

					// Write to buffer
					s.pipeWriter.buffer.Write(data)

					// Broadcast to all clients (blocking with large buffer prevents drops)
					s.pipeWriter.readersMux.RLock()
					clientCount := len(s.pipeWriter.readers)
					for _, ch := range s.pipeWriter.readers {
						// Blocking send - buffer is large enough (2000)
						// If buffer fills up, client is too slow and will experience lag
						select {
						case ch <- data:
							// Sent successfully
						default:
							// Buffer full - skip this packet for this client
							// With 2000 buffer, this should be rare
						}
					}
					s.pipeWriter.readersMux.RUnlock()

					// Track bytes written to clients (n * client_count)
					s.bytesMux.Lock()
					s.bytesWritten += uint64(n * clientCount)
					s.bytesMux.Unlock()
				}
			}
		}
	}()

	// Wait for FFmpeg to finish and handle restart
	go func() {
		s.cmd.Wait()
		
		// Check if we still have clients
		s.clientsMux.RLock()
		hasClients := len(s.clients) > 0
		s.clientsMux.RUnlock()
		
		// If we should keep the stream running (clients exist OR on-demand disabled)
		// and context not cancelled, try to restart.
		select {
		case <-s.ctx.Done():
			// Context cancelled, normal shutdown
			log.Printf("⏹️  FFmpeg stopped (shutdown): %s", s.ID)
		default:
			keepRunning := hasClients || !s.IsOnDemand()
			if keepRunning {
				// Check if stream is running for less than 30 seconds (likely offline/bad stream)
				runDuration := time.Since(s.startTime)
				
				// Increment retry count if failed quickly (< 30 seconds)
				if runDuration < 30*time.Second {
					s.retryCount++
					s.lastFailTime = time.Now()
					log.Printf("⚠️  FFmpeg died after %v for %s (retry: %d/2)", runDuration, s.ID, s.retryCount)
				} else {
					// Reset retry count if stream ran successfully for > 30 seconds
					s.retryCount = 0
				}
				
				// Blacklist if failed 2 times in a row
				if s.retryCount >= 2 {
					s.isBlacklisted = true
					log.Printf("🚫 Channel %s blacklisted after %d consecutive failures. Source likely offline.", s.ID, s.retryCount)
					
					// Disconnect all clients
					s.clientsMux.Lock()
					for clientID := range s.clients {
						delete(s.clients, clientID)
					}
					s.clientsMux.Unlock()
					
					return
				}
				
				time.Sleep(2 * time.Second)
				
				// Mark as inactive and try to restart
				s.activeMux.Lock()
				s.isActive = false
				s.activeMux.Unlock()
				
				// Restart if should keep running and not blacklisted.
				s.clientsMux.RLock()
				stillHasClients := len(s.clients) > 0
				s.clientsMux.RUnlock()
				if (stillHasClients || !s.IsOnDemand()) && !s.isBlacklisted {
					log.Printf("🔄 Auto-restarting FFmpeg: %s (attempt %d/5)", s.ID, s.retryCount+1)
					go s.Start()
				}
			} else {
				log.Printf("⏹️  FFmpeg stopped (no clients): %s", s.ID)
			}
		}
		
		s.activeMux.Lock()
		s.isActive = false
		s.activeMux.Unlock()
	}()

	return true
}

// Stop stops FFmpeg session
func (s *FFmpegSession) Stop() {
	log.Printf("🛑 Stopping FFmpeg stream: %s", s.ID)
	
	s.cancel()
	
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}

	// Close all client channels
	s.pipeWriter.readersMux.Lock()
	for _, ch := range s.pipeWriter.readers {
		close(ch)
	}
	s.pipeWriter.readers = make(map[string]chan []byte)
	s.pipeWriter.readersMux.Unlock()

	s.clientsMux.Lock()
	s.clients = make(map[string]*StreamClient)
	s.clientsMux.Unlock()
}

// monitorSessions monitors and stops idle sessions
func (m *FFmpegManager) monitorSessions() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.sessionsMux.Lock()
		for streamID, session := range m.sessions {
			if session.GetClientCount() == 0 {
				// If on-demand is disabled, keep the FFmpeg session running.
				if !session.IsOnDemand() {
					continue
				}
				idleTime := time.Since(session.lastActivity)
				if idleTime > m.idleTimeout {
					log.Printf("⏰ FFmpeg session idle for %v, stopping: %s", idleTime, streamID)
					session.Stop()
					delete(m.sessions, streamID)
				}
			}
		}
		m.sessionsMux.Unlock()
	}
}

// GetStats returns session statistics
func (s *FFmpegSession) GetStats() map[string]interface{} {
	s.bytesMux.RLock()
	bytesRead := s.bytesRead
	bytesWritten := s.bytesWritten
	s.bytesMux.RUnlock()

	uptime := time.Since(s.startTime).Seconds()
	now := time.Now()
	
	// Add current sample to history
	s.bytesHistory = append(s.bytesHistory, bytesRead)
	s.bytesWriteHistory = append(s.bytesWriteHistory, bytesWritten)
	s.timeHistory = append(s.timeHistory, now)
	
	// Keep only last 10 samples (sliding window ~30 seconds at 3s interval)
	maxSamples := 10
	if len(s.bytesHistory) > maxSamples {
		s.bytesHistory = s.bytesHistory[1:]
		s.bytesWriteHistory = s.bytesWriteHistory[1:]
		s.timeHistory = s.timeHistory[1:]
	}
	
	// Calculate average rate over the sliding window
	// Note: Mbps here means megabits/sec (bytes/sec * 8).
	downloadMbps := float64(0)
	uploadMbps := float64(0)
	
	if len(s.bytesHistory) >= 2 {
		// Calculate rate from oldest to newest sample in window
		oldestIdx := 0
		newestIdx := len(s.bytesHistory) - 1
		
		bytesDiffRead := s.bytesHistory[newestIdx] - s.bytesHistory[oldestIdx]
		bytesDiffWritten := s.bytesWriteHistory[newestIdx] - s.bytesWriteHistory[oldestIdx]
		timeDiff := s.timeHistory[newestIdx].Sub(s.timeHistory[oldestIdx]).Seconds()
		
		if timeDiff > 0 {
			downloadMbps = (float64(bytesDiffRead) * 8 / timeDiff) / 1024 / 1024
			uploadMbps = (float64(bytesDiffWritten) * 8 / timeDiff) / 1024 / 1024
		}
	}

	return map[string]interface{}{
		"id":              s.ID,
		"active":          s.IsActive(),
		"clients":         s.GetClientCount(),
		"output_format":   s.OutputFormat,
		"uptime_seconds":  uptime,
		"last_activity":   s.lastActivity,
		"bytes_read":      bytesRead,
		"bytes_written":   bytesWritten,
		"download_mbps":   downloadMbps,
		"upload_mbps":     uploadMbps,
	}
}

// GetAllSessions returns all FFmpeg sessions
func (m *FFmpegManager) GetAllSessions() []*FFmpegSession {
	m.sessionsMux.RLock()
	defer m.sessionsMux.RUnlock()

	sessions := make([]*FFmpegSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetSession returns specific session
func (m *FFmpegManager) GetSession(streamID string) *FFmpegSession {
	m.sessionsMux.RLock()
	defer m.sessionsMux.RUnlock()
	return m.sessions[streamID]
}
