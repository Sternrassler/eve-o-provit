#!/usr/bin/env bash
# Download EVE SDE SQLite Database from Official Release
# Quelle: https://github.com/Sternrassler/eve-sde/releases

set -euo pipefail

# Konfiguration
GITHUB_REPO="Sternrassler/eve-sde"
TARGET_DIR="backend/data/sde"
TARGET_FILE="sde.sqlite"
TEMP_DIR=$(mktemp -d)

# Farben für Output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

echo -e "${BLUE}[download-sde]${NC} Lade EVE SDE SQLite Datenbank..."

# Schritt 1: Hole neueste Release Info von GitHub
echo -e "${BLUE}[download-sde]${NC} Prüfe neueste Release Version..."

if ! command -v curl &> /dev/null; then
    echo -e "${RED}[download-sde]${NC} ❌ curl nicht gefunden - bitte installieren"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}[download-sde]${NC} ⚠️  jq nicht gefunden - installiere..."
    if command -v apt-get &> /dev/null; then
        sudo apt-get update -qq && sudo apt-get install -y jq
    else
        echo -e "${RED}[download-sde]${NC} ❌ jq Installation fehlgeschlagen"
        exit 1
    fi
fi

# GitHub API: Latest Release
RELEASE_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
RELEASE_JSON=$(curl -sL "$RELEASE_URL")

# Extrahiere Tag Name und Asset URL
TAG_NAME=$(echo "$RELEASE_JSON" | jq -r '.tag_name')
DOWNLOAD_URL=$(echo "$RELEASE_JSON" | jq -r '.assets[] | select(.name | test("\\.(db|sqlite|db\\.gz)$")) | .browser_download_url' | head -1)

if [ -z "$DOWNLOAD_URL" ] || [ "$DOWNLOAD_URL" = "null" ]; then
    echo -e "${RED}[download-sde]${NC} ❌ Keine SQLite DB in Release gefunden"
    echo -e "${YELLOW}[download-sde]${NC} Release Info:"
    echo "$RELEASE_JSON" | jq '.assets[] | {name: .name, url: .browser_download_url}'
    exit 1
fi

# Prüfen ob komprimiert
IS_COMPRESSED=false
ASSET_NAME=$(echo "$RELEASE_JSON" | jq -r '.assets[] | select(.browser_download_url == "'$DOWNLOAD_URL'") | .name')
if [[ "$ASSET_NAME" == *.gz ]]; then
    IS_COMPRESSED=true
    echo -e "${YELLOW}[download-sde]${NC} Komprimierte Datei erkannt: ${ASSET_NAME}"
fi

echo -e "${GREEN}[download-sde]${NC} ✅ Neueste Version: ${TAG_NAME}"
echo -e "${BLUE}[download-sde]${NC} Download URL: ${DOWNLOAD_URL}"

# Schritt 2: Erstelle Ziel-Verzeichnis
mkdir -p "$TARGET_DIR"

# Schritt 3: Prüfe ob Datei bereits existiert
if [ -f "$TARGET_DIR/$TARGET_FILE" ]; then
    echo -e "${YELLOW}[download-sde]${NC} Datei existiert bereits: $TARGET_DIR/$TARGET_FILE"
    read -p "Überschreiben? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}[download-sde]${NC} Download abgebrochen"
        exit 0
    fi
    rm -f "$TARGET_DIR/$TARGET_FILE"
fi

# Schritt 4: Download
echo -e "${BLUE}[download-sde]${NC} Lade Datenbank herunter..."
if [ "$IS_COMPRESSED" = true ]; then
    TEMP_FILE="$TEMP_DIR/eve-sde-download.gz"
else
    TEMP_FILE="$TEMP_DIR/eve-sde-download"
fi

if ! curl -L --progress-bar "$DOWNLOAD_URL" -o "$TEMP_FILE"; then
    echo -e "${RED}[download-sde]${NC} ❌ Download fehlgeschlagen"
    exit 1
fi

# Schritt 4.1: Dekomprimierung (falls .gz)
if [ "$IS_COMPRESSED" = true ]; then
    echo -e "${BLUE}[download-sde]${NC} Dekomprimiere Datei..."
    if ! gunzip -f "$TEMP_FILE"; then
        echo -e "${RED}[download-sde]${NC} ❌ Dekomprimierung fehlgeschlagen"
        exit 1
    fi
    # gunzip entfernt .gz Endung automatisch
    TEMP_FILE="${TEMP_FILE%.gz}"
fi

# Schritt 5: Validierung (einfacher Check: Dateigröße)
FILE_SIZE=$(stat -f%z "$TEMP_FILE" 2>/dev/null || stat -c%s "$TEMP_FILE" 2>/dev/null)
MIN_SIZE=$((100 * 1024 * 1024))  # Mindestens 100 MB

if [ "$FILE_SIZE" -lt "$MIN_SIZE" ]; then
    echo -e "${RED}[download-sde]${NC} ❌ Datei zu klein (${FILE_SIZE} bytes) - möglicherweise fehlerhafter Download"
    exit 1
fi

echo -e "${GREEN}[download-sde]${NC} ✅ Download erfolgreich ($(numfmt --to=iec-i --suffix=B "$FILE_SIZE" 2>/dev/null || echo "${FILE_SIZE} bytes"))"

# Schritt 6: Verschiebe nach Ziel
mv "$TEMP_FILE" "$TARGET_DIR/$TARGET_FILE"

# Schritt 7: Setze Read-Only Permissions
chmod 444 "$TARGET_DIR/$TARGET_FILE"

echo -e "${GREEN}[download-sde]${NC} ✅ SDE Datenbank installiert:"
echo -e "${BLUE}[download-sde]${NC}   Pfad: $TARGET_DIR/$TARGET_FILE"
echo -e "${BLUE}[download-sde]${NC}   Version: $TAG_NAME"
echo -e "${BLUE}[download-sde]${NC}   Größe: $(ls -lh "$TARGET_DIR/$TARGET_FILE" | awk '{print $5}')"
