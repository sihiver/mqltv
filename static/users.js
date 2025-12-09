// User Management Functions

async function loadUsers() {
    try {
        const response = await fetch('/api/users');
        const users = await response.json();
        
        const container = document.getElementById('usersList');
        if (users.length === 0) {
            container.innerHTML = '<p style="text-align: center; color: #999; padding: 40px;">Belum ada user.</p>';
            return;
        }
        
        let html = '<table><thead><tr>';
        html += '<th>Username</th><th>Nama</th><th>Email</th><th>Max Koneksi</th><th>Status</th><th>Sisa Hari</th><th>Aksi</th>';
        html += '</tr></thead><tbody>';
        
        users.forEach(user => {
            const expiresAt = user.expires_at ? new Date(user.expires_at).toLocaleDateString('id-ID') : 'Unlimited';
            const daysRemaining = user.days_remaining || 0;
            
            let statusBadge = '';
            if (!user.is_active) {
                statusBadge = '<span class="badge badge-danger">Nonaktif</span>';
            } else if (user.is_expired) {
                statusBadge = '<span class="badge badge-secondary">Expired</span>';
            } else if (daysRemaining <= 7 && daysRemaining > 0) {
                statusBadge = '<span class="badge" style="background: #ffc107; color: #000;">⚠️ ' + daysRemaining + ' hari</span>';
            } else {
                statusBadge = '<span class="badge badge-success">✅ Aktif</span>';
            }
            
            const daysDisplay = expiresAt === 'Unlimited' ? '∞' : (user.is_expired ? '<span style="color: #dc3545;">Expired</span>' : daysRemaining + ' hari');
            
            html += `
                <tr>
                    <td><strong>${user.username}</strong></td>
                    <td>${user.full_name || '-'}</td>
                    <td>${user.email || '-'}</td>
                    <td style="text-align: center;">${user.max_connections}</td>
                    <td style="text-align: center;">${statusBadge}</td>
                    <td style="text-align: center;">${daysDisplay}</td>
                    <td style="text-align: center;">
                        <button class="btn btn-sm" onclick="viewUserChannels('${user.username}')">📺 Channels</button>
                        <button class="btn btn-success btn-sm" onclick="extendSubscription(${user.id}, '${user.username}')">➕ Perpanjang</button>
                        <button class="btn btn-sm" onclick="editUser(${user.id})">Edit</button>
                        <button class="btn btn-sm" onclick="resetPassword(${user.id})">Reset Pass</button>
                        <button class="btn btn-warning btn-sm" onclick="setExpired(${user.id}, '${user.username}')">⏰ Test Expired</button>
                        <button class="btn btn-danger btn-sm" onclick="deleteUser(${user.id})">Hapus</button>
                    </td>
                </tr>
            `;
        });
        
        html += '</tbody></table>';
        container.innerHTML = html;
    } catch (error) {
        console.error('Error loading users:', error);
    }
}

function showCreateUser() {
    document.getElementById('createUserForm').style.display = 'block';
}

function hideCreateUser() {
    document.getElementById('createUserForm').style.display = 'none';
    document.getElementById('newUsername').value = '';
    document.getElementById('newPassword').value = '';
    document.getElementById('newFullName').value = '';
    document.getElementById('newEmail').value = '';
    document.getElementById('newMaxConnections').value = '1';
    document.getElementById('newDurationDays').value = '';
    document.getElementById('newNotes').value = '';
}

