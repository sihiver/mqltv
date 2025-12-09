// Bandwidth Monitoring Functions

function startBandwidthMonitoring() {
    if (bandwidthMonitorInterval) {
        clearInterval(bandwidthMonitorInterval);
    }
    updateBandwidthStats();
    bandwidthMonitorInterval = setInterval(updateBandwidthStats, 3000);
}

async function updateBandwidthStats() {
    try {
        const response = await fetch('/api/streams/status');
        const data = await response.json();
        
        let totalDownloadMbps = 0;
        let totalUploadMbps = 0;
        let totalDownloadMB = 0;
        let totalUploadMB = 0;
        
        if (data.streams && data.streams.length > 0) {
            data.streams.forEach(stream => {
                totalDownloadMbps += stream.download_mbps || 0;
                totalUploadMbps += stream.upload_mbps || 0;
                totalDownloadMB += (stream.bytes_read || 0) / 1024 / 1024;
                totalUploadMB += (stream.bytes_written || 0) / 1024 / 1024;
            });
        }
        
        document.getElementById('downloadMbps').textContent = totalDownloadMbps.toFixed(2);
        document.getElementById('uploadMbps').textContent = totalUploadMbps.toFixed(2);
        document.getElementById('totalDownloadMB').textContent = totalDownloadMB.toFixed(2);
        document.getElementById('totalUploadMB').textContent = totalUploadMB.toFixed(2);
        
    } catch (error) {
        console.error('Error updating bandwidth:', error);
    }
}
