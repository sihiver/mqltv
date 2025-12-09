// Global variables
let currentPlaylist = null;
let bandwidthMonitorInterval = null;
let selectedChannels = new Set();
let allPlaylistChannels = [];

// Initialize app when DOM and all scripts are loaded
window.addEventListener('DOMContentLoaded', function() {
    loadStats();
    loadPlaylists();
    startBandwidthMonitoring();
});

// Switch between tabs
function switchTab(tabName, event) {
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.remove('active');
    });
    document.querySelectorAll('.tab').forEach(tab => {
        tab.classList.remove('active');
    });
    
    document.getElementById(tabName).classList.add('active');
    
    if (event && event.target) {
        event.target.classList.add('active');
    } else {
        const buttons = document.querySelectorAll('.tab');
        buttons.forEach(btn => {
            if (btn.textContent.toLowerCase().includes(tabName.toLowerCase())) {
                btn.classList.add('active');
            }
        });
    }
    
    if (tabName === 'channels') {
        loadChannels();
    } else if (tabName === 'relays') {
        loadRelays();
    } else if (tabName === 'users') {
        loadUsers();
    } else if (tabName === 'generate-playlist') {
        loadPlaylistUsers();
    }
}

// Statistics functions
async function loadStats() {
    try {
        const response = await fetch('/api/stats');
        const stats = await response.json();
        
        document.getElementById('totalPlaylists').textContent = stats.total_playlists;
        document.getElementById('totalChannels').textContent = stats.total_channels;
        document.getElementById('activeChannels').textContent = stats.active_channels;
        document.getElementById('totalRelays').textContent = stats.total_relays;
    } catch (error) {
        console.error('Error loading stats:', error);
    }
}
