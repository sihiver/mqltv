// Playlist Management Functions

async function loadPlaylists() {
    try {
        const response = await fetch('/api/playlists');
        const playlists = await response.json();
        
        const container = document.getElementById('playlistsList');
        if (playlists.length === 0) {
            container.innerHTML = '<div class="empty-state"><h3>Belum ada playlist</h3><p>Import playlist M3U untuk memulai</p></div>';
            return;
        }
        
        let html = '<table><thead><tr><th>Nama</th><th>URL</th><th>Jumlah Channel</th><th>Aksi</th></tr></thead><tbody>';
        
        playlists.forEach(playlist => {
            html += `
                <tr>
                    <td><strong>${playlist.name}</strong></td>
                    <td><code>${playlist.url}</code></td>
                    <td>${playlist.channel_count || 0} channels</td>
                    <td>
                        <button class="btn btn-primary btn-sm" onclick="viewChannels(${playlist.id}, '${playlist.name}')">Lihat Channels</button>
                        <button class="btn btn-sm" onclick="exportPlaylist(${playlist.id})">📥 Export</button>
                        <button class="btn btn-danger btn-sm" onclick="deletePlaylist(${playlist.id})">Hapus</button>
                    </td>
                </tr>
            `;
        });
        
        html += '</tbody></table>';
        container.innerHTML = html;
    } catch (error) {
        console.error('Error loading playlists:', error);
    }
}

function viewChannels(playlistId, playlistName) {
    currentPlaylist = playlistId;
    document.getElementById('channelSearch').value = '';
    switchTab('channels');
    loadChannels();
}

function exportPlaylist(playlistId) {
    window.open(`/api/playlists/${playlistId}/export`, '_blank');
}

async function deletePlaylist(id) {
    if (!confirm('Hapus playlist ini dan semua channelnya?')) return;
    
    try {
        const response = await fetch(`/api/playlists/${id}`, { method: 'DELETE' });
        if (response.ok) {
            alert('Playlist berhasil dihapus!');
            loadPlaylists();
            loadStats();
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function importPlaylist() {
    const name = document.getElementById('playlistName').value;
    const url = document.getElementById('playlistUrl').value;
    
    if (!name || !url) {
        alert('Nama dan URL harus diisi!');
        return;
    }
    
    const statusDiv = document.getElementById('importStatus');
    statusDiv.innerHTML = '<div class="alert">⏳ Importing playlist...</div>';
    
    try {
        const response = await fetch('/api/playlists/import', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, url })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            statusDiv.innerHTML = `<div class="alert alert-success">✅ ${result.message}</div>`;
            document.getElementById('playlistName').value = '';
            document.getElementById('playlistUrl').value = '';
            setTimeout(() => {
                statusDiv.innerHTML = '';
                loadPlaylists();
                loadStats();
            }, 2000);
        } else {
            statusDiv.innerHTML = `<div class="alert alert-error">❌ ${result.error}</div>`;
        }
    } catch (error) {
        statusDiv.innerHTML = `<div class="alert alert-error">❌ Error: ${error.message}</div>`;
    }
}
