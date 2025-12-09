// Generate Playlist Functions

async function loadPlaylistUsers() {
    try {
        const response = await fetch('/api/users');
        const users = await response.json();
        
        const select = document.getElementById('playlistUser');
        select.innerHTML = '<option value="">-- Pilih User --</option>';
        
        users.forEach(user => {
            const option = document.createElement('option');
            option.value = user.id;
            option.textContent = `${user.username} - ${user.full_name || user.username}`;
            option.dataset.username = user.username;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Error loading users:', error);
    }
}

async function loadUserInfo() {
    const select = document.getElementById('playlistUser');
    const userId = select.value;
    
    if (!userId) {
        document.getElementById('userInfoBox').style.display = 'none';
        document.getElementById('channelSelectionList').innerHTML = '<p style="text-align: center; color: #999;">Pilih user terlebih dahulu</p>';
        document.getElementById('generateBtn').disabled = true;
        return;
    }
    
    try {
        const response = await fetch('/api/users');
        const users = await response.json();
        const user = users.find(u => u.id == userId);
        
        if (user) {
            const username = select.options[select.selectedIndex].dataset.username;
            document.getElementById('playlistFileName').value = `playlist-${username}.m3u`;
            
            let statusBadge = '';
            if (user.is_expired) {
                statusBadge = '<span style="background: #f44336; color: white; padding: 3px 8px; border-radius: 4px;">❌ Expired</span>';
            } else if (!user.is_active) {
                statusBadge = '<span style="background: #9e9e9e; color: white; padding: 3px 8px; border-radius: 4px;">⏸️ Nonaktif</span>';
            } else if (user.days_remaining > 0 && user.days_remaining <= 7) {
                statusBadge = '<span style="background: #ff9800; color: white; padding: 3px 8px; border-radius: 4px;">⚠️ ' + user.days_remaining + ' hari</span>';
            } else {
                statusBadge = '<span style="background: #4caf50; color: white; padding: 3px 8px; border-radius: 4px;">✅ Aktif</span>';
            }
            
            const expiryText = user.expires_at ? new Date(user.expires_at).toLocaleDateString('id-ID') : 'Unlimited';
            
            document.getElementById('userInfoContent').innerHTML = `
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px;">
                    <div><strong>Username:</strong> ${user.username}</div>
                    <div><strong>Status:</strong> ${statusBadge}</div>
                    <div><strong>Max Koneksi:</strong> ${user.max_connections} device</div>
                    <div><strong>Expired:</strong> ${expiryText}</div>
                </div>
            `;
            document.getElementById('userInfoBox').style.display = 'block';
        }
        
        await loadPlaylistChannels();
        
    } catch (error) {
        console.error('Error loading user info:', error);
    }
}

async function loadPlaylistChannels() {
    const loadingStatus = document.getElementById('playlistLoadingStatus');
    loadingStatus.style.display = 'block';
    loadingStatus.textContent = '⏳ Loading channels...';
    
    try {
        const response = await fetch('/api/channels/search?q=');
        const channels = await response.json();
        
        allPlaylistChannels = channels.filter(ch => ch.active === true || ch.active === 1);
        
        loadingStatus.textContent = `✅ Loaded ${allPlaylistChannels.length} channels`;
        setTimeout(() => {
            loadingStatus.style.display = 'none';
        }, 2000);
        
        const categories = [...new Set(allPlaylistChannels.map(ch => ch.group))].filter(c => c);
        const categorySelect = document.getElementById('playlistCategoryFilter');
        categorySelect.innerHTML = '<option value="">Semua Kategori (' + allPlaylistChannels.length + ')</option>';
        categories.forEach(cat => {
            const count = allPlaylistChannels.filter(ch => ch.group === cat).length;
            const option = document.createElement('option');
            option.value = cat;
            option.textContent = `${cat} (${count})`;
            categorySelect.appendChild(option);
        });
        
        document.getElementById('totalChannelCount').textContent = allPlaylistChannels.length;
        renderChannelSelection();
        
    } catch (error) {
        console.error('Error loading channels:', error);
        loadingStatus.textContent = '❌ Error loading channels';
        loadingStatus.style.background = '#ffebee';
    }
}

function filterPlaylistChannels() {
    renderChannelSelection();
}

function renderChannelSelection() {
    const categoryFilter = document.getElementById('playlistCategoryFilter').value;
    const searchQuery = document.getElementById('playlistChannelSearch').value.toLowerCase();
    const container = document.getElementById('channelSelectionList');
    
    let filteredChannels = allPlaylistChannels;
    
    if (categoryFilter) {
        filteredChannels = filteredChannels.filter(ch => ch.group === categoryFilter);
    }
    
    if (searchQuery) {
        filteredChannels = filteredChannels.filter(ch => 
            ch.name.toLowerCase().includes(searchQuery)
        );
    }
    
    if (filteredChannels.length === 0) {
        container.innerHTML = '<p style="text-align: center; color: #999;">Tidak ada channel yang cocok</p>';
        return;
    }
    
    const displayLimit = 200;
    const displayChannels = filteredChannels.slice(0, displayLimit);
    const hasMore = filteredChannels.length > displayLimit;
    
    let html = '<div style="display: grid; gap: 8px;">';
    
    if (hasMore) {
        html += `<div style="background: #fff3cd; padding: 10px; border-radius: 5px; margin-bottom: 10px;">
            ℹ️ Menampilkan ${displayLimit} dari ${filteredChannels.length} channel. Gunakan pencarian untuk hasil lebih spesifik.
        </div>`;
    }
    
    const grouped = {};
    displayChannels.forEach(ch => {
        const cat = ch.group || 'Uncategorized';
        if (!grouped[cat]) grouped[cat] = [];
        grouped[cat].push(ch);
    });
    
    Object.keys(grouped).sort().forEach(category => {
        const categoryChannels = grouped[category];
        const selectedInCategory = categoryChannels.filter(ch => selectedChannels.has(ch.id)).length;
        
        html += `<div style="margin-bottom: 15px;">`;
        html += `<div style="background: #667eea; color: white; padding: 8px; border-radius: 5px; font-weight: bold; margin-bottom: 5px; display: flex; justify-content: space-between;">
            <span>${category}</span>
            <span style="font-size: 0.85em; background: rgba(255,255,255,0.2); padding: 3px 8px; border-radius: 3px;">
                ${selectedInCategory}/${categoryChannels.length}
            </span>
        </div>`;
        
        categoryChannels.forEach(channel => {
            const checked = selectedChannels.has(channel.id) ? 'checked' : '';
            html += `
                <label style="display: flex; align-items: center; padding: 8px; background: ${checked ? '#e3f2fd' : '#f9f9f9'}; margin: 3px 0; border-radius: 5px; cursor: pointer;">
                    <input type="checkbox" ${checked} onchange="togglePlaylistChannel(${channel.id})" style="margin-right: 10px;">
                    <span>${channel.name}</span>
                </label>
            `;
        });
        
        html += '</div>';
    });
    
    html += '</div>';
    container.innerHTML = html;
    
    updateSelectedCount();
    const hasSelection = selectedChannels.size > 0;
    document.getElementById('generateBtn').disabled = !hasSelection;
    document.getElementById('copyUrlBtn').disabled = !hasSelection;
}

function togglePlaylistChannel(channelId) {
    if (selectedChannels.has(channelId)) {
        selectedChannels.delete(channelId);
    } else {
        selectedChannels.add(channelId);
    }
    renderChannelSelection();
}

function selectAllVisible() {
    const categoryFilter = document.getElementById('playlistCategoryFilter').value;
    const searchQuery = document.getElementById('playlistChannelSearch').value.toLowerCase();
    
    let channelsToSelect = allPlaylistChannels;
    if (categoryFilter) {
        channelsToSelect = channelsToSelect.filter(ch => ch.category === categoryFilter);
    }
    if (searchQuery) {
        channelsToSelect = channelsToSelect.filter(ch => ch.name.toLowerCase().includes(searchQuery));
    }
    
    channelsToSelect.forEach(ch => selectedChannels.add(ch.id));
    renderChannelSelection();
}

function deselectAllVisible() {
    selectedChannels.clear();
    renderChannelSelection();
}

function selectByCategory() {
    const categoryFilter = document.getElementById('playlistCategoryFilter').value;
    if (!categoryFilter) {
        alert('Pilih kategori terlebih dahulu dari dropdown filter!');
        return;
    }
    
    const channelsInCategory = allPlaylistChannels.filter(ch => ch.group === categoryFilter);
    channelsInCategory.forEach(ch => selectedChannels.add(ch.id));
    renderChannelSelection();
}

function updateSelectedCount() {
    document.getElementById('selectedChannelCount').textContent = selectedChannels.size;
}

async function generatePlaylist() {
    const select = document.getElementById('playlistUser');
    const userId = select.value;
    const username = select.options[select.selectedIndex].dataset.username;
    const filename = document.getElementById('playlistFileName').value || `playlist-${username}.m3u`;
    
    if (!userId) {
        alert('Pilih user terlebih dahulu!');
        return;
    }
    
    if (selectedChannels.size === 0) {
        alert('Pilih minimal 1 channel!');
        return;
    }
    
    // Ask for user password for authentication
    const password = prompt(`Masukkan password untuk user "${username}":\n\n(Password ini akan dimasukkan ke dalam URL stream untuk autentikasi)`);
    if (!password) {
        alert('Password diperlukan untuk generate playlist!');
        return;
    }
    
    try {
        const selectedChannelsList = allPlaylistChannels.filter(ch => selectedChannels.has(ch.id));
        
        let m3uContent = '#EXTM3U\n';
        m3uContent += `# IPTV Playlist for: ${username}\n`;
        m3uContent += `# Generated: ${new Date().toLocaleString('id-ID')}\n`;
        m3uContent += `# Total Channels: ${selectedChannelsList.length}\n\n`;
        
        selectedChannelsList.forEach(channel => {
            m3uContent += `#EXTINF:-1 tvg-id="${channel.id}" tvg-name="${channel.name}" group-title="${channel.group || 'Uncategorized'}",${channel.name}\n`;
            m3uContent += `http://${window.location.host}/api/proxy/channel/${channel.id}?username=${username}&password=${password}\n`;
        });
        
        const blob = new Blob([m3uContent], { type: 'text/plain' });
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
        
        alert(`✅ Playlist generated!\n\nFile: ${filename}\nChannels: ${selectedChannelsList.length}\n\nFile sudah didownload.`);
        
    } catch (error) {
        alert('Error generating playlist: ' + error.message);
    }
}

async function saveAndCopyPlaylistURL() {
    const select = document.getElementById('playlistUser');
    const userId = select.value;
    const username = select.options[select.selectedIndex].dataset.username;
    const filename = document.getElementById('playlistFileName').value || `playlist-${username}.m3u`;
    
    if (!userId) {
        alert('Pilih user terlebih dahulu!');
        return;
    }
    
    if (selectedChannels.size === 0) {
        alert('Pilih minimal 1 channel!');
        return;
    }
    
    // Ask for user password for authentication
    const password = prompt(`Masukkan password untuk user "${username}":\n\n(Password ini akan dimasukkan ke dalam URL stream untuk autentikasi)`);
    if (!password) {
        alert('Password diperlukan untuk save playlist!');
        return;
    }
    
    try {
        const selectedChannelsList = allPlaylistChannels.filter(ch => selectedChannels.has(ch.id));
        
        let m3uContent = '#EXTM3U\n';
        m3uContent += `# IPTV Playlist for: ${username}\n`;
        m3uContent += `# Generated: ${new Date().toLocaleString('id-ID')}\n`;
        m3uContent += `# Total Channels: ${selectedChannelsList.length}\n\n`;
        
        selectedChannelsList.forEach(channel => {
            m3uContent += `#EXTINF:-1 tvg-id="${channel.id}" tvg-name="${channel.name}" group-title="${channel.group || 'Uncategorized'}",${channel.name}\n`;
            m3uContent += `http://${window.location.host}/api/proxy/channel/${channel.id}?username=${username}&password=${password}\n`;
        });
        
        // Save to server
        const response = await fetch('/api/generated-playlists', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                filename: filename,
                content: m3uContent
            })
        });
        
        const result = await response.json();
        
        if (result.success) {
            const fullUrl = `http://${window.location.host}${result.url}`;
            document.getElementById('generatedPlaylistUrl').value = fullUrl;
            document.getElementById('playlistUrlBox').style.display = 'block';
            
            alert(`✅ Playlist saved!\n\nFile: ${filename}\nChannels: ${selectedChannelsList.length}\n\nURL sudah tersedia di bawah.`);
        } else {
            alert('Error saving playlist');
        }
        
    } catch (error) {
        alert('Error saving playlist: ' + error.message);
    }
}

function copyToClipboard() {
    const urlInput = document.getElementById('generatedPlaylistUrl');
    urlInput.select();
    urlInput.setSelectionRange(0, 99999); // For mobile devices
    
    navigator.clipboard.writeText(urlInput.value).then(() => {
        alert('✅ URL copied to clipboard!');
    }).catch(err => {
        // Fallback for older browsers
        document.execCommand('copy');
        alert('✅ URL copied to clipboard!');
    });
}
