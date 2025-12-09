# IPTV Panel - Systemd Service

File service systemd untuk menjalankan IPTV Panel sebagai background service.

## Files

1. **iptv-panel.service** - Systemd service unit file
2. **service.sh** - Script helper untuk manage service

## Quick Start

### Install Service
```bash
./service.sh install
```

### Start Service
```bash
sudo systemctl start iptv-panel
```

### Check Status
```bash
sudo systemctl status iptv-panel
```

### View Logs
```bash
# Real-time logs
sudo journalctl -u iptv-panel -f

# Or view log file
tail -f server.log
```

## Commands

### Menggunakan service.sh
```bash
./service.sh install    # Install service
./service.sh start      # Start service
./service.sh stop       # Stop service
./service.sh restart    # Restart service
./service.sh status     # Show status
./service.sh logs       # Show live logs
./service.sh enable     # Enable on boot
./service.sh disable    # Disable on boot
./service.sh uninstall  # Remove service
```

### Menggunakan systemctl langsung
```bash
# Start service
sudo systemctl start iptv-panel

# Stop service
sudo systemctl stop iptv-panel

# Restart service
sudo systemctl restart iptv-panel

# Check status
sudo systemctl status iptv-panel

# Enable on boot
sudo systemctl enable iptv-panel

# Disable on boot
sudo systemctl disable iptv-panel

# View logs
sudo journalctl -u iptv-panel -f
```

## Service Configuration

File: `iptv-panel.service`

```ini
[Unit]
Description=IPTV Panel Service
After=network.target

[Service]
Type=simple
User=dindin
WorkingDirectory=/home/dindin/mqltv
ExecStart=/home/dindin/mqltv/iptv-panel
Restart=always
RestartSec=5
StandardOutput=append:/home/dindin/mqltv/server.log
StandardError=append:/home/dindin/mqltv/server.log

[Install]
WantedBy=multi-user.target
```

## Features

- ✅ Auto-restart jika service crash
- ✅ Log output ke file `server.log`
- ✅ Start otomatis saat boot (jika dienable)
- ✅ Isolasi security dengan NoNewPrivileges dan PrivateTmp
- ✅ Graceful restart tanpa downtime

## Troubleshooting

### Service tidak start
```bash
# Check status
sudo systemctl status iptv-panel

# Check logs
sudo journalctl -u iptv-panel -n 50

# Check file permissions
ls -la /home/dindin/mqltv/iptv-panel
```

### Port sudah digunakan
```bash
# Stop manual process
pkill -f iptv-panel

# Start service
sudo systemctl start iptv-panel
```

### Update setelah rebuild
```bash
# Stop service
sudo systemctl stop iptv-panel

# Rebuild
go build -o iptv-panel

# Start service
sudo systemctl start iptv-panel
```

## Log Management

Logs disimpan di 2 tempat:

1. **File log**: `/home/dindin/mqltv/server.log`
   ```bash
   tail -f server.log
   ```

2. **Systemd journal**:
   ```bash
   sudo journalctl -u iptv-panel -f
   ```

### Rotate logs
Jika log file terlalu besar, gunakan logrotate:

```bash
sudo nano /etc/logrotate.d/iptv-panel
```

Isi dengan:
```
/home/dindin/mqltv/server.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
    copytruncate
}
```

## Uninstall

```bash
./service.sh uninstall
```

Atau manual:
```bash
sudo systemctl stop iptv-panel
sudo systemctl disable iptv-panel
sudo rm /etc/systemd/system/iptv-panel.service
sudo systemctl daemon-reload
```
