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
	outageClipOnce sync.Once
	outageClipData []byte
)

func mpeg2CRC32(data []byte) uint32 {
	crc := uint32(0xFFFFFFFF)
	for _, b := range data {
		crc ^= uint32(b) << 24
		for i := 0; i < 8; i++ {
			if (crc & 0x80000000) != 0 {
				crc = (crc << 1) ^ 0x04C11DB7
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// patchOutageTSPackets rewrites PAT (0x0000) and PMT (0x1000) version_number to `version`
// and sets discontinuity_indicator=1 on the first PCR packet so VLC/ExoPlayer immediately
// switches decoders and resets the PCR timeline when transitioning mid-stream ("di tengah jalan").
func patchOutageTSPackets(raw []byte, version uint8) []byte {
	const tsSize = 188
	alignedLen := (len(raw) / tsSize) * tsSize
	out := make([]byte, alignedLen)
	copy(out, raw[:alignedLen])

	pcrDiscontinuitySet := false

	for offset := 0; offset < alignedLen; offset += tsSize {
		pkt := out[offset : offset+tsSize]
		if pkt[0] != 0x47 {
			continue
		}
		pid := (uint16(pkt[1]&0x1F) << 8) | uint16(pkt[2])
		pusi := (pkt[1] & 0x40) != 0
		afc := (pkt[3] >> 4) & 0x03

		// Set discontinuity_indicator = 1 on the first adaptation field of video/PCR PID 0x0200
		if !pcrDiscontinuitySet && pid == 0x0200 && (afc&0x02) != 0 && pkt[4] >= 1 {
			pkt[5] |= 0x80 // discontinuity_indicator = 1
			pcrDiscontinuitySet = true
		}

		// Patch PSI version_number on PAT (0x0000) and PMT (0x1000)
		if (pid == 0x0000 || pid == 0x1000) && pusi && (afc&0x01) != 0 {
			pay := 4
			if (afc & 0x02) != 0 {
				pay += 1 + int(pkt[4])
			}
			if pay < tsSize {
				sec := pay + 1 + int(pkt[pay])
				if sec+8 <= tsSize {
					secLen := (int(pkt[sec+1]&0x0F) << 8) | int(pkt[sec+2])
					totalSec := 3 + secLen
					if sec+totalSec <= tsSize && totalSec >= 12 {
						pkt[sec+5] = (pkt[sec+5] & 0xC1) | ((version & 0x1F) << 1) | 0x01
						crc := mpeg2CRC32(pkt[sec : sec+totalSec-4])
						pkt[sec+totalSec-4] = byte(crc >> 24)
						pkt[sec+totalSec-3] = byte(crc >> 16)
						pkt[sec+totalSec-2] = byte(crc >> 8)
						pkt[sec+totalSec-1] = byte(crc)
					}
				}
			}
		}
	}
	return out
}

func getOrBuildOutageTSClip() []byte {
	outageClipOnce.Do(func() {
		videoPath := "./static/gangguan-notification.mp4"
		imagePath := "./static/gangguan-notification.png"
		ffmpegExe := getFFmpegExecutable()

		if _, err := os.Stat(videoPath); err != nil {
			if _, imgErr := os.Stat(imagePath); imgErr == nil {
				genCmd := exec.Command(
					ffmpegExe,
					"-y",
					"-loop", "1",
					"-framerate", "25",
					"-i", imagePath,
					"-f", "lavfi",
					"-i", "anullsrc=r=48000:cl=stereo",
					"-t", "4",
					"-c:v", "libx264",
					"-preset", "fast",
					"-tune", "stillimage",
					"-pix_fmt", "yuv420p",
					"-g", "12",
					"-keyint_min", "12",
					"-sc_threshold", "0",
					"-bf", "0",
					"-c:a", "mp2",
					"-b:a", "192k",
					"-shortest",
					videoPath,
				)
				_ = genCmd.Run()
			}
		}

		// Convert MP4 to a self-contained 4-second MPEG-TS clip on PID 0x0200/0x0201
		muxCmd := exec.Command(
			ffmpegExe,
			"-v", "quiet",
			"-i", videoPath,
			"-c", "copy",
			"-f", "mpegts",
			"-mpegts_start_pid", "0x0200",
			"-mpegts_flags", "+resend_headers+initial_discontinuity",
			"pipe:1",
		)
		if raw, err := muxCmd.Output(); err == nil && len(raw) >= 188 {
			outageClipData = patchOutageTSPackets(raw, 5)
			log.Printf("✅ Prepared mid-stream outage MPEG-TS clip (%d bytes, PID 0x0200, PMT v5)", len(outageClipData))
		}
	})
	return outageClipData
}

// StreamOutageVideo streams the outage/disruption ("gangguan sinyal") screen in an infinite loop.
// Works both on fresh connections and when spliced mid-stream ("di tengah jalan") in VLC/ExoPlayer/STB.
func StreamOutageVideo(w http.ResponseWriter, r *http.Request) {
	clip := getOrBuildOutageTSClip()
	htmlPath := "./static/gangguan.html"

	if len(clip) == 0 {
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

	w.Header().Set("Content-Type", "video/MP2T")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	// Send the first burst immediately (PAT + PMT v5 + PCR discontinuity + first IDR frame)
	// so VLC/ExoPlayer switches immediately without waiting for demuxer buffer to fill,
	// then pace the remaining loop chunks over 4 seconds per cycle.
	const chunkSize = 188 * 20 // 3760 bytes per chunk
	const burstBytes = 188 * 200

	for loop := 0; ; loop++ {
		// On every loop iteration, re-patch with discontinuity_indicator=1 on the first PCR packet
		offset := 0
		if loop == 0 && len(clip) > burstBytes {
			if _, err := w.Write(clip[:burstBytes]); err != nil {
				return
			}
			flusher.Flush()
			offset = burstBytes
		}

		remaining := len(clip) - offset
		numChunks := (remaining + chunkSize - 1) / chunkSize
		sleepPerChunk := 35 * time.Millisecond
		if numChunks > 0 {
			sleepPerChunk = (3500 * time.Millisecond) / time.Duration(numChunks)
		}

		for offset < len(clip) {
			end := offset + chunkSize
			if end > len(clip) {
				end = len(clip)
			}
			select {
			case <-r.Context().Done():
				return
			default:
				if _, err := w.Write(clip[offset:end]); err != nil {
					return
				}
				flusher.Flush()
				offset = end
				time.Sleep(sleepPerChunk)
			}
		}
	}
}
