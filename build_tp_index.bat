@echo off
echo Starting Export from Elasticsearch...
cd /d "%~dp0es_export"
go run main.go > "%~dp0log.txt" 2>&1
if %ERRORLEVEL% neq 0 (
    echo Elasticsearch export failed! Check log.txt
    exit /b %ERRORLEVEL%
)

echo.
echo Export successful! Now generating Torrent Paradise IPFS Index...
cd /d "%~dp0..\torrent-paradise\index-generator"
node --max-old-space-size=8192 main.js
if %ERRORLEVEL% neq 0 (
    echo Index generation failed! Check log.txt
    exit /b %ERRORLEVEL%
)

echo.
echo Running metadata fix...
python fix-metajson.py ../website/generated/inx >> "%~dp0..\log.txt" 2>&1

echo.
echo Done! The Torrent Paradise frontend is now updated with data from Elasticsearch.
echo The files are ready in:
echo %~dp0..\torrent-paradise\website
