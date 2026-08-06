@echo off
cd /d "%~dp0"

echo Starting MongoDB...
if not exist "data" mkdir data
start "MongoDB" "mongodb4\mongodb-win32-x86_64-windows-4.4.29\bin\mongod.exe" --dbpath data --setParameter textSearchEnabled=true

echo Waiting for MongoDB to start...
timeout /t 5 /nobreak >nul

echo Starting HTTP Daemon...
start "DHTCrawler2 HTTP" erl -pa ebin -s crawler_http_app

echo Both backend and frontend are starting in separate windows.
echo The frontend should be accessible at http://localhost:8000
pause
