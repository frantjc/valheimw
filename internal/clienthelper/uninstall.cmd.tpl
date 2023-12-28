@echo off
setlocal

set "VALHEIM_DIR=%~dp0"

set `DIRS_TO_REMOVE=`
set `DIRS_TO_REMOVE=%DIRS_TO_REMOVE% "%VALHEIM_DIR%BepInEx\"`
set `DIRS_TO_REMOVE=%DIRS_TO_REMOVE% "%VALHEIM_DIR%doorstop_libs\"`

echo Removing Sindri files...

for %%I in (%DIRS_TO_REMOVE%) do (
    rd /s /q "%%I" 2>nul
)

set `FILES_TO_REMOVE=`
set `FILES_TO_REMOVE=%FILES_TO_REMOVE% "%VALHEIM_DIR%changelog.txt"`
set `FILES_TO_REMOVE=%FILES_TO_REMOVE% "%VALHEIM_DIR%doorstop_config.ini"`
set `FILES_TO_REMOVE=%FILES_TO_REMOVE% "%VALHEIM_DIR%start_game_bepinex.sh"`
set `FILES_TO_REMOVE=%FILES_TO_REMOVE% "%VALHEIM_DIR%start_server_bepinex.sh"`
set `FILES_TO_REMOVE=%FILES_TO_REMOVE% "%VALHEIM_DIR%winhttp.dll"`
set `FILES_TO_REMOVE=%FILES_TO_REMOVE% "%VALHEIM_DIR%__README_TXT_NAME__"`
set `FILES_TO_REMOVE=%FILES_TO_REMOVE% "%VALHEIM_DIR%__UNINSTALL_SH_NAME__"`
set `FILES_TO_REMOVE=%FILES_TO_REMOVE% "%VALHEIM_DIR%__UPDATE_SH_NAME__"`
set `FILES_TO_REMOVE=%FILES_TO_REMOVE% "%VALHEIM_DIR%__UPDATE_CMD_NAME__"`

for %%I in (%FILES_TO_REMOVE%) do (
    del "%%I" 2>nul
)

echo Sindri uninstalled successfully.

echo The next line can be ignored; it is caused by this script removing itself.

del "%VALHEIM_DIR%__UNINSTALL_CMD_NAME__" 2>nul
