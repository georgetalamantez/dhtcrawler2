@echo off
echo Syncing .db files to Elasticsearch...
echo This will automatically scan your .db folder and import any new torrents.
echo Existing torrents will be updated/skipped automatically.
cd /d "%~dp0es_ingest"
go run main.go >> log-sync.txt 2>&1
echo.
echo Sync Complete!
pause
