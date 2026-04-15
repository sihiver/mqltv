package handlers

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type hlsCacheEntry struct {
	url       string
	filePath  string
	size      int64
	expiresAt time.Time
}

type hlsProxyCache struct {
	dir          string
	maxBytes     int64
	maxFileBytes int64

	segmentTTL time.Duration
	keyTTL     time.Duration
	playlistTTL time.Duration

	client *http.Client

	mu         sync.Mutex
	entries    map[string]*hlsCacheEntry
	order      []*hlsCacheEntry // simple LRU-ish order (front=most recent)
	totalBytes int64

	inflight map[string]*inflightFetch
}

type inflightFetch struct {
	wg  sync.WaitGroup
	err error
	ent *hlsCacheEntry
}

func newHLSProxyCacheFromEnv() *hlsProxyCache {
	dir := os.Getenv("HLS_CACHE_DIR")
	if dir == "" {
		dir = "/tmp/mqltv_hls_cache"
	}

	maxMB := int64(48)
	if v := os.Getenv("HLS_CACHE_MAX_MB"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxMB = n
		}
	}

	c := &hlsProxyCache{
		dir:          dir,
		maxBytes:     maxMB * 1024 * 1024,
		maxFileBytes: 8 * 1024 * 1024,
		segmentTTL:   20 * time.Second,
		keyTTL:       10 * time.Minute,
		playlistTTL:  3 * time.Second,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		entries:  map[string]*hlsCacheEntry{},
		order:    make([]*hlsCacheEntry, 0, 1024),
		inflight: map[string]*inflightFetch{},
	}

	_ = os.MkdirAll(dir, 0o755)
	return c
}

func (c *hlsProxyCache) ttlForURL(u string) time.Duration {
	lower := strings.ToLower(u)
	switch {
	case strings.Contains(lower, ".m3u8"):
		return c.playlistTTL
	case strings.Contains(lower, ".key"):
		return c.keyTTL
	case strings.Contains(lower, "ext-x-key"):
		return c.keyTTL
	default:
		return c.segmentTTL
	}
}

func (c *hlsProxyCache) contentTypeGuess(u string) string {
	lower := strings.ToLower(u)
	if strings.Contains(lower, ".m3u8") {
		return "application/vnd.apple.mpegurl"
	}
	if strings.Contains(lower, ".key") {
		return "application/octet-stream"
	}
	return "video/MP2T"
}

func (c *hlsProxyCache) keyToPath(u string) string {
	sum := sha1.Sum([]byte(u))
	hexsum := hex.EncodeToString(sum[:])
	sub := hexsum[:2]
	return filepath.Join(c.dir, sub, hexsum)
}

func (c *hlsProxyCache) getFreshEntryLocked(u string) (*hlsCacheEntry, bool) {
	ent, ok := c.entries[u]
	if !ok {
		return nil, false
	}
	if time.Now().After(ent.expiresAt) {
		c.removeLocked(ent)
		return nil, false
	}
	if _, err := os.Stat(ent.filePath); err != nil {
		c.removeLocked(ent)
		return nil, false
	}
	c.bumpLocked(ent)
	return ent, true
}

func (c *hlsProxyCache) bumpLocked(ent *hlsCacheEntry) {
	// Move to front (simple slice LRU)
	idx := -1
	for i, e := range c.order {
		if e == ent {
			idx = i
			break
		}
	}
	if idx >= 0 {
		copy(c.order[idx:], c.order[idx+1:])
		c.order[len(c.order)-1] = nil
		c.order = c.order[:len(c.order)-1]
	}
	c.order = append([]*hlsCacheEntry{ent}, c.order...)
}

func (c *hlsProxyCache) removeLocked(ent *hlsCacheEntry) {
	delete(c.entries, ent.url)
	// Remove from order
	idx := -1
	for i, e := range c.order {
		if e == ent {
			idx = i
			break
		}
	}
	if idx >= 0 {
		copy(c.order[idx:], c.order[idx+1:])
		c.order[len(c.order)-1] = nil
		c.order = c.order[:len(c.order)-1]
	}
	if ent.size > 0 {
		c.totalBytes -= ent.size
		if c.totalBytes < 0 {
			c.totalBytes = 0
		}
	}
	_ = os.Remove(ent.filePath)
}

