package streaming

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"iptv-panel/database"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// resolveHLSMasterPlaylist detects HLS master playlists and resolves them to a
// single media sub-playlist. This avoids multi-program/multi-bitrate inputs that
// cause timestamp discontinuities and player force-close on mobile devices.
func resolveHLSMasterPlaylist(inputURL string) string {
	if !strings.Contains(strings.ToLower(inputURL), ".m3u8") {
		return inputURL
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(inputURL)
	if err != nil {
		log.Printf("⚠️  HLS resolve: failed to fetch playlist: %v", err)
		return inputURL
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return inputURL
	}

	content := string(body)

	// Not a master playlist – already a media playlist
	if !strings.Contains(content, "#EXT-X-STREAM-INF") {
		return inputURL
	}

	type variant struct {
		bandwidth int
		url       string
	}
	var variants []variant

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#EXT-X-STREAM-INF") {
			continue
		}
		bw := 0
		if idx := strings.Index(line, "BANDWIDTH="); idx >= 0 {
			rest := line[idx+10:]
			if end := strings.IndexAny(rest, ",\r\n"); end >= 0 {
				rest = rest[:end]
			}
			bw, _ = strconv.Atoi(strings.TrimSpace(rest))
		}
		for j := i + 1; j < len(lines); j++ {
			next := strings.TrimSpace(lines[j])
			if next == "" || strings.HasPrefix(next, "#") {
				continue
			}
			if !strings.HasPrefix(next, "http") {
				base := inputURL[:strings.LastIndex(inputURL, "/")+1]
				next = base + next
			}
			variants = append(variants, variant{bw, next})
			break
		}
	}

	if len(variants) == 0 {
		return inputURL
	}

	// Sort ascending by bandwidth
	sort.Slice(variants, func(i, j int) bool {
		return variants[i].bandwidth < variants[j].bandwidth
	})

	// Always pick the highest bitrate variant (best quality).
	// The resolver already ensures a single-program sub-playlist, which is what
	// stabilises the stream — not the bitrate selection.
	chosen := variants[len(variants)-1]
	log.Printf("🎯 HLS Master → sub-playlist %d bps: %s", chosen.bandwidth, chosen.url)
	return chosen.url
}


func getFFmpegSettingInt(key string, defaultValue int) int {
	if database.DB == nil {
		return defaultValue
	}

	var value string
	err := database.DB.QueryRow(
		"SELECT value FROM settings WHERE key = ? AND category = 'ffmpeg'",
		key,
	).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return defaultValue
		}
		return defaultValue
	}

	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return defaultValue
	}
	return n
}

func getStatsSettingUint64(key string, defaultValue uint64) uint64 {
	if database.DB == nil {
		return defaultValue
	}

	var value string
	err := database.DB.QueryRow(
		"SELECT value FROM settings WHERE key = ? AND category = 'stats'",
		key,
	).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return defaultValue
		}
		return defaultValue
	}

	n, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return defaultValue
	}
	return n
}

func setStatsSettingUint64(key string, value uint64) {
	if database.DB == nil {
		return
	}

	// Best-effort upsert.
	_, _ = database.DB.Exec(
		`INSERT INTO settings (key, value, category)
		 VALUES (?, ?, 'stats')
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, category = 'stats', updated_at = CURRENT_TIMESTAMP`,
		key, strconv.FormatUint(value, 10),
	)
}

func getFFmpegExecutable() string {
	if database.DB == nil {
		return "ffmpeg"
	}
	var ffmpegPath string
	err := database.DB.QueryRow(
		"SELECT value FROM settings WHERE key = 'ffmpeg_path' AND category = 'ffmpeg'",
	).Scan(&ffmpegPath)
	if err != nil {
		return "ffmpeg"
	}
	ffmpegPath = strings.TrimSpace(ffmpegPath)
	if ffmpegPath == "" {
		return "ffmpeg"
	}
	return ffmpegPath
}

func getFFmpegUserAgent() string {
	if database.DB == nil {
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}
	var value string
	err := database.DB.QueryRow(
		"SELECT value FROM settings WHERE key = 'ffmpeg_user_agent' AND category = 'ffmpeg'",
	).Scan(&value)
	if err != nil || strings.TrimSpace(value) == "" {
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}
	return strings.TrimSpace(value)
}

