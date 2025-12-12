# IPTV Panel - AI Coding Instructions

## Architecture Overview

This is a dual-stack IPTV management system with Go backend and Vue.js admin panel:

**Backend (Go)**: Manages M3U playlist imports, channel streaming, user authentication, and on-demand FFmpeg-based stream relaying.
**Frontend**: Vue 3 + Element Plus admin template in `pannel/` directory (separate build) + legacy jQuery UI in `static/`

### Key Design Patterns

**On-Demand Streaming**: Streams auto-start when first client connects and auto-stop after 30s idle (see `streaming/manager.go`). One channel = one FFmpeg process shared by all clients, reducing bandwidth by 90%+.

**Dual Authentication System**: 
- Admin auth via session cookies (`handlers/auth.go`, default: admin/admin123)
- User streaming auth via username in M3U URLs (`/mql/{user}.m3u`)

**Database**: SQLite with schema in `database/db.go`. Tables: `playlists`, `channels`, `relays`, `users`, `user_sessions`, `user_connections`, `admins`.

## Critical Workflows

### Running the Application

**Backend only**:
```bash
go run main.go  # Serves on :8080
```

**Full stack with Vue panel**:
```bash
# Terminal 1: Go backend
go run main.go

# Terminal 2: Vue dev server
cd pannel && pnpm run dev  # Serves on :3100
```

**Production build**:
```bash
cd pannel && pnpm run build:pro  # Outputs to pannel/dist
# Backend serves static files from ./static (legacy UI still primary)
```

### Key Routes & Middleware

All `/api/*` routes except `/api/auth/*` and `/api/proxy/*` require `AuthMiddleware` (admin session check).

Static files use `StaticAuthMiddleware` - blocks `.html` files without auth but allows assets.

Stream endpoints (`/stream/{path}`, `/api/proxy/channel/{id}`) validate user credentials via query params or URL path.

### FFmpeg Integration

Stream manager (`streaming/manager.go`) launches FFmpeg with `-c copy -f mpegts` for each active channel. FFmpeg output writes to `RingBuffer` which broadcasts to multiple HTTP clients via `io.Copy`.

**Important**: Always use absolute source URLs. Failover sources in `relays.source_urls` (JSON array).

HLS mode uses `/stream/{path}/hls` endpoint with segment caching in `hls_cache/channel_{id}/`.

## Project-Specific Conventions

### File Organization

- `handlers/*.go` - HTTP handlers, grouped by feature (auth, users, playlists)
- `streaming/*.go` - FFmpeg process management, ring buffers, client tracking
- `models/models.go` - All database model structs
- `static/*.js` - Modular frontend (app.js, playlist.js, channels.js, etc.)

### Database Patterns

Always use transactions for multi-table operations (example: `ImportPlaylist` creates playlist + channels atomically).

User expiration check: Compare `expires_at` with `time.Now()`, redirect to `static/expired.html` if expired.

### M3U Playlist Generation

User playlists generated at `generated_playlists/playlist-{username}.m3u` with URLs like:
```
http://{HOST}/stream/{relay_path}?username={user}&password={pass}
```

Short URL pattern: `/mql/{user}.m3u` (see `handlers.ServeUserPlaylist`)

### Frontend Architecture

**Legacy UI** (`static/`): Modular jQuery with separated JS files - each feature gets own file (241-line HTML vs 1870-line monolith).

**Vue Panel** (`pannel/`): Full admin template, mostly unused - future migration target.

When editing UI, modify `static/index.html` and corresponding JS modules, NOT `static/index.html.backup`.

## Testing & Debugging

**API Testing**:
```bash
./test-api.sh      # Tests playlist/channel endpoints
./test-user.sh     # Tests user management
```

**Stream Monitoring**: GET `/api/streams/status` shows active FFmpeg sessions, client counts, bandwidth.

**Common Issues**:
- FFmpeg not found: Install via `apt install ffmpeg`
- Port 8080 in use: Set `PORT` env var
- Stream won't start: Check source URL accessibility, view logs with `journalctl -u iptv-panel -f`

## External Dependencies

- **FFmpeg**: Required for all streaming operations (HLS, MPEGTS relay)
- **SQLite3**: Database engine, auto-initialized on first run
- **Gorilla Mux**: Router (not standard library)

## DO NOT

- Use `index.html.backup` - it's archived legacy code
- Modify `pannel/` Vue app unless specifically requested (it's not integrated yet)
- Create new authentication schemes - two systems already exist
- Transcode streams by default - use `-c copy` for efficiency
- Store passwords in plain text - use MD5 hash (see `handlers/auth.go:38`)
