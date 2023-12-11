@echo off
setlocal

set "VALHEIM_DIR=%~dp0"

set "FILES_TO_REMOVE="
set "FILES_TO_REMOVE=%FILES_TO_REMOVE% %VALHEIM_DIR%changelog.txt"
set "FILES_TO_REMOVE=%FILES_TO_REMOVE% %VALHEIM_DIR%doorstop_config.ini"
set "FILES_TO_REMOVE=%FILES_TO_REMOVE% %VALHEIM_DIR%start_game_bepinex.sh"
set "FILES_TO_REMOVE=%FILES_TO_REMOVE% %VALHEIM_DIR%start_server_bepinex.sh"
set "FILES_TO_REMOVE=%FILES_TO_REMOVE% %VALHEIM_DIR%winhttp.dll"

echo Removing old Sindri files...

for %%I in (%FILES_TO_REMOVE%) do (
    del "%%I" 2>nul
)

set "DIRS_TO_REMOVE="
set "DIRS_TO_REMOVE=%DIRS_TO_REMOVE% %VALHEIM_DIR%BepInEx\"
set "DIRS_TO_REMOVE=%DIRS_TO_REMOVE% %VALHEIM_DIR%doorstop_libs\"

for %%I in (%DIRS_TO_REMOVE%) do (
    rd /s /q "%%I" 2>nul
)

set "PROTOCOL=__PROTOCOL__"
set "HOST=__HOST__"

echo Downloading and extracting new Sindri files from %PROTOCOL%://%HOST%...

curl -fSs "%PROTOCOL%://%HOST%/mods.gz" | tar -C %VALHEIM_DIR% -xzf -
if %errorlevel% neq 0 (
    echo Failed to download new Sindir files.
    exit /b 1
)

set "FILES_TO_CLEANUP="
set "FILES_TO_CLEANUP=%FILES_TO_CLEANUP% %VALHEIM_DIR%__UNINSTALL_SINDRI_SH_NAME__"
set "FILES_TO_CLEANUP=%FILES_TO_CLEANUP% %VALHEIM_DIR%__UPDATE_SINDRI_SH_NAME__"

echo Cleaning up unnecessary Sindri files...

for %%I in (%FILES_TO_CLEANUP%) do (
    del "%%I" 2>nul
)

echo Sindri updated successfully.