func getFFmpegWatchdogConfig() (noDataTimeout time.Duration, interval time.Duration) {
	noDataTimeoutSeconds := getFFmpegSettingInt("no_data_timeout_seconds", 6)
	watchdogIntervalSeconds := getFFmpegSettingInt("watchdog_interval_seconds", 1)

	// Allow disabling watchdog entirely with 0.
	if noDataTimeoutSeconds < 0 {
		noDataTimeoutSeconds = 6
	}
	if noDataTimeoutSeconds > 600 {
		noDataTimeoutSeconds = 600
	}

	if watchdogIntervalSeconds < 1 {
		watchdogIntervalSeconds = 1
	}
	if watchdogIntervalSeconds > 60 {
		watchdogIntervalSeconds = 60
	}

	return time.Duration(noDataTimeoutSeconds) * time.Second, time.Duration(watchdogIntervalSeconds) * time.Second
}

func getFFmpegIdleTimeout() time.Duration {
	idleSeconds := getFFmpegSettingInt("idle_timeout", 60)
	if idleSeconds < 5 {
		idleSeconds = 5
	}
	if idleSeconds > 86400 {
		idleSeconds = 86400
	}
	return time.Duration(idleSeconds) * time.Second
}

func getFFmpegStartupProbeConfig() (reconnectDelayMaxSeconds int, timeoutMicros int, analyzeMicros int, probeBytes int) {
	reconnectDelayMaxSeconds = getFFmpegSettingInt("reconnect_delay_max_seconds", 5)
	if reconnectDelayMaxSeconds < 1 {
		reconnectDelayMaxSeconds = 1
	}
	if reconnectDelayMaxSeconds > 30 {
		reconnectDelayMaxSeconds = 30
	}

	inputTimeoutSeconds := getFFmpegSettingInt("input_timeout_seconds", 10)
	if inputTimeoutSeconds < 1 {
		inputTimeoutSeconds = 1
	}
	if inputTimeoutSeconds > 60 {
		inputTimeoutSeconds = 60
	}
	timeoutMicros = inputTimeoutSeconds * 1000 * 1000

	analyzeMs := getFFmpegSettingInt("startup_analyzeduration_ms", 5000)
	if analyzeMs < 100 {
		analyzeMs = 100
	}
	if analyzeMs > 20000 {
		analyzeMs = 20000
	}
	analyzeMicros = analyzeMs * 1000

	probeKB := getFFmpegSettingInt("startup_probesize_kb", 5000)
	if probeKB < 256 {
		probeKB = 256
	}
	if probeKB > 20000 {
		probeKB = 20000
	}
	probeBytes = probeKB * 1024

	return reconnectDelayMaxSeconds, timeoutMicros, analyzeMicros, probeBytes
}

func getFFmpegAudioDelayMs() int {
	return getFFmpegSettingInt("audio_delay_ms", 500)
}

// FFmpegSession manages FFmpeg-based streaming
type FFmpegSession struct {
	ID            string
	SourceURLs    []string
	OutputFormat  string // "mpegts", "hls", "copy"
	owner         *FFmpegManager
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
	lastDataTime  time.Time
	dataTimeMux   sync.RWMutex
	pipeWriter    *StreamPipe
	bytesRead                  uint64 // Total bytes from source
	bytesWritten               uint64 // Total bytes to clients
	bytesMux                   sync.RWMutex
	currentSourceIdx           int       // Current active index in SourceURLs for failover rotation
	retryCount                 int       // Number of consecutive failures
	corruptRestart             bool      // True when killed by auto-heal due to rain-fade/DTS corruption
	consecutiveCorruptRestarts int       // Count of consecutive corruption restarts
	lastFailTime               time.Time // Last time FFmpeg failed
	isBlacklisted              bool      // If true, stop trying to restart

	lastError     string
	lastErrorTime time.Time
	errorMux      sync.RWMutex

	// Real-time bandwidth tracking with sliding window
	lastBytesRead       uint64
	lastBytesWritten    uint64
	lastStatsTime       time.Time
	currentDownloadMbps float64
	currentUploadMbps   float64
	smoothingFactor     float64

	// Sliding window for stable average
	bytesHistory      []uint64 // History of bytes read
	bytesWriteHistory []uint64 // History of bytes written
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

	// Persisted totals across sessions (so dashboard totals don't drop to 0
	// when all streams are currently stopped).
	totalsMux           sync.Mutex
	persistedBytesRead  uint64
	persistedBytesWrite uint64
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
		ffmpegManager.persistedBytesRead = getStatsSettingUint64("stats_total_bytes_read", 0)
		ffmpegManager.persistedBytesWrite = getStatsSettingUint64("stats_total_bytes_write", 0)
		go ffmpegManager.monitorSessions()
		go ffmpegManager.cleanupOldFiles()
	})
	return ffmpegManager
}

