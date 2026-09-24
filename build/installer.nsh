; Zajuna App: datos locales (%LOCALAPPDATA%\ZajunaApp) en instalación y
; desinstalación. El core Go guarda ahí la base SQLite, evidencias, reportes,
; copias y config.json; electron-builder no toca esa carpeta, por eso el
; checklist, las evidencias y los trabajos sobrevivían a una reinstalación.
;
; Regla de producto: una instalación nueva o una desinstalación manual borra
; los datos locales. Una actualización (${isUpdated}) los conserva: el
; checklist, las evidencias y las copias siguen en %LOCALAPPDATA%\ZajunaApp.
;
; Verificado contra las plantillas de app-builder-lib 26.15.3. makensis corre
; con -WX: no declarar Var/Function que no se usen.

; ---- Primera instalación: empezar desde cero ------------------------------
; En una actualización no tocamos los datos. En una instalación nueva, si
; quedó una carpeta de una desinstalación incompleta, pedimos al core un
; reset completo en el primer arranque.
!macro customInstall
  ${ifNot} ${isUpdated}
    Push $R0
    ${if} $installMode == "all"
      SetShellVarContext current
    ${endif}
    ${if} ${FileExists} "$LOCALAPPDATA\ZajunaApp\*.*"
      ClearErrors
      FileOpen $R0 "$LOCALAPPDATA\ZajunaApp\.reset-pending" w
      ${ifNot} ${Errors}
        FileWrite $R0 "full"
        FileClose $R0
      ${endif}
    ${endif}
    ${if} $installMode == "all"
      SetShellVarContext all
    ${endif}
    Pop $R0
  ${endif}
!macroend

; ---- Desinstalación manual: borrar siempre los datos ----------------------
!macro customUnInstall
  ; ${isUpdated}: el instalador nuevo ejecuta el desinstalador viejo con
  ; --updated. En ese caso no borramos aquí; lo hace customInstall + core.
  ${ifNot} ${isUpdated}
    Push $R0
    ; CHECK_APP_RUNNING sin PowerShell solo termina "Zajuna App.exe":
    ; aseguramos que el core (abre zajuna.db) y su Chromium hayan terminado.
    nsExec::Exec `"$SYSDIR\cmd.exe" /C taskkill /F /T /FI "USERNAME eq %USERNAME%" /IM zajuna-core.exe`
    Pop $R0
    Sleep 500

    ${if} $installMode == "all"
      SetShellVarContext current
    ${endif}

    RMDir /r "$LOCALAPPDATA\ZajunaApp"
    RMDir /r "$APPDATA\zajuna-app"
    RMDir /r "$APPDATA\Zajuna App"
    RMDir /r "$LOCALAPPDATA\zajuna-app-updater"

    ${if} $installMode == "all"
      SetShellVarContext all
    ${endif}
    Pop $R0
  ${endif}
!macroend
