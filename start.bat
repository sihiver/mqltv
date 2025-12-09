@echo off

REM Check if Go is installed
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Go is not installed. Please install Go from https://golang.org/dl/
    exit /b 1
)

REM Download dependencies
echo Downloading dependencies...
go mod download

REM Build application
echo Building application...
go build -o iptv-panel.exe main.go

REM Run application
echo Starting IPTV Panel...
iptv-panel.exe