func (m *FFmpegManager) addPersistedTotals(bytesRead, bytesWrite uint64) {
	if bytesRead == 0 && bytesWrite == 0 {
		return
	}

	m.totalsMux.Lock()
	m.persistedBytesRead += bytesRead
	m.persistedBytesWrite += bytesWrite
	newRead := m.persistedBytesRead
	newWrite := m.persistedBytesWrite
	m.totalsMux.Unlock()

	setStatsSettingUint64("stats_total_bytes_read", newRead)
	setStatsSettingUint64("stats_total_bytes_write", newWrite)
}

func (m *FFmpegManager) GetPersistedTotals() (bytesRead uint64, bytesWrite uint64) {
	m.totalsMux.Lock()
	defer m.totalsMux.Unlock()
	return m.persistedBytesRead, m.persistedBytesWrite
}

func (m *FFmpegManager) ResetPersistedTotals() {
	m.totalsMux.Lock()
	m.persistedBytesRead = 0
	m.persistedBytesWrite = 0
	m.totalsMux.Unlock()

	setStatsSettingUint64("stats_total_bytes_read", 0)
	setStatsSettingUint64("stats_total_bytes_write", 0)
}

// GetOrCreateFFmpegSession gets or creates FFmpeg session
func (m *FFmpegManager) GetOrCreateFFmpegSession(streamID string, sourceURLs []string, format string) *FFmpegSession {
	m.sessionsMux.Lock()
	defer m.sessionsMux.Unlock()

	if session, exists := m.sessions[streamID]; exists {
		session.lastActivity = time.Now()
		session.UpdateSourceURLs(sourceURLs)
		return session
	}

	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()
	session := &FFmpegSession{
		ID:                streamID,
		SourceURLs:        sourceURLs,
		OutputFormat:      format,
		owner:             m,
		onDemand:          true,
		ctx:               ctx,
		cancel:            cancel,
		clients:           make(map[string]*StreamClient),
		lastActivity:      now,
		startTime:         now,
		lastStatsTime:     now,
		smoothingFactor:   0.6,
		bytesHistory:      make([]uint64, 0, 10),
		bytesWriteHistory: make([]uint64, 0, 10),
		timeHistory:       make([]time.Time, 0, 10),
		pipeWriter: &StreamPipe{
			readers: make(map[string]chan []byte),
			buffer:  NewRingBuffer(5 * 1024 * 1024), // 5MB buffer
		},
	}

	m.sessions[streamID] = session
	log.Printf("🎬 Created FFmpeg session: %s (format: %s)", streamID, format)

	return session
}

