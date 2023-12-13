#!/bin/sh

VALHEIM_DIR=$(dirname "$0")

FILES_TO_REMOVE=(
    "$VALHEIM_DIR/changelog.txt"
    "$VALHEIM_DIR/doorstop_config.ini"
    "$VALHEIM_DIR/start_game_bepinex.sh"
    "$VALHEIM_DIR/start_server_bepinex.sh"
    "$VALHEIM_DIR/winhttp.dll"
)

echo "Removing old Sindri files..."

for file in "${FILES_TO_REMOVE[@]}"; do
    rm -f "$file" || true
done

DIRS_TO_REMOVE=(
    "$VALHEIM_DIR/BepInEx/"
    "$VALHEIM_DIR/doorstop_libs/"
)

for file in "${DIRS_TO_REMOVE[@]}"; do
    rm -rf "$file" || true
done

PROTOCOL="__PROTOCOL__"
HOST="__HOST__"

echo "Downloading and extracting new Sindri files from $PROTOCOL://$HOST..."

if ! curl -fSs $PROTOCOL://$HOST/mods.gz | tar -C $VALHEIM_DIR -xzf -; then
    echo "Failed to download and extract new Sindir files."
    exit 1
fi

FILES_TO_CLEANUP=(
    "$VALHEIM_DIR/__UNINSTALL_CMD_NAME__"
    "$VALHEIM_DIR/__UNINSTALL_CMD_NAME__"
)

echo "Cleaning up unnecessary Sindri files..."

for file in "${FILES_TO_CLEANUP[@]}"; do
    rm -f "$file" || true
done

echo "Sindri updated successfully."
