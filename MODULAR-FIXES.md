# Modular Structure Fixes

## Overview
Successfully refactored index.html from 1870 lines to 241 lines (87% reduction) by separating into modular files.

## Files Created

### CSS
- **styles.css** (~300 lines): All styling extracted from inline `<style>` tags

### JavaScript Modules
1. **app.js** (60 lines): Core initialization, stats loading, tab switching
2. **playlist.js** (101 lines): Playlist CRUD operations
3. **channels.js** (167 lines): Channel management with search/filter/delete
4. **bandwidth.js** (67 lines): Realtime bandwidth monitoring
5. **relay.js** (89 lines): Relay management functions
6. **users.js** (167 lines): User management CRUD with subscription tracking
7. **generate-playlist.js** (272 lines): Playlist generator with channel selection

## Issues Fixed

### 1. Function Name Collision
**Problem**: `toggleChannel()` was defined in both `channels.js` and `generate-playlist.js`

**Solution**: Renamed function in `generate-playlist.js` to `togglePlaylistChannel()`
- Updated function declaration
- Updated HTML rendering in `renderChannelSelection()`

### 2. Race Condition on Page Load
**Problem**: `app.js` was calling `loadPlaylists()`, `loadUsers()`, `startBandwidthMonitoring()` immediately, but these functions might not be loaded yet from other modules.

**Solution**: Wrapped initialization in `DOMContentLoaded` event listener
```javascript
window.addEventListener('DOMContentLoaded', function() {
    loadStats();
    loadPlaylists();
    startBandwidthMonitoring();
});
```

### 3. Premature Initialization
**Problem**: `generate-playlist.js` was auto-loading users on page load with setTimeout

**Solution**: Removed auto-initialization, now only loads when tab is switched:
- Tab switching handled by `switchTab()` in `app.js`
- Calls `loadPlaylistUsers()` only when 'generate-playlist' tab is opened

## Module Dependencies

### Global Variables (shared across modules)
Defined in `app.js`:
- `currentPlaylist`: Currently selected playlist ID
- `bandwidthMonitorInterval`: Interval timer for bandwidth monitoring
- `selectedChannels`: Set of selected channel IDs for playlist generation
- `allPlaylistChannels`: Array of all channels for filtering

### Function Call Chain
```
Page Load (index-new.html)
  └─> DOMContentLoaded event
      ├─> loadStats() [app.js]
      ├─> loadPlaylists() [playlist.js]
      └─> startBandwidthMonitoring() [bandwidth.js]

Tab Switch
  └─> switchTab(tabName) [app.js]
      ├─> 'channels' → loadChannels() [channels.js]
      ├─> 'relays' → loadRelays() [relay.js]
      ├─> 'users' → loadUsers() [users.js]
      └─> 'generate-playlist' → loadPlaylistUsers() [generate-playlist.js]
```

## Testing Checklist

### ✅ Playlists Tab
- [ ] Load playlists on page load
- [ ] View channels from playlist
- [ ] Export playlist with panel proxy URLs
- [ ] Delete playlist

### ✅ Channels Tab
- [ ] Load channels with search
- [ ] Category filter working
- [ ] Batch delete by category
- [ ] Toggle channel active/inactive
- [ ] Realtime stream status update

### ✅ Relays Tab
- [ ] Load relays list
- [ ] Create new relay
- [ ] Delete relay
- [ ] Cancel creation form

### ✅ Users Tab
- [ ] Load users with status badges (Active/Warning/Expired/Inactive)
- [ ] Create new user with duration
- [ ] Edit user (update name + extend subscription)
- [ ] Delete user
- [ ] Reset password

### ✅ Generate Playlist Tab
- [ ] Load user dropdown
- [ ] Display user info with status
- [ ] Load 1762 channels with pagination (200 limit)
- [ ] Search channels
- [ ] Filter by category
- [ ] Select/deselect channels
- [ ] Select all visible
- [ ] Select by category
- [ ] Generate and download M3U file

### ✅ Import M3U Tab
- [ ] Import playlist from URL
- [ ] Show import status

### ✅ Bandwidth Monitor
- [ ] Display realtime download/upload Mbps
- [ ] Show total MB downloaded/uploaded
- [ ] Update every 3 seconds

## Performance Notes

### Optimizations Applied
1. **Pagination**: Display limit of 200 channels in generate-playlist to prevent browser hang
2. **Lazy Loading**: Functions only called when tabs are switched
3. **Event Delegation**: Uses DOMContentLoaded to ensure DOM is ready
4. **Debounced Search**: Channel search uses debouncing in channels.js

### Known Limitations
- Generate Playlist: Shows max 200 channels at a time (use search/filter for specific channels)
- Bandwidth Monitor: Updates every 3 seconds (can cause CPU usage with many streams)
- Stream Status: Updates every 3 seconds in channels tab only

## File Structure
```
static/
├── index-new.html          (241 lines) - Clean modular HTML
├── index.html.backup       (1870 lines) - Original monolithic file
├── styles.css              (~300 lines) - All extracted CSS
├── app.js                  (60 lines) - Core initialization
├── playlist.js             (101 lines) - Playlist management
├── channels.js             (167 lines) - Channel management
├── bandwidth.js            (67 lines) - Bandwidth monitoring
├── relay.js                (89 lines) - Relay management
├── users.js                (167 lines) - User management
└── generate-playlist.js    (272 lines) - Playlist generator
```

## Migration Steps

### Current State
- `index.html` - Modified (inline CSS removed, needs full replacement)
- `index-new.html` - New modular version (ready to test)
- `index.html.backup` - Original backup (1870 lines)

### To Complete Migration
1. Test all features in `index-new.html`
2. Verify no JavaScript console errors
3. Test each tab thoroughly
4. Once stable:
   ```bash
   cd /home/dindin/mqltv/static
   mv index.html index-old.html
   mv index-new.html index.html
   ```

## Rollback Plan
If issues found:
```bash
cd /home/dindin/mqltv/static
mv index.html index-broken.html
cp index.html.backup index.html
```

## Next Steps
1. Test index-new.html thoroughly in browser
2. Fix any remaining issues found
3. Replace index.html with index-new.html
4. Update README.md with modular structure info
5. Consider adding authentication middleware for stream endpoints
