# 🧪 Testing Guide - IPTV Panel

## ✅ Testing Fitur On-Demand & Multi-Client

### 1. Test Direct Stream Relay (Multi-Client)

**Setup Test Relay:**
```bash
# Buat relay dengan source HLS
curl -X POST http://localhost:8080/api/relays \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Stream",
    "output_path": "test",
    "source_urls": [
      "https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8"
    ]
  }'
```

**Test Multi-Client (Terminal 1):**
```bash
# Client 1 - Direct stream
curl http://localhost:8080/stream/test > /dev/null &
CLIENT1_PID=$!
```

**Test Multi-Client (Terminal 2):**
```bash
# Client 2 - Menggunakan stream yang sama
curl http://localhost:8080/stream/test > /dev/null &
CLIENT2_PID=$!
```

**Check Status:**
```bash
# Lihat berapa banyak client yang terkoneksi
curl http://localhost:8080/api/streams/status | jq

# Output akan menunjukkan:
# {
#   "total_streams": 1,
#   "streams": [{
#     "id": "test",
#     "clients": 2,  # 2 clients, tapi hanya 1 koneksi ke source!
#     "active": true
#   }]
# }
```

**Cleanup:**
```bash
kill $CLIENT1_PID $CLIENT2_PID
```

### 2. Test Auto Start/Stop

**Step 1: Pastikan stream idle**
```bash
curl http://localhost:8080/api/streams/status
# Output: {"total_streams":0,"streams":[]}
```

**Step 2: Mulai client pertama**
```bash
curl http://localhost:8080/stream/test > /dev/null &
CLIENT_PID=$!
```

**Step 3: Cek stream otomatis start**
```bash
sleep 2
curl http://localhost:8080/api/streams/status
# Output: {"total_streams":1,"streams":[...]}
# Stream otomatis START! ✅
```

**Step 4: Stop client**
```bash
kill $CLIENT_PID
```

**Step 5: Tunggu auto-stop (30 detik)**
```bash
# Tunggu 30 detik...
sleep 35

# Cek stream otomatis stop
curl http://localhost:8080/api/streams/status
# Output: {"total_streams":0,"streams":[]}
# Stream otomatis STOP! ✅
```

### 3. Test HLS Endpoint

**PENTING:** HLS endpoint hanya bekerja jika:
- Source sudah dalam format HLS (.m3u8)
- Atau menggunakan relay yang source-nya HLS

**Test HLS Relay:**
```bash
# Buat relay dengan HLS source
curl -X POST http://localhost:8080/api/relays \
  -H "Content-Type: application/json" \
  -d '{
    "name": "HLS Test",
    "output_path": "hls-test",
    "source_urls": [
      "https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8"
    ]
  }'

# Akses HLS endpoint (akan redirect ke source HLS)
curl -L http://localhost:8080/stream/hls-test/hls

# Atau gunakan VLC
vlc http://localhost:8080/stream/hls-test/hls
```

**Test dengan Channel:**
```bash
# Cari channel yang sudah ada
curl http://localhost:8080/api/channels/search?q=sport | jq

# Ambil ID channel yang URL-nya .m3u8
# Misalnya channel ID 1

# Test HLS
curl -L http://localhost:8080/api/proxy/channel/1/hls
```

### 4. Test Failover

**Setup relay dengan multiple sources:**
```bash
curl -X POST http://localhost:8080/api/relays \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Failover Test",
    "output_path": "failover",
    "source_urls": [
      "http://invalid-source-1.com/stream.m3u8",
      "http://invalid-source-2.com/stream.m3u8",
      "https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8"
    ]
  }'
```

**Test:**
```bash
# Akan otomatis failover ke source ke-3 yang valid
curl http://localhost:8080/stream/failover > /dev/null &
sleep 2

# Check logs untuk melihat failover
tail -20 /tmp/iptv.log | grep -i "failed\|connected"
```

### 5. Test Performance Multi-Client

**Benchmark berapa banyak client yang bisa ditangani:**
```bash
#!/bin/bash

# Mulai 10 clients sekaligus
for i in {1..10}; do
    curl http://localhost:8080/stream/test > /dev/null 2>&1 &
    echo "Started client $i"
done

sleep 3

# Cek status
curl http://localhost:8080/api/streams/status | jq

# Expected output:
# "clients": 10,  # 10 clients terhubung
# "active": true
# Hanya 1 koneksi ke source!

# Cleanup
pkill -f "curl http://localhost:8080/stream"
```

### 6. Test Resource Usage