func (c *hlsProxyCache) evictIfNeededLocked() {
	for c.totalBytes > c.maxBytes && len(c.order) > 0 {
		victim := c.order[len(c.order)-1]
		c.removeLocked(victim)
	}
}

func (c *hlsProxyCache) fetchToFile(ctx context.Context, upstream string) (*hlsCacheEntry, error) {
	// Deduplicate concurrent fetches for the same upstream URL
	c.mu.Lock()
	if ent, ok := c.getFreshEntryLocked(upstream); ok {
		c.mu.Unlock()
		return ent, nil
	}
	if infl, ok := c.inflight[upstream]; ok {
		infl.wg.Add(1)
		c.mu.Unlock()
		infl.wg.Wait()
		return infl.ent, infl.err
	}
	infl := &inflightFetch{}
	infl.wg.Add(1)
	c.inflight[upstream] = infl
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.inflight, upstream)
		infl.wg.Done()
		c.mu.Unlock()
	}()

	// Prepare destination
	dstPath := c.keyToPath(upstream)
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		infl.err = err
		return nil, err
	}
	tmpPath := dstPath + ".tmp"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream, nil)
	if err != nil {
		infl.err = err
		return nil, err
	}
	req.Header.Set("User-Agent", "iptv-panel")

	resp, err := c.client.Do(req)
	if err != nil {
		infl.err = err
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		infl.err = errors.New("upstream returned non-2xx")
		return nil, infl.err
	}

	f, err := os.Create(tmpPath)
	if err != nil {
		infl.err = err
		return nil, err
	}

	limited := io.LimitReader(resp.Body, c.maxFileBytes+1)
	n, copyErr := io.Copy(f, limited)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		infl.err = copyErr
		return nil, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		infl.err = closeErr
		return nil, closeErr
	}
	if n > c.maxFileBytes {
		_ = os.Remove(tmpPath)
		infl.err = errors.New("segment too large to cache")
		return nil, infl.err
	}

	if err := os.Rename(tmpPath, dstPath); err != nil {
		_ = os.Remove(tmpPath)
		infl.err = err
		return nil, err
	}

	ent := &hlsCacheEntry{
		url:       upstream,
		filePath:  dstPath,
		size:      n,
		expiresAt: time.Now().Add(c.ttlForURL(upstream)),
	}

	c.mu.Lock()
	c.entries[upstream] = ent
	c.totalBytes += n
	c.bumpLocked(ent)
	c.evictIfNeededLocked()
	c.mu.Unlock()

	infl.ent = ent
	return ent, nil
}

func (c *hlsProxyCache) ServeUpstream(w http.ResponseWriter, r *http.Request, upstream string) {
	ct := c.contentTypeGuess(upstream)

	// Try cache
	c.mu.Lock()
	ent, ok := c.getFreshEntryLocked(upstream)
	c.mu.Unlock()

	if ok {
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "max-age=10")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Cache", "HIT")
		f, err := os.Open(ent.filePath)
		if err != nil {
			// Fall back to miss path
			ok = false
		} else {
			defer f.Close()
			w.WriteHeader(http.StatusOK)
			_, _ = io.Copy(w, f)
			return
		}
	}

	// Cache miss: attempt fetch+cache; if caching fails, proxy directly.
	ent, err := c.fetchToFile(r.Context(), upstream)
	if err == nil && ent != nil {
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "max-age=10")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Cache", "MISS")
		f, err2 := os.Open(ent.filePath)
		if err2 == nil {
			defer f.Close()
			w.WriteHeader(http.StatusOK)
			_, _ = io.Copy(w, f)
			return
		}
	}

	// Direct proxy (no cache)
	req, errReq := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream, nil)
	if errReq != nil {
		http.Error(w, "Invalid upstream URL", http.StatusBadRequest)
		return
	}
	req.Header.Set("User-Agent", "iptv-panel")
	resp, errDo := c.client.Do(req)
	if errDo != nil {
		http.Error(w, "Failed to fetch segment", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "max-age=10")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Cache", "BYPASS")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

var defaultHLSProxyCache = newHLSProxyCacheFromEnv()
