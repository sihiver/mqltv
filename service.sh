#!/bin/bash

# Script untuk install dan manage systemd service untuk IPTV Panel

SERVICE_NAME="iptv-panel"
SERVICE_FILE="${SERVICE_NAME}.service"
SYSTEMD_PATH="/etc/systemd/system/${SERVICE_FILE}"

case "$1" in
    install)
        echo "📦 Installing IPTV Panel service..."
        
        # Copy service file to systemd directory
        sudo cp "$SERVICE_FILE" "$SYSTEMD_PATH"
        
        # Reload systemd daemon
        sudo systemctl daemon-reload
        
        # Enable service to start on boot
        sudo systemctl enable "$SERVICE_NAME"
        
        echo "✅ Service installed successfully!"
        echo ""
        echo "Commands available:"
        echo "  sudo systemctl start $SERVICE_NAME     - Start service"
        echo "  sudo systemctl stop $SERVICE_NAME      - Stop service"
        echo "  sudo systemctl restart $SERVICE_NAME   - Restart service"
        echo "  sudo systemctl status $SERVICE_NAME    - Check status"
        echo "  sudo journalctl -u $SERVICE_NAME -f    - View logs"
        ;;
        
    uninstall)
        echo "🗑️  Uninstalling IPTV Panel service..."
        
        # Stop and disable service
        sudo systemctl stop "$SERVICE_NAME"
        sudo systemctl disable "$SERVICE_NAME"
        
        # Remove service file
        sudo rm -f "$SYSTEMD_PATH"
        
        # Reload systemd daemon
        sudo systemctl daemon-reload
        
        echo "✅ Service uninstalled successfully!"
        ;;
        
    start)
        echo "▶️  Starting IPTV Panel service..."
        sudo systemctl start "$SERVICE_NAME"
        sudo systemctl status "$SERVICE_NAME" --no-pager
        ;;
        
    stop)
        echo "⏹️  Stopping IPTV Panel service..."
        sudo systemctl stop "$SERVICE_NAME"
        sudo systemctl status "$SERVICE_NAME" --no-pager
        ;;
        
    restart)
        echo "🔄 Restarting IPTV Panel service..."
        sudo systemctl restart "$SERVICE_NAME"
        sudo systemctl status "$SERVICE_NAME" --no-pager
        ;;
        
    status)
        sudo systemctl status "$SERVICE_NAME"
        ;;
        
    logs)
        echo "📋 Viewing logs (press Ctrl+C to exit)..."
        sudo journalctl -u "$SERVICE_NAME" -f
        ;;
        
    enable)
        echo "🔧 Enabling service to start on boot..."
        sudo systemctl enable "$SERVICE_NAME"
        echo "✅ Service enabled!"
        ;;
        
    disable)
        echo "🔧 Disabling service from starting on boot..."
        sudo systemctl disable "$SERVICE_NAME"
        echo "✅ Service disabled!"
        ;;
        
    *)
        echo "IPTV Panel Service Manager"
        echo ""
        echo "Usage: $0 {install|uninstall|start|stop|restart|status|logs|enable|disable}"
        echo ""
        echo "Commands:"
        echo "  install    - Install service to systemd"
        echo "  uninstall  - Remove service from systemd"
        echo "  start      - Start the service"
        echo "  stop       - Stop the service"
        echo "  restart    - Restart the service"
        echo "  status     - Show service status"
        echo "  logs       - Show live logs"
        echo "  enable     - Enable service on boot"
        echo "  disable    - Disable service on boot"
        exit 1
        ;;
esac