**Monitor memory dan CPU:**
```bash
# Terminal 1: Monitor resource
watch -n 1 'ps aux | grep iptv-panel | grep -v grep'

# Terminal 2: Start multiple streams
for i in {1..5}; do
    curl http://localhost:8080/stream/test > /dev/null &
done

# Lihat memory usage tetap rendah karena multi-client sharing
```

### 7. Integration Test dengan VLC

**Test 1: Direct Stream**
```bash
vlc http://localhost:8080/stream/test
```

**Test 2: Multiple VLC Instances**
```bash
# Terminal 1
vlc http://localhost:8080/stream/test &

# Terminal 2
vlc http://localhost:8080/stream/test &

# Terminal 3
vlc http://localhost:8080/stream/test &

# Check - hanya 1 koneksi ke source!
curl http://localhost:8080/api/streams/status | jq
```

**Test 3: HLS Stream (jika source HLS)**
```bash
vlc http://localhost:8080/stream/hls-test/hls
```

## 🔍 Debugging

### Check Server Logs
```bash
tail -f /tmp/iptv.log

# Look for:
# - "Created new stream session" = Stream started
# - "Client connected" = New viewer
# - "Client disconnected" = Viewer left
# - "Stream idle" = Auto-stop triggered
```

### Check Active Streams
```bash
# Continuously monitor
watch -n 1 'curl -s http://localhost:8080/api/streams/status | jq'
```

### Check Database
```bash
sqlite3 iptv.db "SELECT * FROM relays;"
sqlite3 iptv.db "SELECT COUNT(*) FROM channels;"
```

### Check Network Connections
```bash
# Lihat berapa koneksi yang dibuat
netstat -an | grep :8080 | grep ESTABLISHED | wc -l

# vs

# Lihat berapa koneksi ke source (seharusnya hanya 1 per stream)
netstat -an | grep ESTABLISHED | grep -v :8080
```

## ✅ Expected Results

### Multi-Client Test
- ✅ 10 clients connected
- ✅ Only 1 connection to source
- ✅ All clients receive same stream
- ✅ 90%+ bandwidth savings

### Auto Start/Stop Test
- ✅ Stream starts when first client connects
- ✅ Stream runs while clients are connected
- ✅ Stream stops 30s after last client disconnects
- ✅ 95%+ resource savings when idle

### Failover Test
- ✅ Tries source 1: fails
- ✅ Tries source 2: fails
- ✅ Tries source 3: success
- ✅ All clients get stream from source 3

## 📊 Performance Metrics

**What to measure:**
```bash
# 1. Memory usage (should be low)
ps aux | grep iptv-panel | awk '{print $6/1024 " MB"}'

# 2. Number of active streams
curl -s http://localhost:8080/api/streams/status | jq '.total_streams'

# 3. Total clients across all streams
curl -s http://localhost:8080/api/streams/status | jq '[.streams[].clients] | add'

# 4. Bandwidth per stream (from logs)
grep "bytes_streamed" /tmp/iptv.log
```

## 🎯 Success Criteria

✅ **Multi-Client Working** if:
- Multiple clients can connect to same stream
- Server shows `clients: N` where N > 1
- Only 1 connection to source provider

✅ **Auto Start/Stop Working** if:
- Stream status goes from 0 → 1 when client connects
- Stream status goes from 1 → 0 after idle timeout
- Logs show "Stream idle" and "Stopping stream"

✅ **HLS Working** if:
- HLS endpoint returns M3U8 playlist
- VLC can play the HLS stream
- Source must be HLS format (.m3u8)

## 🐛 Common Issues

### Issue: "VLC unable to open MRL"
**Cause:** Source channel is not HLS format or channel doesn't exist

**Solution:**
```bash
# Check if channel exists and get its URL
curl http://localhost:8080/api/playlists/1/channels | jq '.[0]'

# If URL doesn't end with .m3u8, use direct proxy instead:
vlc http://localhost:8080/api/proxy/channel/1

# NOT the /hls endpoint (that's only for HLS sources)
```

### Issue: "Stream not auto-stopping"
**Check:**
```bash
# Make sure all clients are disconnected
curl http://localhost:8080/api/streams/status

# Wait full 30 seconds for timeout
sleep 35 && curl http://localhost:8080/api/streams/status
```

### Issue: "Multiple connections to source"
**This means multi-client is not working. Check:**
```bash
# Verify stream manager is active
curl http://localhost:8080/api/streams/status

# Should show multiple clients but same stream ID
```

---

**Happy Testing! 🧪🎉**
