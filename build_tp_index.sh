#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

echo "Starting Export from Elasticsearch..."
cd "$DIR/es_export"
./es_export.exe 2>&1 | tee "$DIR/log.txt"

echo "Export successful! Now generating Torrent Paradise IPFS Index..."
cd "$DIR/../torrent-paradise/index-generator"
node --max-old-space-size=8192 main.js 2>&1 | tee -a "$DIR/../log.txt"

echo "Running metadata fix..."
# Assuming python means python3 or python in the path
python fix-metajson.py ../website/generated/inx 2>&1 | tee -a "$DIR/../log.txt"

echo "Done! The Torrent Paradise frontend is now updated with data from Elasticsearch."
echo "The files are ready in:"
echo "$DIR/../torrent-paradise/website"
