@echo off
cd /d "%~dp0"
if "%~1"=="" (
    echo Usage: import_db.bat ^<path_to_db_or_directory^>
    echo Using default directory: .db
    python migrate.py .db
) else (
    python migrate.py "%~1"
)
pause
