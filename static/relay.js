// Relay Management - Full functions from original index.html
// (Placeholder - to be extracted from original file)
console.log('Relay module loaded');

async function loadRelays() {
    try {
        const response = await fetch('/api/relays');
        const relays = await response.json();
        
        const container = document.getElementById('relaysList');
        if (relays.length === 0) {
            container.innerHTML = '<div class="empty-state"><h3>Belum ada relay</h3></div>';
            return;
        }
        
        // Render relays list
        let html = '<table><thead><tr><th>Path</th><th>Sources</th><th>Status</th><th>Aksi</th></tr></thead><tbody>';
        relays.forEach(relay => {
            html += `
                <tr>
                    <td><strong>/stream/${relay.output_path}</strong></td>
                    <td>${relay.source_urls ? JSON.parse(relay.source_urls).length : 0} sources</td>
                    <td>${relay.active ? '<span class="badge badge-success">Aktif</span>' : '<span class="badge badge-secondary">Nonaktif</span>'}</td>
                    <td>
                        <button class="btn btn-danger btn-sm" onclick="deleteRelay(${relay.id})">Hapus</button>
                    </td>
                </tr>
            `;
        });
        html += '</tbody></table>';
        container.innerHTML = html;
    } catch (error) {
        console.error('Error loading relays:', error);
    }
}

function showCreateRelay() {
    document.getElementById('createRelayForm').style.display = 'block';
}

function hideCreateRelay() {
    document.getElementById('createRelayForm').style.display = 'none';
}

async function createRelay() {
    const path = document.getElementById('relayPath').value;
    const urlsText = document.getElementById('relayUrls').value;
    
    if (!path || !urlsText) {
        alert('Path dan URLs harus diisi!');
        return;
    }
    
    const urls = urlsText.split('\n').filter(u => u.trim());
    
    try {
        const response = await fetch('/api/relays', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ output_path: path, source_urls: urls })
        });
        
        if (response.ok) {
            alert('Relay berhasil dibuat!');
            hideCreateRelay();
            document.getElementById('relayPath').value = '';
            document.getElementById('relayUrls').value = '';
            loadRelays();
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function deleteRelay(id) {
    if (!confirm('Hapus relay ini?')) return;
    
    try {
        const response = await fetch(`/api/relays/${id}`, { method: 'DELETE' });
        if (response.ok) {
            loadRelays();
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}