async function createUser() {
    const username = document.getElementById('newUsername').value;
    const password = document.getElementById('newPassword').value;
    const full_name = document.getElementById('newFullName').value;
    const email = document.getElementById('newEmail').value;
    const max_connections = parseInt(document.getElementById('newMaxConnections').value);
    const duration_days = parseInt(document.getElementById('newDurationDays').value) || 0;
    const notes = document.getElementById('newNotes').value;
    
    if (!username || !password) {
        alert('Username dan password harus diisi!');
        return;
    }
    
    try {
        const response = await fetch('/api/users', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password, full_name, email, max_connections, duration_days, notes })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            alert('User berhasil dibuat!');
            hideCreateUser();
            loadUsers();
        } else {
            alert('Error: ' + (result.error || 'Failed to create user'));
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function deleteUser(userId) {
    if (!confirm('Hapus user ini?')) return;
    
    try {
        const response = await fetch(`/api/users/${userId}`, { method: 'DELETE' });
        if (response.ok) {
            alert('User berhasil dihapus!');
            loadUsers();
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function editUser(userId) {
    const newName = prompt('Nama lengkap baru:');
    if (!newName) return;
    
    const extendDays = prompt('Extend subscription (hari):');
    const days = parseInt(extendDays) || 0;
    
    try {
        const response = await fetch(`/api/users/${userId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ full_name: newName, extend_days: days, is_active: true, max_connections: 1 })
        });
        
        if (response.ok) {
            alert('User berhasil diupdate!');
            loadUsers();
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function resetPassword(userId) {
    const newPassword = prompt('Password baru:');
    if (!newPassword) return;
    
    try {
        const response = await fetch(`/api/users/${userId}/reset-password`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ new_password: newPassword })
        });
        
        if (response.ok) {
            alert('Password berhasil direset!');
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function viewUserChannels(username) {
    try {
        // Fetch the playlist file from playlists folder
        const response = await fetch(`/playlists/playlist-${username}.m3u`);
        
        if (!response.ok) {
            alert(`User "${username}" belum memiliki generated playlist.\n\nSilakan generate playlist terlebih dahulu di tab "Generate Playlist".`);
            return;
        }
        
        const m3uContent = await response.text();
        
        // Parse M3U content
        const lines = m3uContent.split('\n');
        const channels = [];
        let currentChannel = {};
        
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i].trim();
            
            if (line.startsWith('#EXTINF:')) {
                // Parse channel info
                const nameMatch = line.match(/,(.+)$/);
                const groupMatch = line.match(/group-title="([^"]+)"/);
                
                if (nameMatch) {
                    currentChannel = {
                        name: nameMatch[1],
                        group: groupMatch ? groupMatch[1] : 'Uncategorized'
                    };
                }
            } else if (line && !line.startsWith('#') && currentChannel.name) {
                currentChannel.url = line;
                channels.push(currentChannel);
                currentChannel = {};
            }
        }
        
        // Display channels in a modal/alert
        if (channels.length === 0) {
            alert('Tidak ada channel ditemukan dalam playlist.');
            return;
        }
        
        // Group by category
        const grouped = {};
        channels.forEach(ch => {
            if (!grouped[ch.group]) {
                grouped[ch.group] = [];
            }
            grouped[ch.group].push(ch.name);
        });
        
        // Create display text
        let displayText = `📺 Channels untuk user: ${username}\n`;
        displayText += `Total: ${channels.length} channels\n\n`;
        
        Object.keys(grouped).sort().forEach(group => {
            displayText += `\n【${group}】(${grouped[group].length})\n`;
            grouped[group].forEach((name, idx) => {
                displayText += `  ${idx + 1}. ${name}\n`;
            });
        });
        
        // Show in alert (for simple display) or create a modal
        alert(displayText);
        
    } catch (error) {
        console.error('Error:', error);
        alert('Error loading channels: ' + error.message);
    }
}

async function extendSubscription(userId, username) {
    const daysInput = prompt(`Perpanjang Subscription untuk "${username}":\n\nMasukkan jumlah hari perpanjangan:\n- 7 = 1 minggu\n- 30 = 1 bulan\n- 90 = 3 bulan\n- 365 = 1 tahun`, '30');
    
    if (daysInput === null) return;
    
    const days = parseInt(daysInput);
    if (isNaN(days) || days <= 0) {
        alert('Input harus berupa angka positif (lebih dari 0)!');
        return;
    }
    
    try {
        const response = await fetch(`/api/users/${userId}/extend`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ days: days })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            const expiresAt = new Date(result.expires_at);
            const daysRemaining = result.days_remaining || 0;
            alert(`✅ Subscription Berhasil Diperpanjang!\n\nUser: ${username}\nDiperpanjang: ${days} hari\nExpires: ${expiresAt.toLocaleString('id-ID')}\nSisa: ${daysRemaining} hari`);
            loadUsers();
        } else {
            alert('Error: ' + (result.error || 'Failed to extend subscription'));
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

async function setExpired(userId, username) {
    const daysInput = prompt(`[TEST] Set expiration untuk user "${username}":\n\nMasukkan jumlah hari:\n- Angka positif = extend (contoh: 30 untuk 30 hari lagi)\n- Angka negatif = expired (contoh: -1 untuk sudah expired)\n- 0 = expired hari ini`, '-1');
    
    if (daysInput === null) return;
    
    const days = parseInt(daysInput);
    if (isNaN(days)) {
        alert('Input harus berupa angka!');
        return;
    }
    
    try {
        const response = await fetch(`/api/users/${userId}/set-expired`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ days: days })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            const expiresAt = new Date(result.expires_at);
            const status = days < 0 ? '❌ EXPIRED' : (days === 0 ? '⚠️ Expired Hari Ini' : '✅ Extended');
            alert(`${status}\n\nUser: ${username}\nExpires: ${expiresAt.toLocaleString('id-ID')}\n\nUser berhasil diupdate!`);
            loadUsers();
        } else {
            alert('Error: ' + (result.error || 'Failed to update expiration'));
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}