// DeleteSession stops and removes a session from the manager.
// Safe to call from any goroutine.
func (m *FFmpegManager) DeleteSession(streamID string) {
	m.sessionsMux.Lock()
	session, ok := m.sessions[streamID]
	if ok {
		delete(m.sessions, streamID)
	}
	m.sessionsMux.Unlock()

	if ok {
		// Capture counters before stopping.
		session.bytesMux.RLock()
		read := session.bytesRead
		written := session.bytesWritten
		session.bytesMux.RUnlock()
		m.addPersistedTotals(read, written)

		session.Stop()
	}
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

// UpdateSourceURLs updates the failover source URLs list for an existing session.
func (s *FFmpegSession) UpdateSourceURLs(urls []string) {
	if len(urls) == 0 {
		return
	}
	s.activeMux.Lock()
	s.SourceURLs = append([]string(nil), urls...)
	if s.currentSourceIdx >= len(s.SourceURLs) {
		s.currentSourceIdx = 0
	}
	s.activeMux.Unlock()
}

// SwitchToSource manually switches the active source index and restarts FFmpeg if active.
func (s *FFmpegSession) SwitchToSource(index int) bool {
	s.activeMux.Lock()
	if index < 0 || index >= len(s.SourceURLs) {
		s.activeMux.Unlock()
		return false
	}
	s.currentSourceIdx = index
	s.corruptRestart = true
	s.consecutiveCorruptRestarts = 0
	s.retryCount = 0
	cmd := s.cmd
	isActive := s.isActive
	s.activeMux.Unlock()

	if isActive && cmd != nil && cmd.Process != nil {
		log.Printf("🔀 Manual source switch for %s to source [%d/%d]", s.ID, index+1, len(s.SourceURLs))
		_ = cmd.Process.Kill()
	}
	return true
}

// AddClient adds a client to FFmpeg session
func (s *FFmpegSession) AddClient(clientID, remoteAddr string) (chan []byte, error) {
	// Check if blacklisted
	if s.isBlacklisted {
		return nil, fmt.Errorf("channel is offline or unavailable")
	}

	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	s.pipeWriter.readersMux.Lock()
	dataChan, exists := s.pipeWriter.readers[clientID]
	if !exists {
		dataChan = make(chan []byte, 2000) // Large buffer to prevent packet drops
		s.pipeWriter.readers[clientID] = dataChan
	}
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

// IsBlacklisted checks if the session failed all retries due to upstream outage
func (s *FFmpegSession) IsBlacklisted() bool {
	s.activeMux.RLock()
	defer s.activeMux.RUnlock()
	return s.isBlacklisted
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
	s.dataTimeMux.Lock()
	s.lastDataTime = s.startTime
	s.dataTimeMux.Unlock()
	startIdx := s.currentSourceIdx
	s.activeMux.Unlock()

	log.Printf("▶️  Starting FFmpeg stream: %s", s.ID)

	// Try each source starting from currentSourceIdx (supports multi-source failover rotation)
	numSources := len(s.SourceURLs)
	for i := 0; i < numSources; i++ {
		idx := (startIdx + i) % numSources
		url := s.SourceURLs[idx]
		s.activeMux.Lock()
		s.currentSourceIdx = idx
		s.activeMux.Unlock()

		if s.startFFmpeg(url) {
			return
		}
		log.Printf("⚠️  FFmpeg failed to spawn for source [%d/%d]: %s, trying next...", idx+1, numSources, url)
	}

	log.Printf("❌ All sources failed for FFmpeg stream: %s", s.ID)
	s.activeMux.Lock()
	s.isActive = false
	s.activeMux.Unlock()
}

// startFFmpeg starts FFmpeg process
func (s *FFmpegSession) startFFmpeg(sourceURL string) bool {
	_, watchdogInterval := getFFmpegWatchdogConfig()
	reconnectDelayMaxSeconds, timeoutMicros, analyzeMicros, probeBytes := getFFmpegStartupProbeConfig()

	// Auto-resolve HLS master playlists to a single sub-playlist.
	// This prevents multi-program timestamp discontinuities that crash mobile players.
	sourceURL = resolveHLSMasterPlaylist(sourceURL)

	ua := getFFmpegUserAgent()
	audioDelayMs := getFFmpegAudioDelayMs()

	var audioDelayArgs []string
	if audioDelayMs != 0 {
		// Bitstream filter to shift audio PTS/DTS without transcoding (saves CPU).
		// Positive value delays audio (fixes audio playing earlier than video).
		delaySec := float64(audioDelayMs) / 1000.0
		audioDelayArgs = []string{"-bsf:a", fmt.Sprintf("setts=ts=TS+(%g)/TB", delaySec)}
		log.Printf("⏳ Audio delay configured: %d ms (%+.3fs) for stream %s", audioDelayMs, delaySec, s.ID)
	}

	// Build FFmpeg command optimized for multiple concurrent streams
	args := []string{
		"-threads", "1", // Limit to 1 thread per stream
		"-reconnect", "1", // Enable auto reconnect
		"-reconnect_streamed", "1", // Reconnect for streamed protocols
		"-reconnect_delay_max", strconv.Itoa(reconnectDelayMaxSeconds), // Max seconds between reconnects
		"-timeout", strconv.Itoa(timeoutMicros), // Input timeout (microseconds)
		"-user_agent", ua,
		"-fflags", "+genpts", // Generate missing PTS cleanly without dropping keyframes
		// NOTE: NO -fflags +discardcorrupt (drops keyframes at segment boundaries → gap → ExoPlayer crash)
		// NOTE: NO -flags low_delay (causes ExoPlayer timestamp issues)
		// NOTE: NO -re on live inputs (causes clock drift, HLS 404 segment skips, and TCP buffer overruns)
		"-analyzeduration", strconv.Itoa(analyzeMicros), // Analysis duration (microseconds)
		"-probesize", strconv.Itoa(probeBytes), // Probe size (bytes)
		"-i", sourceURL, // Input URL
		"-map", "0:v:0?", // Map ONLY first video stream (prevents multi-stream crash)
		"-map", "0:a:0?", // Map ONLY first audio stream
		"-sn",            // Drop subtitles
		"-dn",            // Drop data streams (timed_id3, etc.)
		"-c", "copy",     // Copy codec (no transcoding)
	}
	if len(audioDelayArgs) > 0 {
		args = append(args, audioDelayArgs...)
	}
	args = append(args,
		"-f", "mpegts", // Output format MPEG-TS
		// NOTE: NO -reset_timestamps 1 (causes container/bitstream PTS mismatch → ExoPlayer crash)
		// NOTE: NO -avoid_negative_ts (causes timestamp manipulation → ExoPlayer crash)
		// NOTE: NO -max_interleave_delta 0 (experimental, can cause issues)
		"-max_muxing_queue_size", "4096",
		"-mpegts_flags", "+resend_headers", // Periodically resend PAT/PMT so clients recover after glitches
		"-flush_packets", "1",
		"pipe:1", // Output to stdout
	)

	// If HLS output is requested
	if s.OutputFormat == "hls" {
		outputDir := filepath.Join("hls_cache", s.ID)
		os.MkdirAll(outputDir, 0755)

		args = []string{
			"-threads", "1",
			"-reconnect", "1",
			"-reconnect_streamed", "1",
			"-reconnect_delay_max", strconv.Itoa(reconnectDelayMaxSeconds),
			"-timeout", strconv.Itoa(timeoutMicros),
			"-user_agent", ua,
			"-fflags", "+genpts+discardcorrupt",
			"-flags", "low_delay",
			"-analyzeduration", strconv.Itoa(analyzeMicros),
			"-probesize", strconv.Itoa(probeBytes),
			"-i", sourceURL,
			"-map", "0:v:0?", // Map ONLY first video (prevents multi-stream crash)
			"-map", "0:a:0?", // Map ONLY first audio
			"-sn",            // Drop subtitles (can cause copy crash)
			"-c", "copy",
		}
		if len(audioDelayArgs) > 0 {
			args = append(args, audioDelayArgs...)
		}
		args = append(args,
			"-f", "hls",
			"-hls_time", "4",
			"-hls_list_size", "6",
			"-hls_flags", "delete_segments+append_list",
			"-progress", "pipe:1", // Send progress to stdout to keep watchdog alive
			filepath.Join(outputDir, "index.m3u8"),
		)
	}

	ffmpegExe := getFFmpegExecutable()
	s.cmd = exec.CommandContext(s.ctx, ffmpegExe, args...)

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

	cmd := s.cmd
	s.bytesMux.RLock()
	startBytes := s.bytesRead
	s.bytesMux.RUnlock()

	// Watchdog: if FFmpeg produces no output for a while (upstream stalled/down),
	// kill the stuck FFmpeg process so the wait loop can auto-restart or rotate to a backup source
	// or show the outage notification screen quickly.
	go func() {
		// Poll every 1 second for fast outage reaction
		checkInterval := 1 * time.Second
		if watchdogInterval < checkInterval {
			checkInterval = watchdogInterval
		}
		timer := time.NewTimer(checkInterval)
		defer timer.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-timer.C:
				s.activeMux.RLock()
				sameCmd := s.isActive && s.cmd == cmd
				s.activeMux.RUnlock()
				if !sameCmd {
					return
				}

				currentNoDataTimeout, _ := getFFmpegWatchdogConfig()
				timer.Reset(1 * time.Second)

				// If disabled, just keep looping.
				if currentNoDataTimeout <= 0 {
					continue
				}

				// Ensure startup timeout is always at least (analyzeDuration + 4s) so watchdog never kills FFmpeg
				// while it is still inside avformat_find_stream_info() probing a healthy stream.
				s.bytesMux.RLock()
				gotAnyBytes := s.bytesRead > startBytes
				s.bytesMux.RUnlock()

				effectiveTimeout := currentNoDataTimeout
				minStartupTimeout := time.Duration(analyzeMicros)*time.Microsecond + 4*time.Second
				if !gotAnyBytes && effectiveTimeout < minStartupTimeout {
					effectiveTimeout = minStartupTimeout
				}

				s.dataTimeMux.RLock()
				last := s.lastDataTime
				s.dataTimeMux.RUnlock()
				if !last.IsZero() && time.Since(last) > effectiveTimeout {
					log.Printf("⚠️  No stream data for %v on %s (upstream stalled). Killing FFmpeg to trigger failover/outage screen...", effectiveTimeout, s.ID)
					if cmd != nil && cmd.Process != nil {
						_ = cmd.Process.Kill()
					}
					return
				}
			}
		}
	}()

	// Read FFmpeg stderr in background (for logging errors and auto-restarting on severe DTS/stream corruption)
	go func() {
		buf := make([]byte, 4096)
		var corruptCount int
		corruptWindowStart := time.Now()

		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				errMsg := string(buf[:n])
				// Ignore harmless decoding spam during stream copy (also indicates upstream is actively sending packets during startup probe)
				if strings.Contains(errMsg, "decode_slice_header error") || strings.Contains(errMsg, "no frame!") || strings.Contains(errMsg, "non-existing PPS") {
					if time.Since(s.startTime) < 10*time.Second {
						s.dataTimeMux.Lock()
						s.lastDataTime = time.Now()
						s.dataTimeMux.Unlock()
					}
					continue
				}

				// Detect severe muxer/timestamp desync or upstream segment/rain-fade/wind corruption ("kaset rusak" state)
				dtsErrors := strings.Count(errMsg, "Non-monotonous DTS") +
					strings.Count(errMsg, "non monotonically increasing dts") +
					strings.Count(errMsg, "PES packet size mismatch") +
					strings.Count(errMsg, "out of order") +
					strings.Count(errMsg, "backward in time") +
					strings.Count(errMsg, "Failed to open segment") +
					strings.Count(errMsg, "Server returned 404")

				// Cooldown 15s after start so heavy rain doesn't cause rapid 5s restart loops
				if dtsErrors > 0 && time.Since(s.startTime) > 15*time.Second {
					now := time.Now()
					if now.Sub(corruptWindowStart) > 10*time.Second {
						corruptWindowStart = now
						corruptCount = 0
					}
					corruptCount += dtsErrors
					if corruptCount >= 20 {
						log.Printf("⚠️  Severe stream desync/rain-fade corruption detected for %s (%d errors in %v). Triggering clean auto-heal restart...",
							s.ID, corruptCount, now.Sub(corruptWindowStart).Round(time.Second))
						corruptCount = 0
						s.activeMux.Lock()
						s.corruptRestart = true
						s.activeMux.Unlock()
						if cmd != nil && cmd.Process != nil {
							_ = cmd.Process.Kill()
						}
						return
					}
				}

				// Log FFmpeg errors/warnings
				if strings.Contains(errMsg, "error") || strings.Contains(errMsg, "Error") ||
					strings.Contains(errMsg, "Invalid") || strings.Contains(errMsg, "failed") ||
					dtsErrors > 0 {
					log.Printf("🔴 FFmpeg error for %s: %s", s.ID, strings.TrimSpace(errMsg))
					s.errorMux.Lock()
					s.lastError = strings.TrimSpace(errMsg)
					s.lastErrorTime = time.Now()
					s.errorMux.Unlock()
				}
			}
			if err != nil {
				break
			}
		}
	}()

	// Read FFmpeg stdout and broadcast 188-byte aligned MPEG-TS packets to all clients
	go func() {
		defer func() {
			s.activeMux.Lock()
			s.isActive = false
			s.activeMux.Unlock()
			log.Printf("⏹️  FFmpeg reader stopped: %s", s.ID)
		}()

		const tsPacketSize = 188
		const tsReadSize = tsPacketSize * 44 // 8272 bytes (exact multiple of 188-byte MPEG-TS packet)

		buffer := make([]byte, tsReadSize)
		carry := make([]byte, 0, tsReadSize*2)

		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				n, err := stdout.Read(buffer)
				if err != nil {
					if err.Error() != "EOF" {
						log.Printf("⚠️  FFmpeg read error for %s: %v", s.ID, err)
					}
					return
				}

				if n > 0 {
					s.dataTimeMux.Lock()
					s.lastDataTime = time.Now()
					s.dataTimeMux.Unlock()

					// Track bytes read from source
					s.bytesMux.Lock()
					s.bytesRead += uint64(n)
					s.bytesMux.Unlock()

					var data []byte
					if s.OutputFormat == "hls" {
						data = make([]byte, n)
						copy(data, buffer[:n])
					} else {
						// Enforce strict 188-byte MPEG-TS packet alignment & 0x47 sync byte framing
						carry = append(carry, buffer[:n]...)

						if len(carry) < tsPacketSize {
							continue
						}

						// If not aligned to 0x47 sync byte, scan forward for two consecutive 0x47 sync bytes 188 bytes apart
						if carry[0] != 0x47 {
							if len(carry) < tsPacketSize*2 {
								continue
							}
							syncIdx := -1
							limit := len(carry) - tsPacketSize
							for i := 0; i < limit; i++ {
								if carry[i] == 0x47 && carry[i+tsPacketSize] == 0x47 {
									syncIdx = i
									break
								}
							}
							if syncIdx > 0 {
								carry = carry[syncIdx:]
							} else if syncIdx < 0 {
								// Keep last 188 bytes in case a sync byte is near the boundary
								carry = carry[len(carry)-tsPacketSize:]
								continue
							}
						}

						alignedLen := (len(carry) / tsPacketSize) * tsPacketSize
						if alignedLen == 0 {
							continue
						}

						// Filter out MPEG-TS packets with Transport Error Indicator (TEI bit: packet[1] & 0x80 != 0)
						// which commonly appear during Ku-Band satellite rain fade.
						data = make([]byte, 0, alignedLen)
						for offset := 0; offset < alignedLen; offset += tsPacketSize {
							pkt := carry[offset : offset+tsPacketSize]
							if pkt[0] == 0x47 && (pkt[1]&0x80) == 0 {
								data = append(data, pkt...)
							}
						}

						rem := len(carry) - alignedLen
						if rem > 0 {
							copy(carry[:rem], carry[alignedLen:])
						}
						carry = carry[:rem]

						if len(data) == 0 {
							continue
						}
					}

					// Broadcast to all clients (blocking with large buffer prevents drops)
					s.pipeWriter.readersMux.RLock()
					clientCount := len(s.pipeWriter.readers)
					for _, ch := range s.pipeWriter.readers {
						select {
						case ch <- data:
							// Sent successfully
						default:
							// Buffer full - skip this aligned chunk for this client
						}
					}
					s.pipeWriter.readersMux.RUnlock()

					// Track bytes written to clients
					s.bytesMux.Lock()
					s.bytesWritten += uint64(len(data) * clientCount)
					s.bytesMux.Unlock()
				}
			}
		}
	}()

	// Wait for FFmpeg to finish and handle restart / failover
	go func() {
		cmd.Wait()

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
				runDuration := time.Since(s.startTime)
				s.dataTimeMux.RLock()
				timeSinceLastData := time.Since(s.lastDataTime)
				s.dataTimeMux.RUnlock()

				s.activeMux.Lock()
				wasCorruptRestart := s.corruptRestart
				s.corruptRestart = false
				s.activeMux.Unlock()

				// Fast outage transition:
				// - If 0 bytes were received on this attempt and there is only 1 source URL, go straight to outage screen (maxRetries = 1)
				// - Otherwise try each source URL once (or 1 quick retry if stream was previously playing)
				s.bytesMux.RLock()
				gotAnyBytes := s.bytesRead > startBytes
				s.bytesMux.RUnlock()

				maxRetries := len(s.SourceURLs)
				if !gotAnyBytes && len(s.SourceURLs) <= 1 {
					maxRetries = 1
				} else if maxRetries < 2 {
					maxRetries = 2
				}

				if wasCorruptRestart {
					// Stream was alive and delivering data, but killed by auto-heal to reset corrupted DVB/Ku-Band timestamps.
					// Do NOT increment retryCount so prolonged rain fade won't blacklist/drop the channel.
					s.consecutiveCorruptRestarts++
					if s.consecutiveCorruptRestarts >= 2 && len(s.SourceURLs) > 1 {
						s.activeMux.Lock()
						s.currentSourceIdx = (s.currentSourceIdx + 1) % len(s.SourceURLs)
						nextIdx := s.currentSourceIdx
						s.activeMux.Unlock()
						s.consecutiveCorruptRestarts = 0
						log.Printf("🌧️  Repeated signal corruption on %s; failing over to backup source URL [%d/%d]",
							s.ID, nextIdx+1, len(s.SourceURLs))
					}
				} else if runDuration < 30*time.Second || timeSinceLastData >= 4*time.Second {
					// Consider the run failed/stalled if it died within 30s OR had no data flowing for >= 4s (watchdog kill)
					s.retryCount++
					s.lastFailTime = time.Now()
					// Rotate to next backup source URL if multiple sources are configured
					if len(s.SourceURLs) > 1 {
						s.activeMux.Lock()
						s.currentSourceIdx = (s.currentSourceIdx + 1) % len(s.SourceURLs)
						nextIdx := s.currentSourceIdx
						s.activeMux.Unlock()
						log.Printf("🔀 Rotating failover source for %s to URL [%d/%d] (retry: %d/%d)",
							s.ID, nextIdx+1, len(s.SourceURLs), s.retryCount, maxRetries)
					} else {
						log.Printf("⚠️  FFmpeg exited/stalled after %v for %s (retry: %d/%d)",
							runDuration.Round(time.Second), s.ID, s.retryCount, maxRetries)
					}
				} else {
					// Stream was healthy and actively receiving data for > 30s
					s.retryCount = 0
					s.consecutiveCorruptRestarts = 0
				}

				// Handle reaching max consecutive retries
				if s.retryCount >= maxRetries {
					if !s.IsOnDemand() {
						// Always-on stream (on_demand=false): do not delete permanently; wait 15s and keep trying
						log.Printf("⏳ Always-on stream %s reached %d retries; waiting 15s before retrying upstream...", s.ID, s.retryCount)
						s.retryCount = 0
						time.Sleep(15 * time.Second)
					} else {
						s.activeMux.Lock()
						s.isBlacklisted = true
						s.activeMux.Unlock()
						log.Printf("🚫 Channel %s offline after %d consecutive failures; switching viewers to outage screen.", s.ID, s.retryCount)

						// Stop & remove the session so connected clients immediately transition to ServeOutageNotification
						// and the session can be recreated fresh when a viewer retries later.
						if s.owner != nil {
							s.owner.DeleteSession(s.ID)
						} else {
							s.Stop()
						}
						return
					}
				} else {
					// Fast 1s backoff before retry/failover so viewers don't wait long
					time.Sleep(1 * time.Second)
				}

				// Mark as inactive and try to restart
				s.activeMux.Lock()
				s.isActive = false
				s.activeMux.Unlock()

				// Restart if should keep running and not blacklisted.
				s.clientsMux.RLock()
				stillHasClients := len(s.clients) > 0
				s.clientsMux.RUnlock()
				if (stillHasClients || !s.IsOnDemand()) && !s.isBlacklisted {
					log.Printf("🔄 Auto-restarting FFmpeg: %s (attempt %d/%d)", s.ID, s.retryCount+1, maxRetries)
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
		// Apply idle timeout setting live (helps on-demand feel faster by keeping sessions warm longer).
		m.idleTimeout = getFFmpegIdleTimeout()

		// Collect sessions to stop outside the lock.
		toDelete := make([]string, 0)
		m.sessionsMux.RLock()
		for streamID, session := range m.sessions {
			if session.GetClientCount() != 0 {
				continue
			}
			// If on-demand is disabled, keep the FFmpeg session running.
			if !session.IsOnDemand() {
				continue
			}
			idleTime := time.Since(session.lastActivity)
			if idleTime > m.idleTimeout {
				toDelete = append(toDelete, streamID)
			}
		}
		m.sessionsMux.RUnlock()

		for _, streamID := range toDelete {
			log.Printf("⏰ FFmpeg session idle for %v, stopping: %s", m.idleTimeout, streamID)
			m.DeleteSession(streamID)
		}
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

	s.errorMux.RLock()
	lastErr := s.lastError
	lastErrTime := s.lastErrorTime
	s.errorMux.RUnlock()

	s.activeMux.RLock()
	activeIdx := s.currentSourceIdx
	var activeURL string
	if activeIdx >= 0 && activeIdx < len(s.SourceURLs) {
		activeURL = s.SourceURLs[activeIdx]
	}
	sourceCount := len(s.SourceURLs)
	s.activeMux.RUnlock()

	return map[string]interface{}{
		"id":                  s.ID,
		"active":              s.IsActive(),
		"clients":             s.GetClientCount(),
		"output_format":       s.OutputFormat,
		"uptime_seconds":      uptime,
		"last_activity":       s.lastActivity,
		"bytes_read":          bytesRead,
		"bytes_written":       bytesWritten,
		"download_mbps":       downloadMbps,
		"upload_mbps":         uploadMbps,
		"last_error":          lastErr,
		"last_error_time":     lastErrTime,
		"active_source_index": activeIdx,
		"active_source_url":   activeURL,
		"source_count":        sourceCount,
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

// cleanupOldFiles removes old HLS files periodically
func (m *FFmpegManager) cleanupOldFiles() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	baseDir := "./hls_cache"

	for range ticker.C {
		// Remove directories not in active sessions
		m.sessionsMux.RLock()
		activeDirs := make(map[string]bool)
		for streamID := range m.sessions {
			activeDirs[streamID] = true
		}
		m.sessionsMux.RUnlock()

		// Clean up inactive directories
		dirs, _ := os.ReadDir(baseDir)
		for _, dir := range dirs {
			if dir.IsDir() && !activeDirs[dir.Name()] {
				dirPath := filepath.Join(baseDir, dir.Name())
				if info, err := os.Stat(dirPath); err == nil {
					if time.Since(info.ModTime()) > 10*time.Minute {
						os.RemoveAll(dirPath)
						log.Printf("🧹 Cleaned up old HLS directory: %s", dir.Name())
					}
				}
			}
		}
	}
}
