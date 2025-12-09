package streaming

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HLSSession manages HLS streaming for a source
type HLSSession struct {
	ID            string
	SourceURLs    []string
	OutputDir     string
	ctx           context.Context
	cancel        context.CancelFunc
	clients       map[string]time.Time
	clientsMux    sync.RWMutex
	isActive      bool
	activeMux     sync.RWMutex
	lastActivity  time.Time
	segmentIndex  int
	playlistFile  string
	segments      []string
	maxSegments   int
	segmentDur    time.Duration
}

// HLSManager manages all HLS sessions
type HLSManager struct {
	sessions    map[string]*HLSSession
	sessionsMux sync.RWMutex
	baseDir     string
	idleTimeout time.Duration
}

var (
	hlsManager     *HLSManager
	hlsManagerOnce sync.Once
)

// GetHLSManager returns the global HLS manager instance
func GetHLSManager() *HLSManager {
	hlsManagerOnce.Do(func() {
		baseDir := "./hls_cache"
		os.MkdirAll(baseDir, 0755)

		hlsManager = &HLSManager{
			sessions:    make(map[string]*HLSSession),
			baseDir:     baseDir,
			idleTimeout: 60 * time.Second,
		}
		go hlsManager.monitorSessions()
		go hlsManager.cleanupOldFiles()
	})
	return hlsManager
}

// GetOrCreateHLSSession gets existing HLS session or creates new one
func (m *HLSManager) GetOrCreateHLSSession(streamID string, sourceURLs []string) *HLSSession {
	m.sessionsMux.Lock()
	defer m.sessionsMux.Unlock()

	if session, exists := m.sessions[streamID]; exists {
		session.UpdateActivity()
		return session
	}

	// Create new HLS session
	ctx, cancel := context.WithCancel(context.Background())
	outputDir := filepath.Join(m.baseDir, streamID)
	os.MkdirAll(outputDir, 0755)

	session := &HLSSession{
		ID:           streamID,
		SourceURLs:   sourceURLs,
		OutputDir:    outputDir,
		ctx:          ctx,
		cancel:       cancel,
		clients:      make(map[string]time.Time),
		lastActivity: time.Now(),
		playlistFile: filepath.Join(outputDir, "playlist.m3u8"),
		maxSegments:  6,
		segmentDur:   4 * time.Second,
	}

	m.sessions[streamID] = session
	log.Printf("🎬 Created new HLS session: %s", streamID)

	// Start HLS generation
	go session.Start()

	return session
}

// UpdateActivity updates the last activity time
func (s *HLSSession) UpdateActivity() {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	s.lastActivity = time.Now()
}

// AddClient adds a client to the HLS session
func (s *HLSSession) AddClient(clientID string) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	s.clients[clientID] = time.Now()
	s.lastActivity = time.Now()
	log.Printf("👤 HLS client added to %s: %s (total: %d)", s.ID, clientID, len(s.clients))
}

// RemoveClient removes a client from the HLS session
func (s *HLSSession) RemoveClient(clientID string) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	delete(s.clients, clientID)
	log.Printf("👋 HLS client removed from %s: %s (remaining: %d)", s.ID, clientID, len(s.clients))
}

// GetClientCount returns the number of active clients
func (s *HLSSession) GetClientCount() int {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()
	return len(s.clients)
}

// Start starts the HLS stream processing
func (s *HLSSession) Start() {
	s.activeMux.Lock()
	if s.isActive {
		s.activeMux.Unlock()
		return
	}
	s.isActive = true
	s.activeMux.Unlock()

	log.Printf("▶️  Starting HLS stream: %s", s.ID)

	// Connect to source
	var sourceResp *http.Response
	var err error

	for _, url := range s.SourceURLs {
		sourceResp, err = http.Get(url)
		if err == nil && sourceResp.StatusCode == http.StatusOK {
			log.Printf("✅ HLS connected to source: %s", url)
			break
		}
		if sourceResp != nil {
			sourceResp.Body.Close()
		}
	}

	if sourceResp == nil || sourceResp.StatusCode != http.StatusOK {
		log.Printf("❌ All sources failed for HLS stream: %s", s.ID)
		s.activeMux.Lock()
		s.isActive = false
		s.activeMux.Unlock()
		return
	}

	defer sourceResp.Body.Close()

	// Create segments from stream
	go s.createSegments(sourceResp.Body)
}

