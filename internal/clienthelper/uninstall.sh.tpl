#!/bin/sh

VALHEIM_DIR=$(dirname "$0")

DIRS_TO_REMOVE=(
    "$VALHEIM_DIR/BepInEx/"
    "$VALHEIM_DIR/doorstop_libs/"
)

echo "Removing Sindri files..."

for file in "${DIRS_TO_REMOVE[@]}"; do
    rm -rf "$file" || true
done

FILES_TO_REMOVE=(
    "$VALHEIM_DIR/changelog.txt"
    "$VALHEIM_DIR/doorstop_config.ini"
    "$VALHEIM_DIR/start_game_bepinex.sh"
    "$VALHEIM_DIR/start_server_bepinex.sh"
    "$VALHEIM_DIR/winhttp.dll"
    "$VALHEIM_DIR/__README_TXT_NAME__"
    "$VALHEIM_DIR/__UNINSTALL_CMD_NAME__"
    "$VALHEIM_DIR/__UPDATE_CMD_NAME__"
    "$VALHEIM_DIR/__UNINSTALL_SH_NAME__"
    "$VALHEIM_DIR/__UPDATE_SH_NAME__"
)

for file in "${FILES_TO_REMOVE[@]}"; do
    rm -f "$file" || true
done

echo "Sindri uninstalled successfully."
