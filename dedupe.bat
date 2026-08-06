@echo off
echo Building deduplication script...
cd dedupe
go build main.go
if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b %errorlevel%
)
echo Running deduplication...
main.exe
echo.
pause
