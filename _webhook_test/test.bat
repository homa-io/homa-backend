@echo off
REM Webhook Test Script for Windows
REM This script automates the full webhook testing process

echo 🧪 Homa Webhook Test Script
echo ============================
echo.

REM Check if test server is running
curl -s http://localhost:9000 >nul 2>&1
if %errorlevel% neq 0 (
    echo ⚠️  Test server not running on port 9000
    echo Starting test server in background...
    cd _webhook_test
    start /B go run server.go > server.log 2>&1
    cd ..
    timeout /t 2 >nul
    echo ✅ Test server started
) else (
    echo ✅ Test server already running
)

echo.
echo 📡 Creating test webhook...

REM Create test webhook using mock generator
go run main.go --generate-webhook --url http://localhost:9000/webhook --send

echo.
echo 🎫 Creating test ticket to trigger webhook...

REM Create a test ticket
curl -s -X POST http://localhost:8000/api/client/tickets -H "Content-Type: application/json" -d "{\"title\":\"Webhook Test Ticket\",\"client_name\":\"Test User\",\"client_email\":\"test@webhook.local\",\"status\":\"new\",\"priority\":\"medium\",\"message\":\"This ticket was created by the automated test script\"}"

echo.
echo ✅ Test ticket created!
echo.
echo 📊 Checking webhook logs...
timeout /t 1 >nul

REM List log files
if exist "_webhook_test\logs\" (
    echo ✅ Webhook logs directory exists
    echo.
    echo 📄 Recent webhook logs:
    dir /B /O-D "_webhook_test\logs\*.json" 2>nul

    echo.
    echo 📋 Latest webhook content:
    for /f "delims=" %%i in ('dir /B /O-D "_webhook_test\logs\*.json" 2^>nul ^| findstr /n "^" ^| findstr "^1:"') do (
        set "latest=%%i"
    )
    if defined latest (
        set "latest=%latest:*:=%"
        type "_webhook_test\logs\%latest%"
    )
) else (
    echo ⚠️  Logs directory not found
)

echo.
echo ================================
echo ✅ Test Complete!
echo.
echo 📁 Check logs: _webhook_test\logs\
echo 🖥️  View server console: type _webhook_test\server.log
echo 🌐 Open browser: http://localhost:9000
echo.