// createSegments creates HLS segments from the stream
func (s *HLSSession) createSegments(reader io.Reader) {
	buffer := new(bytes.Buffer)
	segmentBuffer := make([]byte, 188*7) // MPEG-TS packet size
	lastSegmentTime := time.Now()

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("⏹️  HLS stream stopped: %s", s.ID)
			return
		default:
			n, err := reader.Read(segmentBuffer)
			if n > 0 {
				buffer.Write(segmentBuffer[:n])

				// Create segment every segmentDur seconds or when buffer is large enough
				if time.Since(lastSegmentTime) >= s.segmentDur || buffer.Len() > 1024*1024 {
					s.writeSegment(buffer.Bytes())
					buffer.Reset()
					lastSegmentTime = time.Now()
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("⚠️  HLS stream read error: %s - %v", s.ID, err)
				}
				return
			}
		}
	}
}

// writeSegment writes a segment to disk and updates playlist
func (s *HLSSession) writeSegment(data []byte) {
	if len(data) == 0 {
		return
	}

	segmentName := fmt.Sprintf("segment_%d.ts", s.segmentIndex)
	segmentPath := filepath.Join(s.OutputDir, segmentName)

	// Write segment file
	if err := os.WriteFile(segmentPath, data, 0644); err != nil {
		log.Printf("❌ Failed to write segment: %v", err)
		return
	}

	// Update segment list
	s.segments = append(s.segments, segmentName)
	if len(s.segments) > s.maxSegments {
		// Remove old segment
		oldSegment := s.segments[0]
		s.segments = s.segments[1:]
		os.Remove(filepath.Join(s.OutputDir, oldSegment))
	}

	s.segmentIndex++

	// Update playlist
	s.updatePlaylist()
}

// updatePlaylist updates the HLS playlist file
func (s *HLSSession) updatePlaylist() {
	playlist := "#EXTM3U\n"
	playlist += "#EXT-X-VERSION:3\n"
	playlist += fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", int(s.segmentDur.Seconds()))
	playlist += fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", s.segmentIndex-len(s.segments))

	for _, segment := range s.segments {
		playlist += fmt.Sprintf("#EXTINF:%.3f,\n", s.segmentDur.Seconds())
		playlist += segment + "\n"
	}

	if err := os.WriteFile(s.playlistFile, []byte(playlist), 0644); err != nil {
		log.Printf("❌ Failed to write playlist: %v", err)
	}
}

// Stop stops the HLS session
func (s *HLSSession) Stop() {
	log.Printf("🛑 Stopping HLS stream: %s", s.ID)
	s.cancel()
	s.activeMux.Lock()
	s.isActive = false
	s.activeMux.Unlock()
}

// monitorSessions monitors all HLS sessions and stops idle ones
func (m *HLSManager) monitorSessions() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.sessionsMux.Lock()
		for streamID, session := range m.sessions {
			// Clean up old clients
			session.clientsMux.Lock()
			for clientID, lastSeen := range session.clients {
				if time.Since(lastSeen) > 30*time.Second {
					delete(session.clients, clientID)
				}
			}
			clientCount := len(session.clients)
			session.clientsMux.Unlock()

			// Stop idle sessions
			if clientCount == 0 && time.Since(session.lastActivity) > m.idleTimeout {
				log.Printf("⏰ HLS session idle for %v, stopping: %s", m.idleTimeout, streamID)
				session.Stop()
				delete(m.sessions, streamID)
			}
		}
		m.sessionsMux.Unlock()
	}
}

// cleanupOldFiles removes old HLS files periodically
func (m *HLSManager) cleanupOldFiles() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// Remove directories not in active sessions
		m.sessionsMux.RLock()
		activeDirs := make(map[string]bool)
		for streamID := range m.sessions {
			activeDirs[streamID] = true
		}
		m.sessionsMux.RUnlock()

		// Clean up inactive directories
		dirs, _ := os.ReadDir(m.baseDir)
		for _, dir := range dirs {
			if dir.IsDir() && !activeDirs[dir.Name()] {
				dirPath := filepath.Join(m.baseDir, dir.Name())
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

// GetPlaylistPath returns the path to the HLS playlist
func (s *HLSSession) GetPlaylistPath() string {
	return s.playlistFile
}

// GetOutputDir returns the output directory
func (s *HLSSession) GetOutputDir() string {
	return s.OutputDir
}
