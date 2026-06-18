// Channel Management Functions

let allChannels = [];
let channelSearchTimeout = null;

async function loadChannels() {
    const query = document.getElementById('channelSearch').value;
    
    try {
        const response = await fetch(`/api/channels/search?q=${encodeURIComponent(query)}`);
        const channels = await response.json();
        
        allChannels = channels;
        renderChannels(channels);
        updateCategoryFilter(channels);
        updateStreamStatus();
    } catch (error) {
        console.error('Error loading channels:', error);
    }
}

function renderChannels(channels) {
    const container = document.getElementById('channelsList');
    
    if (channels.length === 0) {
        container.innerHTML = '<div class="empty-state"><h3>Tidak ada channel</h3></div>';
        return;
    }
    
    let html = '<table><thead><tr><th>Nama</th><th>Kategori</th><th>Status</th><th>Streaming</th><th>Aksi</th></tr></thead><tbody>';
    
    channels.forEach(channel => {
        const statusBadge = channel.active ? '<span class="badge badge-success">✅ Aktif</span>' : '<span class="badge badge-danger">⏸️ Nonaktif</span>';
        const streamStatus = `<span class="stream-status" data-channel="${channel.id}">⏸️ Stopped</span>`;
        
        html += `
            <tr>
                <td><strong>${channel.name}</strong></td>
                <td>${channel.category || '-'}</td>
                <td>${statusBadge}</td>
                <td>${streamStatus}</td>
                <td>
                    <button class="btn btn-sm" onclick="toggleChannel(${channel.id}, ${channel.active})">
                        ${channel.active ? 'Nonaktifkan' : 'Aktifkan'}
                    </button>
                    <button class="btn btn-danger btn-sm" onclick="deleteChannel(${channel.id})">Hapus</button>
                </td>
            </tr>
        `;
    });
    
    html += '</tbody></table>';
    container.innerHTML = html;
}

function updateCategoryFilter(channels) {
    const categories = [...new Set(channels.map(ch => ch.category))].filter(c => c);
    const select = document.getElementById('categoryFilter');
    const currentValue = select.value;
    
    select.innerHTML = '<option value="">📂 Semua Kategori</option>';
    categories.forEach(cat => {
        const count = channels.filter(ch => ch.category === cat).length;
        const option = document.createElement('option');
        option.value = cat;
        option.textContent = `${cat} (${count})`;
        select.appendChild(option);
    });
    
    select.value = currentValue;
}

function filterByCategory() {
    const category = document.getElementById('categoryFilter').value;
    const deleteBtn = document.getElementById('deleteCategoryBtn');
    
    if (category) {
        deleteBtn.disabled = false;
        const filtered = allChannels.filter(ch => ch.category === category);
        renderChannels(filtered);
    } else {
        deleteBtn.disabled = true;
        renderChannels(allChannels);
    }
}

function debounceSearch() {
    clearTimeout(channelSearchTimeout);
    channelSearchTimeout = setTimeout(loadChannels, 500);
}

async function toggleChannel(id, currentStatus) {
    try {
        const response = await fetch(`/api/channels/${id}/toggle`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ active: !currentStatus })
        });
        
        if (response.ok) {
            loadChannels();
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function deleteChannel(id) {
    if (!confirm('Hapus channel ini?')) return;
    
    try {
        const response = await fetch(`/api/channels/${id}`, { method: 'DELETE' });
        if (response.ok) {
            loadChannels();
            loadStats();
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function deleteCategoryBatch() {
    const category = document.getElementById('categoryFilter').value;
    if (!category) {
        alert('Pilih kategori terlebih dahulu!');
        return;
    }
    
    const filtered = allChannels.filter(ch => ch.category === category);
    if (!confirm(`Hapus ${filtered.length} channel dalam kategori "${category}"?`)) return;
    
    try {
        const response = await fetch('/api/channels/batch-delete', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ category })
        });
        
        if (response.ok) {
            alert('Channels berhasil dihapus!');
            document.getElementById('categoryFilter').value = '';
            loadChannels();
            loadStats();
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

// Stream status monitoring
async function updateStreamStatus() {
    try {
        const response = await fetch('/api/streams/status');
        const data = await response.json();
        
        document.querySelectorAll('.stream-status').forEach(el => {
            const channelId = el.getAttribute('data-channel');
            // Backend returns data in data.data.streams
            const streams = data.data && data.data.streams ? data.data.streams : [];
            const stream = streams.find(s => 
                s.id === `channel_${channelId}` || 
                s.id === `channel_${channelId}_hls` || 
                s.id === `channel-${channelId}`
            );
            
            if (stream) {
                let statusHtml = '';
                if (stream.active) {
                    statusHtml = `🟢 Streaming (${stream.clients} viewers)`;
                    el.style.color = '#28a745';
                } else {
                    statusHtml = '⏸️ Stopped';
                    el.style.color = '#6c757d';
                }

                if (stream.last_error) {
                    // Escape quotes for HTML title attribute
                    const safeError = stream.last_error.replace(/"/g, '&quot;');
                    statusHtml += ` <span title="${safeError}" style="cursor:help;">⚠️ Error Log</span>`;
                }
                
                el.innerHTML = statusHtml;
            } else {
                el.innerHTML = '⏸️ Stopped';
                el.style.color = '#6c757d';
            }
        });
    } catch (error) {
        console.error('Error updating stream status:', error);
    }
}

// Update stream status every 3 seconds
setInterval(updateStreamStatus, 3000);
