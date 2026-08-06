@echo off
echo Starting DHT Search Engine...
cd /d "%~dp0\dhtsearch"

echo Starting backend server on port 8080...
start "DHT Backend" cmd /c "go run main.go"

echo Waiting for server to start...
timeout /t 2 /nobreak >nul

echo Opening browser to http://localhost:8080...
start http://localhost:8080

echo Done!
