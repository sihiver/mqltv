package streaming

import (
	"context"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// StreamSession represents an active stream session
type StreamSession struct {
	ID            string
	SourceURL     string
	SourceURLs    []string // For failover
	ctx           context.Context
	cancel        context.CancelFunc
	clients       map[string]*StreamClient
	clientsMux    sync.RWMutex
	buffer        *RingBuffer
	isActive      bool
	activeMux     sync.RWMutex
	lastActivity  time.Time
	startTime     time.Time
	bytesStreamed int64
}

// StreamClient represents a connected client
type StreamClient struct {
	ID         string
	Connected  time.Time
	RemoteAddr string
	Writer     io.Writer
	Done       chan bool
}

// StreamManager manages all active streams
type StreamManager struct {
	sessions    map[string]*StreamSession
	sessionsMux sync.RWMutex
	idleTimeout time.Duration
}

var (
	globalManager *StreamManager
	managerOnce   sync.Once
)

// GetManager returns the global stream manager instance
func GetManager() *StreamManager {
	managerOnce.Do(func() {
		globalManager = &StreamManager{
			sessions:    make(map[string]*StreamSession),
			idleTimeout: 30 * time.Second, // Stop stream after 30s idle
		}
		go globalManager.monitorSessions()
	})
	return globalManager
}

// GetOrCreateSession gets existing session or creates new one
func (m *StreamManager) GetOrCreateSession(streamID string, sourceURLs []string) *StreamSession {
	m.sessionsMux.Lock()
	defer m.sessionsMux.Unlock()

	if session, exists := m.sessions[streamID]; exists {
		session.lastActivity = time.Now()
		return session
	}

	// Create new session
	ctx, cancel := context.WithCancel(context.Background())
	session := &StreamSession{
		ID:           streamID,
		SourceURLs:   sourceURLs,
		ctx:          ctx,
		cancel:       cancel,
		clients:      make(map[string]*StreamClient),
		buffer:       NewRingBuffer(2 * 1024 * 1024), // 2MB buffer
		lastActivity: time.Now(),
		startTime:    time.Now(),
	}

	m.sessions[streamID] = session
	log.Printf("📺 Created new stream session: %s", streamID)

	return session
}

// AddClient adds a client to the stream session
func (s *StreamSession) AddClient(clientID, remoteAddr string, writer io.Writer) *StreamClient {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	client := &StreamClient{
		ID:         clientID,
		Connected:  time.Now(),
		RemoteAddr: remoteAddr,
		Writer:     writer,
		Done:       make(chan bool, 1),
	}

	s.clients[clientID] = client
	s.lastActivity = time.Now()

	log.Printf("👤 Client connected to stream %s: %s (total: %d)", s.ID, clientID, len(s.clients))

	// Start stream if not active
	if !s.IsActive() {
		go s.Start()
	}

	return client
}

// RemoveClient removes a client from the stream session
func (s *StreamSession) RemoveClient(clientID string) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if _, exists := s.clients[clientID]; exists {
		delete(s.clients, clientID)
		log.Printf("👋 Client disconnected from stream %s: %s (remaining: %d)", s.ID, clientID, len(s.clients))
	}

	s.lastActivity = time.Now()
}

// GetClientCount returns the number of active clients
func (s *StreamSession) GetClientCount() int {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()
	return len(s.clients)
}

// IsActive checks if stream is currently active
func (s *StreamSession) IsActive() bool {
	s.activeMux.RLock()
	defer s.activeMux.RUnlock()
	return s.isActive
}

// Start starts the stream from source
func (s *StreamSession) Start() {
	s.activeMux.Lock()
	if s.isActive {
		s.activeMux.Unlock()
		return
	}
	s.isActive = true
	s.activeMux.Unlock()

	log.Printf("▶️  Starting stream: %s", s.ID)

	// Try each source URL until one works
	var sourceResp *http.Response
	var sourceURL string
	var err error

	for _, url := range s.SourceURLs {
		sourceResp, err = http.Get(url)
		if err == nil && sourceResp.StatusCode == http.StatusOK {
			sourceURL = url
			log.Printf("✅ Connected to source: %s", url)
			break
		}
		if sourceResp != nil {
			sourceResp.Body.Close()
		}
		log.Printf("❌ Failed to connect to source: %s - %v", url, err)
	}

	if sourceResp == nil || sourceResp.StatusCode != http.StatusOK {
		log.Printf("❌ All sources failed for stream: %s", s.ID)
		s.activeMux.Lock()
		s.isActive = false
		s.activeMux.Unlock()
		return
	}

	defer sourceResp.Body.Close()
	s.SourceURL = sourceURL

	// Read from source and broadcast to all clients
	buffer := make([]byte, 32*1024)
	for {
		select {
		case <-s.ctx.Done():
			log.Printf("⏹️  Stream stopped: %s", s.ID)
			s.activeMux.Lock()
			s.isActive = false
			s.activeMux.Unlock()
			return
		default:
			n, err := sourceResp.Body.Read(buffer)
			if n > 0 {
				s.bytesStreamed += int64(n)
				data := make([]byte, n)
				copy(data, buffer[:n])

				// Write to ring buffer
				s.buffer.Write(data)

				// Broadcast to all clients
				s.broadcastToClients(data)
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("⚠️  Stream read error: %s - %v", s.ID, err)
				}
				s.activeMux.Lock()
				s.isActive = false
				s.activeMux.Unlock()
				return
			}
		}
	}
}

// broadcastToClients sends data to all connected clients
func (s *StreamSession) broadcastToClients(data []byte) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	for clientID, client := range s.clients {
		if _, err := client.Writer.Write(data); err != nil {
			log.Printf("⚠️  Error writing to client %s: %v", clientID, err)
			// Client will be removed on next check
		}
	}
}

// Stop stops the stream
func (s *StreamSession) Stop() {
	log.Printf("🛑 Stopping stream: %s", s.ID)
	s.cancel()

	// Close all clients
	s.clientsMux.Lock()
	for clientID, client := range s.clients {
		select {
		case client.Done <- true:
		default:
		}
		delete(s.clients, clientID)
	}
	s.clientsMux.Unlock()
}

// monitorSessions monitors all sessions and stops idle ones
func (m *StreamManager) monitorSessions() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.sessionsMux.Lock()
		for streamID, session := range m.sessions {
			// Check if idle
			if session.GetClientCount() == 0 {
				idleTime := time.Since(session.lastActivity)
				if idleTime > m.idleTimeout {
					log.Printf("⏰ Stream idle for %v, stopping: %s", idleTime, streamID)
					session.Stop()
					delete(m.sessions, streamID)
				}
			}
		}
		m.sessionsMux.Unlock()
	}
}

// GetSessionStats returns statistics for a session
func (s *StreamSession) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"id":              s.ID,
		"active":          s.IsActive(),
		"clients":         s.GetClientCount(),
		"source_url":      s.SourceURL,
		"uptime_seconds":  time.Since(s.startTime).Seconds(),
		"bytes_streamed":  s.bytesStreamed,
		"last_activity":   s.lastActivity,
	}
}

// GetAllSessions returns all active sessions
func (m *StreamManager) GetAllSessions() []*StreamSession {
	m.sessionsMux.RLock()
	defer m.sessionsMux.RUnlock()

	sessions := make([]*StreamSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetSession returns a specific session by ID
func (m *StreamManager) GetSession(streamID string) *StreamSession {
	m.sessionsMux.RLock()
	defer m.sessionsMux.RUnlock()
	return m.sessions[streamID]
}
