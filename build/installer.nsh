; Zajuna App: datos locales (%LOCALAPPDATA%\ZajunaApp) en instalación y
; desinstalación. El core Go guarda ahí la base SQLite, evidencias, reportes,
; copias y config.json; electron-builder no toca esa carpeta.
;
; Regla de producto: los datos y las copias locales se conservan entre
; versiones, actualizaciones, reinstalaciones y desinstalaciones. El esquema se
; actualiza con las migraciones de SQLite al abrir la versión nueva. El único
; borrado es el explícito desde la app («Restablecer datos»), que conserva las
; copias de seguridad.
;
; Verificado contra las plantillas de app-builder-lib 26.15.3. makensis corre
; con -WX: no declarar Var/Function que no se usen.

; ---- Desinstalación manual: cerrar el core, conservar los datos -----------
!macro customUnInstall
  ; ${isUpdated}: el instalador nuevo ejecuta el desinstalador viejo con
  ; --updated; la app ya se cerró para la actualización.
  ${ifNot} ${isUpdated}
    Push $R0
    ; CHECK_APP_RUNNING sin PowerShell solo termina "Zajuna App.exe":
    ; aseguramos que el core (abre zajuna.db) y su Chromium hayan terminado
    ; antes de retirar sus binarios.
    nsExec::Exec `"$SYSDIR\cmd.exe" /C taskkill /F /T /FI "USERNAME eq %USERNAME%" /IM zajuna-core.exe`
    Pop $R0
    Sleep 500

    ${if} $installMode == "all"
      SetShellVarContext current
    ${endif}

    ; Solo cachés reconstruibles del launcher y del actualizador. Los datos
    ; del usuario ($LOCALAPPDATA\ZajunaApp) no se tocan.
    RMDir /r "$LOCALAPPDATA\zajuna-app-updater"

    ${if} $installMode == "all"
      SetShellVarContext all
    ${endif}
    Pop $R0
  ${endif}
!macroend
